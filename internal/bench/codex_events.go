package bench

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

type codexRPC struct {
	ID     *int64          `json:"id"`
	Method string          `json:"method"`
	Params json.RawMessage `json:"params"`
	Result json.RawMessage `json:"result"`
	Error  json.RawMessage `json:"error"`
}

type codexThreadStart struct {
	Thread struct {
		ID        string `json:"id"`
		Ephemeral *bool  `json:"ephemeral"`
	} `json:"thread"`
	Model              *string   `json:"model"`
	ModelProvider      *string   `json:"modelProvider"`
	ReasoningEffort    *string   `json:"reasoningEffort"`
	ApprovalPolicy     *string   `json:"approvalPolicy"`
	InstructionSources *[]string `json:"instructionSources"`
	Sandbox            *struct {
		Type          string `json:"type"`
		NetworkAccess *bool  `json:"networkAccess"`
	} `json:"sandbox"`
}

func decodeCodexThreadStart(data []byte) (codexThreadStart, error) {
	var start codexThreadStart
	var rpc codexRPC
	if err := uniqueJSON(data); err != nil {
		return start, fmt.Errorf("codex thread/start JSON: %w", err)
	}
	if err := json.Unmarshal(data, &rpc); err != nil {
		return start, fmt.Errorf("codex thread/start envelope: %w", err)
	}
	if rpc.ID == nil || *rpc.ID != 2 || rpc.Method != "" || claudePresent(rpc.Error) || !claudePresent(rpc.Result) {
		return start, fmt.Errorf("codex requires successful thread/start response id 2")
	}
	if err := json.Unmarshal(rpc.Result, &start); err != nil {
		return start, fmt.Errorf("codex thread/start response: %w", err)
	}
	if strings.TrimSpace(start.Thread.ID) == "" || start.Thread.Ephemeral == nil || !*start.Thread.Ephemeral {
		return start, fmt.Errorf("codex requires a fresh ephemeral thread")
	}
	if start.ApprovalPolicy == nil || *start.ApprovalPolicy != "never" || start.Sandbox == nil || start.Sandbox.Type != "readOnly" || start.Sandbox.NetworkAccess == nil || *start.Sandbox.NetworkAccess || start.InstructionSources == nil || len(*start.InstructionSources) != 0 {
		return start, fmt.Errorf("codex native thread must deny approvals, sandbox writes/network and ambient instruction files")
	}
	return start, nil
}

// validateCodexThreadStart checks native resolution before the executor submits
// a paid turn. Requested settings are compared, never substituted into evidence.
func validateCodexThreadStart(data []byte, model, reasoning string) (string, error) {
	start, err := decodeCodexThreadStart(data)
	if err != nil {
		return "", err
	}
	if start.Model == nil || *start.Model != model || start.ModelProvider == nil || *start.ModelProvider != "benchmark_openai" || start.ReasoningEffort == nil || *start.ReasoningEffort != reasoning {
		return "", fmt.Errorf("codex native thread configuration does not match requested model, provider and reasoning")
	}
	return start.Thread.ID, nil
}

type codexItem struct {
	ID    string  `json:"id"`
	Type  string  `json:"type"`
	Text  *string `json:"text"`
	Phase *string `json:"phase"`
}

type codexTurn struct {
	ID     string          `json:"id"`
	Status string          `json:"status"`
	Error  json.RawMessage `json:"error"`
}

type codexNotification struct {
	ThreadID string          `json:"threadId"`
	TurnID   string          `json:"turnId"`
	ItemID   string          `json:"itemId"`
	Status   json.RawMessage `json:"status"`
	Thread   struct {
		ID string `json:"id"`
	} `json:"thread"`
	Turn       *codexTurn `json:"turn"`
	Item       *codexItem `json:"item"`
	TokenUsage *struct {
		Total *CodexTokens `json:"total"`
		Last  *CodexTokens `json:"last"`
	} `json:"tokenUsage"`
}

type codexHistory struct {
	response                      AdapterResponse
	initialized, threadAck        bool
	threadStarted, turnAck        bool
	turnStarted, completed        bool
	items                         map[string]string
	completedItems                map[string]bool
	observedTypes, completedTypes map[string]bool
	final                         *codexItem
	lastMessage                   *codexItem
	finalCount                    int
}

// parseCodexOutput reads app-server stdout only: responses to fixed request IDs
// 1/2/3 and native notifications. It never accepts client-side requested identity
// as reported model evidence. Schemas are generated from Codex CLI 0.160.1.
func parseCodexOutput(stdout string) (AdapterResponse, error) {
	h := newCodexHistory()
	if len(stdout) > maxAdapterBytes || strings.TrimSpace(stdout) == "" {
		return h.response, fmt.Errorf("codex output is empty or exceeds 1 MiB")
	}
	for _, line := range strings.Split(stdout, "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		if err := h.consume([]byte(line)); err != nil {
			return h.response, err
		}
	}
	return h.finish()
}

