package bench

import (
	"context"
	"encoding/json"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const mercuryTestUsage = `{"prompt_tokens":20,"completion_tokens":5,"total_tokens":25,"prompt_tokens_details":{"cached_tokens":3},"completion_tokens_details":{"reasoning_tokens":2}}`

func mercuryTestBody(usage string) string {
	return "data: {\"model\":\"mercury-3\",\"choices\":[]}\n\ndata: {\"model\":\"mercury-3\",\"usage\":" + usage + "}\n\ndata: [DONE]\n\n"
}

func mercuryTestJournal(t *testing.T, bodies ...string) string {
	t.Helper()
	var out strings.Builder
	for i, body := range bodies {
		records := []any{
			map[string]any{"type": "intent", "id": i + 1, "model": "mercury-3", "reasoning_effort": "low", "max_tokens": 8192, "stream": true, "include_usage": true, "retry_count": 0},
			map[string]any{"type": "response", "id": i + 1, "status": 200, "eof": true, "body": body},
		}
		for _, r := range records {
			b, err := json.Marshal(r)
			if err != nil {
				t.Fatal(err)
			}
			out.Write(b)
			out.WriteByte('\n')
		}
	}
	return out.String()
}

func mercuryTestEnvelope(t *testing.T, capture, failure string) string {
	t.Helper()
	// Deliberately invalid bridge usage must not contaminate provider accounting.
	native := `{"type":"result","subtype":"success","is_error":false,"session_id":"native-session","structured_output":{"source":"package answer\n"},"usage":{"input_tokens":99999},"modelUsage":{"fabricated":{}},"total_cost_usd":-1}`
	b, err := json.Marshal(mercuryOutput{Claude: native, Capture: capture, Error: failure})
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: Retained independently graded artifacts
func TestMercuryProviderAccounting(t *testing.T) {
	journal := mercuryTestJournal(t, mercuryTestBody(mercuryTestUsage), mercuryTestBody(mercuryTestUsage))
	r, err := parseMercuryOutput(mercuryTestEnvelope(t, journal, ""))
	if err != nil {
		t.Fatal(err)
	}
	if r.Usage == nil || r.Usage.InputTokens != 40 || r.Usage.OutputTokens != 10 || r.Usage.CachedInputTokens == nil || *r.Usage.CachedInputTokens != 6 || r.Usage.ReasoningOutputTokens == nil || *r.Usage.ReasoningOutputTokens != 4 {
		t.Fatalf("provider usage must sum primary buckets and preserve subsets: %+v", r.Usage)
	}
	if r.Source != "package answer\n" || r.Native.ClaudeReportedUsage != nil || r.Native.EventCount != 2 || r.Native.UsageStatus != "reported" || r.Native.Terminal != "completed" || len(r.Native.ReportedModels) != 1 || r.Native.ReportedModels[0] != "mercury-3" {
		t.Fatalf("provider identity/native result mismatch: %+v", r)
	}
	for _, usage := range []string{`{"prompt_tokens":0,"completion_tokens":0}`, `{"prompt_tokens":0,"completion_tokens":0,"prompt_tokens_details":{"cached_tokens":0},"completion_tokens_details":{"reasoning_tokens":0}}`} {
		r, err = parseMercuryOutput(mercuryTestEnvelope(t, mercuryTestJournal(t, mercuryTestBody(usage)), ""))
		if err != nil || r.Usage == nil || r.Usage.InputTokens != 0 || r.Usage.OutputTokens != 0 {
			t.Fatalf("explicit zero is valid: %+v %v", r, err)
		}
	}
	r, err = parseMercuryOutput(mercuryTestEnvelope(t, mercuryTestJournal(t, mercuryTestBody(mercuryTestUsage), mercuryTestBody(`{"prompt_tokens":1,"completion_tokens":1}`)), ""))
	if err != nil || r.Usage.CachedInputTokens != nil || r.Usage.ReasoningOutputTokens != nil {
		t.Fatalf("missing subsets must stay unknown: %+v %v", r, err)
	}
}

// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: Controlled bounded experiments
func TestMercuryUsageRefusals(t *testing.T) {
	for name, usage := range map[string]string{
		"missing": "null", "partial": `{"prompt_tokens":20}`, "negative": `{"prompt_tokens":-1,"completion_tokens":5}`,
		"fraction": `{"prompt_tokens":1.5,"completion_tokens":5}`, "boolean": `{"prompt_tokens":true,"completion_tokens":5}`,
		"overflow":  `{"prompt_tokens":1000000000001,"completion_tokens":5}`,
		"duplicate": `{"prompt_tokens":20,"prompt_tokens":21,"completion_tokens":5}`,
		"cache":     `{"prompt_tokens":20,"completion_tokens":5,"prompt_tokens_details":{"cached_tokens":21}}`,
		"reasoning": `{"prompt_tokens":20,"completion_tokens":5,"completion_tokens_details":{"reasoning_tokens":6}}`,
		"total":     `{"prompt_tokens":20,"completion_tokens":5,"total_tokens":99}`,
	} {
		t.Run(name, func(t *testing.T) {
			r, err := parseMercuryOutput(mercuryTestEnvelope(t, mercuryTestJournal(t, mercuryTestBody(usage)), ""))
			if err == nil || r.Usage != nil {
				t.Fatalf("invalid provider usage accepted: %+v %v", r, err)
			}
		})
	}
}

// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: Controlled bounded experiments
func TestMercuryCaptureRefusals(t *testing.T) {
	good := mercuryTestJournal(t, mercuryTestBody(mercuryTestUsage))
	lines := strings.SplitAfter(good, "\n")
	for name, capture := range map[string]string{
		"empty": "", "truncated": strings.TrimSuffix(good, "\n"), "intent-only": lines[0], "response-only": lines[1],
		"duplicate-intent": lines[0] + good, "duplicate-response": good + lines[1], "out-of-order": lines[1] + lines[0],
		"model": strings.ReplaceAll(good, "mercury-3", "other"), "effort": strings.ReplaceAll(good, "low", "high"),
		"budget": strings.ReplaceAll(good, "8192", "8193"), "retry": strings.ReplaceAll(good, `"retry_count":0`, `"retry_count":1`),
		"usage-option": strings.ReplaceAll(good, `"include_usage":true`, `"include_usage":false`),
		"nonstream":    strings.ReplaceAll(good, `"stream":true`, `"stream":false`),
		"eof":          strings.ReplaceAll(good, `"eof":true`, `"eof":false`), "http": strings.ReplaceAll(good, `"status":200`, `"status":429`),
		"capture-error":  good + `{"type":"error","error":"capture_failed"}` + "\n",
		"unknown-record": `{"type":"other","id":1}` + "\n", "foreign-id": strings.ReplaceAll(good, `"id":1`, `"id":5`),
	} {
		t.Run(name, func(t *testing.T) {
			r, err := parseMercuryOutput(mercuryTestEnvelope(t, capture, ""))
			if err == nil || r.Usage != nil {
				t.Fatalf("invalid capture accepted: %+v %v", r, err)
			}
		})
	}
	r, err := parseMercuryOutput(mercuryTestEnvelope(t, good, "bridge_timeout"))
	if err == nil || r.Usage != nil {
		t.Fatalf("runner failure must invalidate complete-looking capture: %+v %v", r, err)
	}
	for n := 1; n <= 5; n++ {
		bodies := make([]string, n)
		for i := range bodies {
			bodies[i] = mercuryTestBody(mercuryTestUsage)
		}
		r, err := parseMercuryOutput(mercuryTestEnvelope(t, mercuryTestJournal(t, bodies...), ""))
		if n <= 4 {
			if err != nil || r.Usage.InputTokens != int64(n*20) {
				t.Fatalf("bounded inventory %d: %+v %v", n, r, err)
			}
		} else if err == nil || r.Usage != nil {
			t.Fatal("fifth exchange accepted")
		}
	}
}

// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: Retained independently graded artifacts
func TestMercurySSECompletion(t *testing.T) {
	good := mercuryTestBody(mercuryTestUsage)
	for name, body := range map[string]string{
		"missing-done": strings.ReplaceAll(good, "data: [DONE]\n\n", ""), "after-done": good + "data: {}\n\n", "truncated-frame": strings.TrimSuffix(good, "\n"),
		"repeated-usage": strings.Replace(good, "data: [DONE]", "data: {\"usage\":"+mercuryTestUsage+"}\n\ndata: [DONE]", 1),
		"wrong-model":    strings.ReplaceAll(good, "mercury-3", "mercury-2"), "missing-model": strings.ReplaceAll(good, `"model":"mercury-3",`, ""),
		"provider-error": strings.Replace(good, `"choices":[]`, `"error":{"message":"failure"}`, 1), "invalid-json": strings.Replace(good, `"choices":[]`, `"choices":`, 1),
		"invalid-utf8": "data: \xff\n\n", "unsupported-field": "event: chunk\n\n" + good,
	} {
		t.Run(name, func(t *testing.T) {
			r, err := parseMercuryOutput(mercuryTestEnvelope(t, mercuryTestJournal(t, body), ""))
			if err == nil || r.Usage != nil {
				t.Fatalf("incomplete SSE accepted: %+v %v", r, err)
			}
		})
	}
	for _, body := range []string{good, strings.ReplaceAll(good, "\n", "\r\n"), ": keepalive\n\n" + good, strings.Replace(good, `"usage":`, `
`+`data: "usage":`, 1)} {
		if _, err := parseMercurySSE(body); err != nil {
			t.Fatalf("valid framing rejected: %v", err)
		}
	}
}

// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: Controlled bounded experiments
func TestMercurySizeAndNativeFailure(t *testing.T) {
	good := mercuryTestJournal(t, mercuryTestBody(mercuryTestUsage))
	envelope := mercuryTestEnvelope(t, good, "")
	for name, raw := range map[string]string{
		"native-failure": strings.Replace(envelope, `\"is_error\":false`, `\"is_error\":true`, 1),
		"native-invalid": `{"claude":"{","capture":""}`, "envelope-invalid": "{", "envelope-duplicate": `{"claude":"{}","claude":"{}","capture":""}`,
		"capture-limit":  mercuryTestEnvelope(t, strings.Repeat(" ", maxAdapterBytes)+"\n", ""),
		"envelope-limit": strings.Repeat(" ", 4*maxAdapterBytes+1),
	} {
		t.Run(name, func(t *testing.T) {
			r, err := parseMercuryOutput(raw)
			if err == nil || r.Usage != nil {
				t.Fatalf("invalid output accepted: %+v %v", r, err)
			}
		})
	}
	// Reach exact capture limit using an SSE comment; the next byte must refuse.
	padding := maxAdapterBytes - len(good) - 4
	body := ": " + strings.Repeat("x", padding) + "\n\n" + mercuryTestBody(mercuryTestUsage)
	atLimit := mercuryTestJournal(t, body)
	// Four newline bytes escape to eight in JSON, so trim the measured excess.
	body = strings.Replace(body, strings.Repeat("x", padding), strings.Repeat("x", padding-(len(atLimit)-maxAdapterBytes)), 1)
	atLimit = mercuryTestJournal(t, body)
	if len(atLimit) != maxAdapterBytes {
		t.Fatal("test must hit exact journal boundary")
	}
	if _, _, err := parseMercuryCapture(atLimit); err != nil {
		t.Fatalf("exact journal limit refused: %v", err)
	}
	if _, _, err := parseMercuryCapture(atLimit + "\n"); err == nil {
		t.Fatal("journal byte limit ignored")
	}
}

// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: Controlled bounded experiments
func TestMercuryHookContract(t *testing.T) {
	python, err := exec.LookPath("python3")
	if err != nil {
		t.Skip("Python required only for the optional LiteLLM bridge hook")
	}
	hook, err := filepath.Abs("mercury_hook.py")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	// Stub only vendor interfaces: this proves hook control/recording behavior,
	// not compatibility of an installed LiteLLM/OpenAI/HTTPX version.
	script := `
import asyncio, importlib.util, json, os, sys, types
sys.dont_write_bytecode = True
from pathlib import Path
root=Path(sys.argv[2])
os.environ.update(MERCURY_CAPTURE_PATH=str(root/"good.jsonl"),MERCURY_API_BASE="http://127.0.0.1:1234/v1",INCEPTION_API_KEY="test-key")
httpx=types.ModuleType("httpx")
httpx.AsyncByteStream=object
httpx.AsyncClient=lambda **kw: types.SimpleNamespace(**kw)
openai=types.ModuleType("openai")
openai.AsyncOpenAI=lambda **kw: types.SimpleNamespace(**kw)
custom=types.ModuleType("litellm.integrations.custom_logger")
custom.CustomLogger=object
sys.modules.update(httpx=httpx,openai=openai,litellm=types.ModuleType("litellm"))
sys.modules["litellm.integrations"]=types.ModuleType("litellm.integrations")
sys.modules["litellm.integrations.custom_logger"]=custom
spec=importlib.util.spec_from_file_location("hook",sys.argv[1]); hook=importlib.util.module_from_spec(spec);spec.loader.exec_module(hook)
class Stream:
    async def __aiter__(self):
        yield b'data: {}\n\n'
        yield b'data: [DONE]\n\n'
    async def aclose(self): pass
class Response:
    def __init__(self,request):
        self.headers={}; self.request=request; self.status_code=200; self.stream=Stream()
    async def aread(self):
        return b''.join([chunk async for chunk in self.stream])
def request():
    return types.SimpleNamespace(method="POST",url="http://127.0.0.1:1234/v1/chat/completions",extensions={},headers={"x-stainless-retry-count":"0"},content=json.dumps(dict(model="mercury-3",reasoning_effort="low",max_tokens=8192,stream=True,stream_options={"include_usage":True})).encode())
async def must_fail(action):
    try: await action
    except Exception: return
    raise AssertionError("capture boundary did not fail")
async def main():
    capture=hook.capture
    kwargs=await capture.async_pre_call_deployment_hook({"allowed_openai_params":["existing"]},"acompletion")
    assert kwargs["allowed_openai_params"]==["existing","max_retries"] and kwargs["max_retries"]==0
    assert kwargs["client"].http_client.trust_env is False and kwargs["client"].http_client.follow_redirects is False
    for i in range(4):
        req=request(); await capture.request(req); await capture.response(Response(req))
    records=[json.loads(line) for line in (root/"good.jsonl").read_text().splitlines()]
    assert len(records)==8 and records[0]["include_usage"] is True and records[1]["eof"] is True
    assert records[1]["body"]=='data: {}\n\ndata: [DONE]\n\n' and records[-1]["id"]==4
    await must_fail(capture.request(request()))
    assert capture.failed
    for name in ("retry","budget","missing_usage","encoded","oversize","journal","bad_allowed"):
        os.environ["MERCURY_CAPTURE_PATH"]=str(root/(name+".jsonl"))
        c=hook.Capture(); await c.async_pre_call_deployment_hook({},"acompletion")
        req=request()
        if name=="bad_allowed":
            await must_fail(c.async_pre_call_deployment_hook({"allowed_openai_params":False},"acompletion"))
        elif name=="retry":
            c.client.max_retries=2; await must_fail(c.request(req))
        elif name in ("budget","missing_usage"):
            data=json.loads(req.content)
            if name=="budget": data["max_tokens"]=8193
            else: data["stream_options"]={}
            req.content=json.dumps(data).encode(); await must_fail(c.request(req))
        else:
            await c.request(req); response=Response(req)
            if name=="encoded": response.headers["content-encoding"]="gzip"
            if name=="oversize": c.body_bytes=hook.LIMIT
            if name=="journal": c.written=hook.LIMIT
            await must_fail(c.response(response))
        assert c.failed, name
asyncio.run(main())
`
	cmd := exec.CommandContext(ctx, python, "-I", "-c", script, hook, t.TempDir())
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("hook contract: %v\n%s", err, output)
	}
}
