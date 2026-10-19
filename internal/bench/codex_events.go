package bench

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
)

type codexEvent struct {
	Type     string          `json:"type"`
	ThreadID string          `json:"thread_id"`
	Item     *codexItem      `json:"item"`
	Usage    json.RawMessage `json:"usage"`
}

type codexItem struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	Text string `json:"text"`
}

type codexStream struct {
	response AdapterResponse
	started  bool
	errored  bool
	message  string
	observed map[string]bool
	complete map[string]bool
}

// parseCodexEvents also returns evidence for incomplete and failed native streams.
// Native fields outside the consumed schema remain in stdout; new event types fail closed.
func parseCodexEvents(stdout string) (AdapterResponse, error) {
	s := codexStream{response: AdapterResponse{Version: 1, Native: &CodexEvidence{Terminal: "incomplete", UsageStatus: "missing"}}, observed: map[string]bool{}, complete: map[string]bool{}}
	if len(stdout) > maxAdapterBytes {
		return s.response, fmt.Errorf("codex events exceed 1 MiB")
	}
	var streamErr error
	for i, line := range strings.Split(stdout, "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		if err := s.consume([]byte(line)); err != nil {
			streamErr = fmt.Errorf("codex event line %d: %w", i+1, err)
			// Ambiguous histories cannot provide accountable totals.
			s.response.Usage = nil
			s.response.Native.UsageStatus = "invalid"
			break
		}
	}
	for typ := range s.observed {
		s.response.Native.ObservedItemTypes = append(s.response.Native.ObservedItemTypes, typ)
	}
	for typ := range s.complete {
		s.response.Native.CompletedItemTypes = append(s.response.Native.CompletedItemTypes, typ)
	}
	sort.Strings(s.response.Native.ObservedItemTypes)
	sort.Strings(s.response.Native.CompletedItemTypes)
	if streamErr != nil {
		return s.response, streamErr
	}
	if s.errored || s.response.Native.Terminal != "completed" {
		return s.response, fmt.Errorf("codex stream did not complete successfully (%s)", s.response.Native.Terminal)
	}
	var final struct {
		Source string `json:"source"`
	}
	if err := uniqueJSON([]byte(s.message)); err != nil {
		return s.response, fmt.Errorf("codex final message: %w", err)
	}
	if err := strictJSON([]byte(s.message), &final); err != nil {
		return s.response, fmt.Errorf("codex final message: %w", err)
	}
	if strings.TrimSpace(final.Source) == "" {
		return s.response, fmt.Errorf("codex final message requires nonempty source")
	}
	s.response.Source = final.Source
	return s.response, nil
}

func (s *codexStream) consume(line []byte) error {
	if err := uniqueJSON(line); err != nil {
		return err
	}
	var e codexEvent
	if err := json.Unmarshal(line, &e); err != nil {
		return err
	}
	n := s.response.Native
	if n.Terminal == "completed" || n.Terminal == "failed" {
		return fmt.Errorf("event after terminal outcome")
	}
	n.EventCount++
	switch e.Type {
	case "thread.started":
		if n.SessionID != "" || s.started || strings.TrimSpace(e.ThreadID) == "" || s.errored {
			return fmt.Errorf("invalid thread start")
		}
		n.SessionID = e.ThreadID
	case "turn.started":
		if n.SessionID == "" || s.started || s.errored {
			return fmt.Errorf("invalid turn start")
		}
		s.started = true
	case "item.started", "item.updated", "item.completed":
		if !s.started || s.errored || e.Item == nil || e.Item.ID == "" || e.Item.Type == "" {
			return fmt.Errorf("invalid item event")
		}
		s.observed[e.Item.Type] = true
		if e.Type == "item.completed" {
			s.complete[e.Item.Type] = true
			if e.Item.Type == "agent_message" {
				s.message = e.Item.Text
			}
		}
	case "turn.completed", "turn.failed":
		if !s.started || (s.errored && e.Type == "turn.completed") {
			return fmt.Errorf("invalid turn terminal")
		}
		n.Terminal = strings.TrimPrefix(e.Type, "turn.")
		return s.consumeUsage(e.Usage)
	case "error":
		s.errored = true
		n.Terminal = "error"
	default:
		return fmt.Errorf("unknown native event type %q", e.Type)
	}
	return nil
}

func (s *codexStream) consumeUsage(raw json.RawMessage) error {
	if len(raw) == 0 || bytes.Equal(raw, []byte("null")) {
		return nil
	}
	var reported CodexReportedUsage
	if err := strictJSON(raw, &reported); err != nil {
		return fmt.Errorf("native usage: %w", err)
	}
	for _, count := range []*int64{reported.InputTokens, reported.OutputTokens, reported.CachedInputTokens, reported.CacheWriteInputTokens, reported.ReasoningOutputTokens} {
		if count != nil && (*count < 0 || *count > 1e12) {
			return fmt.Errorf("invalid native token count")
		}
	}
	if reported.InputTokens != nil && reported.CachedInputTokens != nil && *reported.CachedInputTokens > *reported.InputTokens {
		return fmt.Errorf("cached input exceeds input")
	}
	if reported.OutputTokens != nil && reported.ReasoningOutputTokens != nil && *reported.ReasoningOutputTokens > *reported.OutputTokens {
		return fmt.Errorf("reasoning output exceeds output")
	}
	s.response.Native.ReportedUsage = &reported
	s.response.Native.UsageStatus = "partial"
	if reported.InputTokens != nil && reported.OutputTokens != nil {
		s.response.Usage = &Usage{InputTokens: *reported.InputTokens, OutputTokens: *reported.OutputTokens, CachedInputTokens: reported.CachedInputTokens, ReasoningOutputTokens: reported.ReasoningOutputTokens}
		s.response.Native.UsageStatus = "reported"
	}
	return nil
}

// uniqueJSON refuses duplicate object keys, trailing values and excessive nesting.
// Decoder's normal duplicate-key overwrite would hide contradictory usage or outcomes.
func uniqueJSON(data []byte) error {
	d := json.NewDecoder(bytes.NewReader(data))
	d.UseNumber()
	if err := uniqueJSONValue(d, 0); err != nil {
		return err
	}
	if _, err := d.Token(); err != io.EOF {
		return fmt.Errorf("expected one JSON value")
	}
	return nil
}

func uniqueJSONValue(d *json.Decoder, depth int) error {
	if depth > 64 {
		return fmt.Errorf("JSON nesting exceeds 64")
	}
	token, err := d.Token()
	if err != nil {
		return err
	}
	open, compound := token.(json.Delim)
	if !compound {
		return nil
	}
	seen := map[string]bool{}
	for d.More() {
		if open == '{' {
			key, err := d.Token()
			if err != nil {
				return err
			}
			name, ok := key.(string)
			if !ok || seen[name] {
				return fmt.Errorf("duplicate or invalid object key")
			}
			seen[name] = true
		}
		if err := uniqueJSONValue(d, depth+1); err != nil {
			return err
		}
	}
	_, err = d.Token()
	return err
}