func newCodexHistory() *codexHistory {
	return &codexHistory{
		response: AdapterResponse{Version: 1, Native: &NativeEvidence{
			Terminal: "incomplete", UsageStatus: "missing", Codex: &CodexEvidence{},
		}},
		items: map[string]string{}, completedItems: map[string]bool{},
		observedTypes: map[string]bool{}, completedTypes: map[string]bool{},
	}
}

func (h *codexHistory) consume(data []byte) error {
	n := h.response.Native
	if err := uniqueJSON(data); err != nil {
		n.UsageStatus = "invalid"
		return fmt.Errorf("codex JSONL: %w", err)
	}
	var rpc codexRPC
	if err := json.Unmarshal(data, &rpc); err != nil {
		n.UsageStatus = "invalid"
		return fmt.Errorf("codex JSONL envelope: %w", err)
	}
	n.EventCount++
	if claudePresent(rpc.Error) {
		n.Terminal = "failed"
		return fmt.Errorf("codex RPC reported an error; see retained native output")
	}
	if rpc.ID != nil {
		if rpc.Method != "" {
			return fmt.Errorf("codex requested an unapproved client action")
		}
		return h.consumeResponse(rpc, data)
	}
	if rpc.Method == "" || !claudePresent(rpc.Params) || claudePresent(rpc.Result) {
		return fmt.Errorf("codex output requires a response or notification")
	}
	return h.consumeNotification(rpc)
}

func (h *codexHistory) consumeResponse(rpc codexRPC, data []byte) error {
	if !claudePresent(rpc.Result) {
		return fmt.Errorf("codex RPC response lacks a result")
	}
	n := h.response.Native
	switch *rpc.ID {
	case 1:
		var result struct {
			UserAgent string `json:"userAgent"`
		}
		if h.initialized || json.Unmarshal(rpc.Result, &result) != nil || strings.TrimSpace(result.UserAgent) == "" {
			return fmt.Errorf("codex initialize response is duplicate or invalid")
		}
		h.initialized = true
	case 2:
		if !h.initialized || h.threadAck {
			return fmt.Errorf("codex thread/start response is duplicate or out of order")
		}
		start, err := decodeCodexThreadStart(data)
		if err != nil {
			return err
		}
		if err := h.matchThread(start.Thread.ID); err != nil {
			return err
		}
		n.SessionID = start.Thread.ID
		n.Codex.IdentitySource = "cli-thread-start"
		n.Codex.Model, n.Codex.ModelProvider, n.Codex.ReasoningEffort = start.Model, start.ModelProvider, start.ReasoningEffort
		n.ResolvedModel, n.ResolvedReasoning = start.Model, start.ReasoningEffort
		h.threadAck = true
	case 3:
		var result struct {
			Turn *codexTurn `json:"turn"`
		}
		if !h.threadAck || h.turnAck || json.Unmarshal(rpc.Result, &result) != nil || result.Turn == nil || result.Turn.Status != "inProgress" || claudePresent(result.Turn.Error) {
			return fmt.Errorf("codex turn/start response is duplicate, failed or out of order")
		}
		if err := h.matchTurn(result.Turn.ID); err != nil {
			return err
		}
		h.turnAck = true
	default:
		return fmt.Errorf("codex returned an unexpected RPC response id")
	}
	return nil
}

func (h *codexHistory) matchThread(id string) error {
	if strings.TrimSpace(id) == "" || (h.response.Native.Codex.ThreadID != "" && h.response.Native.Codex.ThreadID != id) {
		return fmt.Errorf("codex output has a missing or mismatched thread id")
	}
	h.response.Native.Codex.ThreadID = id
	return nil
}

func (h *codexHistory) matchTurn(id string) error {
	if strings.TrimSpace(id) == "" || (h.response.Native.Codex.TurnID != "" && h.response.Native.Codex.TurnID != id) {
		return fmt.Errorf("codex output has a missing or mismatched turn id")
	}
	h.response.Native.Codex.TurnID = id
	return nil
}

