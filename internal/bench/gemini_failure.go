package bench

import (
	"encoding/json"
	"strings"
	"unicode/utf8"
)

const geminiTurnLimitDiagnostic = "gemini execution: configured four-turn limit reached (exit status 53); native usage unavailable"

// Only the pinned CLI's complete, line-framed fatal envelopes are recognized.
// Any malformed/conflicting candidate makes the whole interpretation ambiguous.
// Stderr never supplies source, usage, or normalized backend/session identity.
func parseGeminiTurnLimit(stderr string) *GeminiFailureEvidence {
	if len(stderr) > maxAdapterBytes || !utf8.ValidString(stderr) {
		return nil
	}
	var found *GeminiFailureEvidence
	for len(stderr) > 0 {
		line, rest, _ := strings.Cut(stderr, "\n")
		start := strings.TrimSpace(line)
		start = strings.TrimPrefix(start, "[ERROR] ")
		if !strings.HasPrefix(start, "{") {
			stderr = rest
			continue
		}
		// Slice the original buffer: concatenating the remaining stream for each
		// repeated envelope would make processing quadratic.
		candidate := stderr[strings.Index(line, "{"):]
		dec := json.NewDecoder(strings.NewReader(candidate))
		var raw json.RawMessage
		if err := dec.Decode(&raw); err != nil || uniqueJSON(raw) != nil {
			return nil
		}
		trailing := candidate[dec.InputOffset():]
		tail, next, _ := strings.Cut(trailing, "\n")
		if strings.TrimSpace(tail) != "" {
			return nil
		}
		// encoding/json matches struct fields case-insensitively. Require exact
		// keys so aliases such as code/Code cannot overwrite one another.
		var root, detail map[string]json.RawMessage
		if json.Unmarshal(raw, &root) != nil || len(root) != 2 || root["session_id"] == nil || root["error"] == nil {
			return nil
		}
		if json.Unmarshal(root["error"], &detail) != nil || len(detail) != 3 || detail["type"] == nil || detail["message"] == nil || detail["code"] == nil {
			return nil
		}
		var envelope struct {
			SessionID string `json:"session_id"`
			Error     struct {
				Type    string `json:"type"`
				Message string `json:"message"`
				Code    int    `json:"code"`
			} `json:"error"`
		}
		if strictJSON(raw, &envelope) != nil || strings.TrimSpace(envelope.SessionID) == "" || strings.TrimSpace(envelope.Error.Message) == "" || envelope.Error.Type != "FatalTurnLimitedError" || envelope.Error.Code != 53 {
			return nil
		}
		evidence := GeminiFailureEvidence{SessionID: envelope.SessionID, Type: envelope.Error.Type, Message: envelope.Error.Message, ExitCode: envelope.Error.Code}
		if found != nil && *found != evidence {
			return nil
		}
		found = &evidence
		stderr = next
	}
	return found
}
