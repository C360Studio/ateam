// Package mcpserver implements the low-tax (<300 token) stdio MCP server for ateam.
package mcpserver

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/c360studio/ateam/internal/hook"
	"github.com/c360studio/ateam/internal/state"
	"github.com/c360studio/ateam/internal/taskrunner"
)

// MaxSchemaTokens is the strict upper bound on total MCP tool schema tokens (~4 chars/token).
const MaxSchemaTokens = 300

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type rpcResponse struct {
	JSONRPC string    `json:"jsonrpc"`
	ID      any       `json:"id,omitempty"`
	Result  any       `json:"result,omitempty"`
	Error   *rpcError `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// ToolDef represents a compact MCP tool definition.
type ToolDef struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
}

// Tools returns the 2 multiplexed tools kept strictly under 300 tokens total.
func Tools() []ToolDef {
	return []ToolDef{
		{
			Name:        "ateam_state",
			Description: "Reconcile pickup/handoff, OpenSpec queue, shortcut debt, board, or Ponytail mode.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"action": map[string]any{
						"type": "string",
						"enum": []string{"pickup", "handoff", "debt", "mode", "spec_queue", "spec_check", "board"},
					},
					"mode":          map[string]any{"type": "string"},
					"goal":          map[string]any{"type": "string"},
					"addresses":     map[string]any{"type": "string"},
					"constraints":   map[string]any{"type": "string"},
					"done_when":     map[string]any{"type": "string"},
					"checks_to_run": map[string]any{"type": "string"},
				},
				"required": []string{"action"},
			},
		},
		{
			Name:        "ateam_task",
			Description: "List or run a standardized Taskfile.yml target.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"target": map[string]any{"type": "string"},
					"args":   map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
				},
			},
		},
	}
}

// EstimateSchemaTokens estimates the token count of the serialized tools/list payload (~4 bytes/token).
func EstimateSchemaTokens() (int, int, error) {
	data, err := json.Marshal(Tools())
	if err != nil {
		return 0, 0, err
	}
	bytesLen := len(data)
	return (bytesLen + 3) / 4, bytesLen, nil
}

// Serve runs the JSON-RPC 2.0 MCP server over stdin and stdout.
func Serve(ctx context.Context, root string, in io.Reader, out io.Writer) error {
	scanner := bufio.NewScanner(in)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	enc := json.NewEncoder(out)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var req rpcRequest
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			_ = enc.Encode(rpcResponse{
				JSONRPC: "2.0",
				Error:   &rpcError{Code: -32700, Message: "parse error"},
			})
			continue
		}
		if req.ID == nil && strings.HasPrefix(req.Method, "notifications/") {
			continue
		}
		resp := handleRPC(ctx, root, req)
		if err := enc.Encode(resp); err != nil {
			return err
		}
	}
	return scanner.Err()
}

func handleRPC(ctx context.Context, root string, req rpcRequest) rpcResponse {
	switch req.Method {
	case "initialize":
		return rpcResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: map[string]any{
				"protocolVersion": "2024-11-05",
				"serverInfo":      map[string]any{"name": "ateam", "version": "0.1.0"},
				"capabilities": map[string]any{
					"tools":     map[string]any{},
					"prompts":   map[string]any{},
					"resources": map[string]any{},
				},
			},
		}
	case "ping":
		return rpcResponse{JSONRPC: "2.0", ID: req.ID, Result: map[string]any{}}
	case "tools/list":
		return rpcResponse{JSONRPC: "2.0", ID: req.ID, Result: map[string]any{"tools": Tools()}}
	case "tools/call":
		return handleToolCall(ctx, root, req)
	case "prompts/list":
		return rpcResponse{JSONRPC: "2.0", ID: req.ID, Result: map[string]any{"prompts": listPrompts()}}
	case "prompts/get":
		return handlePromptGet(root, req)
	case "resources/list":
		return rpcResponse{JSONRPC: "2.0", ID: req.ID, Result: map[string]any{"resources": listResources()}}
	case "resources/read":
		return handleResourceRead(ctx, root, req)
	default:
		return rpcResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   &rpcError{Code: -32601, Message: "method not found: " + req.Method},
		}
	}
}

func handleToolCall(ctx context.Context, root string, req rpcRequest) rpcResponse {
	var call struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}
	if err := json.Unmarshal(req.Params, &call); err != nil {
		return rpcErrorResp(req.ID, -32602, "invalid tool call params")
	}

	switch call.Name {
	case "ateam_state":
		text, isErr := execStateTool(ctx, root, call.Arguments)
		return toolTextResp(req.ID, text, isErr)
	case "ateam_task":
		text, isErr := execTaskTool(ctx, root, call.Arguments)
		return toolTextResp(req.ID, text, isErr)
	default:
		return rpcErrorResp(req.ID, -32602, "unknown tool: "+call.Name)
	}
}

func execStateTool(ctx context.Context, root string, rawArgs json.RawMessage) (string, bool) {
	var args struct {
		Action      string `json:"action"`
		Mode        string `json:"mode"`
		Goal        string `json:"goal"`
		Addresses   string `json:"addresses"`
		Constraints string `json:"constraints"`
		DoneWhen    string `json:"done_when"`
		ChecksToRun string `json:"checks_to_run"`
	}
	_ = json.Unmarshal(rawArgs, &args)

	st, err := state.Refresh(ctx, root, false)
	if err != nil {
		return err.Error(), true
	}

	switch args.Action {
	case "pickup":
		return state.BuildPickupSummary(st), false
	case "handoff":
		h := state.Handoff{
			Goal:          args.Goal,
			Addresses:     args.Addresses,
			Constraints:   args.Constraints,
			DoneWhen:      args.DoneWhen,
			ChecksToRun:   args.ChecksToRun,
			Branch:        st.Git.Branch,
			CheckpointSHA: st.Git.HeadSHA,
			Dirty:         st.Git.Dirty,
			UpdatedAt:     time.Now().UTC(),
		}
		if h.Addresses == "" {
			h.Addresses = fmt.Sprintf("branch %s, SHA %s", st.Git.Branch, st.Git.HeadSHA)
		}
		if _, err := state.Update(root, func(cur *state.State) error {
			cur.LastHandoff = &h
			return nil
		}); err != nil {
			return err.Error(), true
		}
		return state.FormatFiveFieldPrompt(h), false
	case "debt":
		data, _ := json.MarshalIndent(st.Shortcuts, "", "  ")
		return string(data), false
	case "mode":
		if args.Mode != "" {
			m, ok := state.NormalizeMode(args.Mode)
			if !ok {
				return "invalid mode: " + args.Mode, true
			}
			updated, err := state.Update(root, func(cur *state.State) error {
				cur.Mode = m
				return nil
			})
			if err != nil {
				return err.Error(), true
			}
			st = updated
		}
		return fmt.Sprintf("active mode: %s\n%s", st.Mode, hook.LadderPrompt(st.Mode)), false
	case "spec_queue":
		out, _ := state.FormatOpenSpecQueue(st.OpenSpecQueue)
		return out, false
	case "spec_check":
		out, code, err := state.CheckOpenSpec(ctx, root)
		if err != nil || code != 0 {
			return fmt.Sprintf("%s\n%v", out, err), true
		}
		return out, false
	case "board":
		data, _ := json.MarshalIndent(st.Cards, "", "  ")
		return string(data), false
	default:
		return "unsupported action: " + args.Action, true
	}
}

func execTaskTool(ctx context.Context, root string, rawArgs json.RawMessage) (string, bool) {
	var args struct {
		Target string   `json:"target"`
		Args   []string `json:"args"`
	}
	_ = json.Unmarshal(rawArgs, &args)
	if strings.TrimSpace(args.Target) == "" || args.Target == "list" {
		targets, err := taskrunner.ListTargets(root)
		if err != nil {
			return err.Error(), true
		}
		data, _ := json.MarshalIndent(targets, "", "  ")
		return string(data), false
	}
	out, err := taskrunner.Run(ctx, root, args.Target, args.Args)
	if err != nil {
		return err.Error(), true
	}
	return out, false
}

func listPrompts() []map[string]any {
	return []map[string]any{
		{"name": "pickup", "description": "Reconcile live git/worktree, GitHub/Gitea, OpenSpec, and handoff state"},
		{"name": "handoff", "description": "Checkpoint current work and emit the 5-field pickup prompt"},
		{"name": "preflight", "description": "Select and run Taskfile verification gates for the current diff"},
		{"name": "ponytail", "description": "Apply the Ponytail 6-step decision ladder (lite, full, ultra, off)"},
	}
}

func handlePromptGet(root string, req rpcRequest) rpcResponse {
	var p struct {
		Name string `json:"name"`
	}
	_ = json.Unmarshal(req.Params, &p)
	skillPath := filepath.Join(root, ".agents", "skills", p.Name, "SKILL.md")
	data, err := os.ReadFile(skillPath)
	if err != nil {
		return rpcErrorResp(req.ID, -32602, "unknown prompt: "+p.Name)
	}
	return rpcResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: map[string]any{
			"description": "Canonical skill " + p.Name,
			"messages": []map[string]any{
				{
					"role":    "user",
					"content": map[string]any{"type": "text", "text": string(data)},
				},
			},
		},
	}
}

func listResources() []map[string]any {
	return []map[string]any{
		{"uri": "ateam://state/pickup", "name": "Pickup Summary", "mimeType": "text/plain"},
		{"uri": "ateam://state/board", "name": "PM Board Cards (GitHub/Gitea/Local)", "mimeType": "application/json"},
		{"uri": "ateam://state/shortcuts", "name": "Ponytail Shortcut Debt", "mimeType": "application/json"},
		{"uri": "ateam://state/openspec", "name": "OpenSpec In-Flight Queue", "mimeType": "text/plain"},
	}
}

func handleResourceRead(ctx context.Context, root string, req rpcRequest) rpcResponse {
	var r struct {
		URI string `json:"uri"`
	}
	_ = json.Unmarshal(req.Params, &r)
	st, err := state.Refresh(ctx, root, false)
	if err != nil {
		return rpcErrorResp(req.ID, -32603, err.Error())
	}
	var text string
	switch r.URI {
	case "ateam://state/pickup":
		text = state.BuildPickupSummary(st)
	case "ateam://state/board":
		b, _ := json.MarshalIndent(st.Cards, "", "  ")
		text = string(b)
	case "ateam://state/shortcuts":
		b, _ := json.MarshalIndent(st.Shortcuts, "", "  ")
		text = string(b)
	case "ateam://state/openspec":
		text, _ = state.FormatOpenSpecQueue(st.OpenSpecQueue)
	default:
		return rpcErrorResp(req.ID, -32602, "unknown resource URI: "+r.URI)
	}
	return rpcResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: map[string]any{
			"contents": []map[string]any{
				{"uri": r.URI, "text": text},
			},
		},
	}
}

func toolTextResp(id any, text string, isError bool) rpcResponse {
	return rpcResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: map[string]any{
			"content": []map[string]any{{"type": "text", "text": text}},
			"isError": isError,
		},
	}
}

func rpcErrorResp(id any, code int, msg string) rpcResponse {
	return rpcResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error:   &rpcError{Code: code, Message: msg},
	}
}