func (h *codexHistory) consumeNotification(rpc codexRPC) error {
	var p codexNotification
	if err := json.Unmarshal(rpc.Params, &p); err != nil {
		return fmt.Errorf("codex notification: %w", err)
	}
	if !h.initialized {
		return fmt.Errorf("codex notification arrived before initialization")
	}
	if rpc.Method == "remoteControl/status/changed" {
		var status string
		if json.Unmarshal(p.Status, &status) != nil || status != "disabled" {
			return fmt.Errorf("codex remote control must remain disabled")
		}
		return nil
	}
	if rpc.Method == "thread/started" {
		if h.threadStarted || h.turnStarted {
			return fmt.Errorf("codex started more than one thread")
		}
		h.threadStarted = true
		return h.matchThread(p.Thread.ID)
	}
	if rpc.Method == "model/rerouted" || rpc.Method == "error" {
		h.response.Native.Terminal = "failed"
		return fmt.Errorf("codex reported a model reroute or execution error")
	}
	if err := h.matchThread(p.ThreadID); err != nil {
		return err
	}
	if rpc.Method == "thread/status/changed" {
		return nil
	}
	if rpc.Method == "thread/closed" {
		if !h.completed {
			return fmt.Errorf("codex thread closed before turn completion")
		}
		return nil
	}
	if !h.threadAck {
		return fmt.Errorf("codex turn event arrived before thread configuration")
	}
	if rpc.Method == "turn/started" {
		if h.turnStarted || p.Turn == nil || p.Turn.Status != "inProgress" || claudePresent(p.Turn.Error) {
			return fmt.Errorf("codex turn start is duplicate or invalid")
		}
		h.turnStarted = true
		return h.matchTurn(p.Turn.ID)
	}
	if !h.turnStarted || h.completed {
		return fmt.Errorf("codex turn event arrived outside the active turn")
	}
	if rpc.Method == "turn/completed" {
		if p.Turn == nil || h.matchTurn(p.Turn.ID) != nil {
			return fmt.Errorf("codex completion lacks the active turn")
		}
		h.completed = true
		if p.Turn.Status != "completed" || claudePresent(p.Turn.Error) {
			h.response.Native.Terminal = "failed"
			return fmt.Errorf("codex turn did not complete successfully")
		}
		return nil
	}
	if err := h.matchTurn(p.TurnID); err != nil {
		return err
	}
	switch rpc.Method {
	case "thread/tokenUsage/updated":
		return h.consumeUsage(p)
	case "item/started", "item/completed":
		return h.consumeItem(rpc.Method, p.Item)
	case "item/agentMessage/delta", "item/reasoning/summaryTextDelta", "item/reasoning/summaryPartAdded", "item/reasoning/textDelta":
		kind, ok := h.items[p.ItemID]
		if !ok || h.completedItems[p.ItemID] || (strings.HasPrefix(rpc.Method, "item/agentMessage/") && kind != "agentMessage") || (strings.HasPrefix(rpc.Method, "item/reasoning/") && kind != "reasoning") {
			return fmt.Errorf("codex delta does not belong to an active matching item")
		}
		return nil
	default:
		return fmt.Errorf("codex emitted an unsupported notification: %s", rpc.Method)
	}
}

func (h *codexHistory) consumeItem(method string, item *codexItem) error {
	if item == nil || strings.TrimSpace(item.ID) == "" {
		return fmt.Errorf("codex item requires an id")
	}
	if item.Type != "agentMessage" && item.Type != "reasoning" && item.Type != "userMessage" {
		return fmt.Errorf("codex emitted a tool, auxiliary agent or unsupported item")
	}
	if method == "item/started" {
		if _, found := h.items[item.ID]; found {
			return fmt.Errorf("codex item was started more than once")
		}
		h.items[item.ID] = item.Type
		h.observedTypes[item.Type] = true
		return nil
	}
	if kind, found := h.items[item.ID]; !found || kind != item.Type || h.completedItems[item.ID] {
		return fmt.Errorf("codex item completion has no unique matching start")
	}
	h.completedItems[item.ID], h.completedTypes[item.Type] = true, true
	if item.Type == "agentMessage" {
		h.lastMessage = item
		if item.Text == nil || (item.Phase != nil && *item.Phase != "commentary" && *item.Phase != "partial_answer" && *item.Phase != "final_answer") {
			return fmt.Errorf("codex agent message has invalid text or phase")
		}
		if item.Phase == nil || *item.Phase == "final_answer" {
			h.final, h.finalCount = item, h.finalCount+1
		}
	}
	return nil
}

func (h *codexHistory) consumeUsage(p codexNotification) error {
	n := h.response.Native
	if p.TokenUsage == nil {
		return fmt.Errorf("codex token update has no usage object")
	}
	previous := n.Codex.Total
	n.Codex.Total, n.Codex.Last = p.TokenUsage.Total, p.TokenUsage.Last
	n.UsageStatus = "partial"
	// A missing update must not erase the baseline used to detect a later
	// cumulative decrease. Fail at the first loss of any previously known count.
	current := codexCounts(n.Codex.Total)
	for i, known := range codexCounts(previous) {
		if known != nil && (len(current) == 0 || current[i] == nil) {
			n.UsageStatus = "invalid"
			return fmt.Errorf("codex cumulative token counters disappeared")
		}
	}
	for _, tokens := range []*CodexTokens{n.Codex.Total, n.Codex.Last} {
		if err := validateCodexTokens(tokens); err != nil {
			n.UsageStatus = "invalid"
			return err
		}
	}
	if err := codexCountsAtMost(previous, n.Codex.Total); err != nil {
		n.UsageStatus = "invalid"
		return fmt.Errorf("codex cumulative token counts decreased: %w", err)
	}
	if err := codexCountsAtMost(n.Codex.Last, n.Codex.Total); err != nil {
		n.UsageStatus = "invalid"
		return fmt.Errorf("codex last request exceeds cumulative usage: %w", err)
	}
	return nil
}

func codexCounts(tokens *CodexTokens) []*int64 {
	if tokens == nil {
		return nil
	}
	return []*int64{tokens.InputTokens, tokens.CachedInputTokens, tokens.CacheWriteInputTokens, tokens.OutputTokens, tokens.ReasoningOutputTokens, tokens.TotalTokens}
}

func codexCountsAtMost(before, after *CodexTokens) error {
	if before == nil || after == nil {
		return nil
	}
	for i, left := range codexCounts(before) {
		right := codexCounts(after)[i]
		if left != nil && right != nil && *left > *right {
			return fmt.Errorf("inconsistent native token counters")
		}
	}
	return nil
}

func validateCodexTokens(tokens *CodexTokens) error {
	for _, value := range codexCounts(tokens) {
		if value != nil && (*value < 0 || *value > 1e12) {
			return fmt.Errorf("invalid codex token count")
		}
	}
	if tokens == nil {
		return nil
	}
	if tokens.InputTokens != nil {
		for _, cache := range []*int64{tokens.CachedInputTokens, tokens.CacheWriteInputTokens} {
			if cache != nil && *cache > *tokens.InputTokens {
				return fmt.Errorf("codex cache tokens exceed input")
			}
		}
	}
	if tokens.OutputTokens != nil && tokens.ReasoningOutputTokens != nil && *tokens.ReasoningOutputTokens > *tokens.OutputTokens {
		return fmt.Errorf("codex reasoning tokens exceed output")
	}
	if tokens.InputTokens != nil && tokens.OutputTokens != nil && tokens.TotalTokens != nil && *tokens.InputTokens+*tokens.OutputTokens != *tokens.TotalTokens {
		return fmt.Errorf("codex total tokens disagree with input plus output")
	}
	return nil
}

func (h *codexHistory) finish() (AdapterResponse, error) {
	n := h.response.Native
	if !h.initialized || !h.threadAck || !h.threadStarted || !h.turnAck || !h.turnStarted || !h.completed || len(h.items) != len(h.completedItems) {
		return h.response, fmt.Errorf("codex transcript has an incomplete lifecycle")
	}
	n.Terminal = "completed"
	for kind := range h.observedTypes {
		n.ObservedItemTypes = append(n.ObservedItemTypes, kind)
	}
	for kind := range h.completedTypes {
		n.CompletedItemTypes = append(n.CompletedItemTypes, kind)
	}
	sort.Strings(n.ObservedItemTypes)
	sort.Strings(n.CompletedItemTypes)
	u := n.Codex.Total
	if u != nil && u.InputTokens != nil && u.CachedInputTokens != nil && u.OutputTokens != nil && u.ReasoningOutputTokens != nil && u.TotalTokens != nil {
		h.response.Usage = &Usage{InputTokens: *u.InputTokens, OutputTokens: *u.OutputTokens, CachedInputTokens: u.CachedInputTokens, ReasoningOutputTokens: u.ReasoningOutputTokens}
		n.UsageStatus = "reported"
	}
	if h.final == nil || h.finalCount != 1 || h.final != h.lastMessage {
		return h.response, fmt.Errorf("codex requires exactly one completed final agent message")
	}
	data := []byte(*h.final.Text)
	if err := uniqueJSON(data); err != nil {
		return h.response, fmt.Errorf("codex final source JSON: %w", err)
	}
	var source map[string]string
	if err := json.Unmarshal(data, &source); err != nil || len(source) != 1 || strings.TrimSpace(source["source"]) == "" {
		return h.response, fmt.Errorf("codex final source requires only nonempty source")
	}
	h.response.Source = source["source"]
	return h.response, nil
}
