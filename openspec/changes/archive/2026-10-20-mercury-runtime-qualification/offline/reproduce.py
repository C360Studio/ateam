# Research reproducer: macOS, synthetic provider, no production adapter.
import hashlib, http.client, http.server, json, os, pathlib, signal, socket, subprocess, sys, tempfile, threading, time, urllib.request
# Arguments: existing isolated root, exact pinned Claude binary, fresh output directory.
R = pathlib.Path(sys.argv[1]).resolve(strict=True)
CLI = pathlib.Path(sys.argv[2]).resolve(strict=True)
OUT = pathlib.Path(sys.argv[3]).resolve()
assert R.is_relative_to(pathlib.Path.home()), "root must be under the user home for the file-denial control"
assert OUT.is_relative_to(R) and OUT != R, "output must be a fresh child of the isolated root"
assert hashlib.sha256(CLI.read_bytes()).hexdigest() == "97a01e5bc74a199e67189435d0331ea3a24eac2e07db4b76d9148c5b0386138f", "Claude binary pin differs"
OUT.mkdir(parents=True, exist_ok=False)
bridge_log = pathlib.Path.cwd() / ".enjoy-logs" / (OUT.name + "-bridge.log")
bridge_log.parent.mkdir(exist_ok=True)
ledger = []
case_name = "startup"
class Stub(http.server.BaseHTTPRequestHandler):
    def log_message(self, *args): pass
    def do_POST(self):
        raw = self.rfile.read(int(self.headers.get("Content-Length", "0")))
        request = json.loads(raw)
        ledger.append({"boundary":"provider", "path":self.path, "request":request})
        result = {"id":"chatcmpl-offline", "object":"chat.completion", "created":1, "model":"mercury-3", "choices":[{"index":0,"message":{"role":"assistant","content":"offline-fixture-ok"},"finish_reason":"stop"}],"usage":{"prompt_tokens":20,"completion_tokens":5,"total_tokens":25,"prompt_tokens_details":{"cached_tokens":3},"completion_tokens_details":{"reasoning_tokens":2}}}
        if request.get("stream"):
            chunks = [{"id":"chatcmpl-offline","object":"chat.completion.chunk","created":1,"model":"mercury-3","choices":[{"index":0,"delta":{"role":"assistant","content":"offline-fixture-ok"},"finish_reason":None}]}, {"id":"chatcmpl-offline","object":"chat.completion.chunk","created":1,"model":"mercury-3","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}, {"id":"chatcmpl-offline","object":"chat.completion.chunk","created":1,"model":"mercury-3","choices":[],"usage":result["usage"]}]
            if case_name == "missing": chunks = chunks[:-1]
            body = ("".join("data: "+json.dumps(c)+"\n\n" for c in chunks)+"data: [DONE]\n\n").encode()
            mime = "text/event-stream"
        else:
            body=json.dumps(result).encode(); mime="application/json"
        ledger.append({"boundary":"provider_response","case":case_name,"body":body.decode()})
        self.send_response(200);self.send_header("Content-Type",mime);self.send_header("Content-Length",str(len(body)));self.end_headers();self.wfile.write(body)
class Capture(http.server.BaseHTTPRequestHandler):
    def log_message(self,*args): pass
    def do_POST(self):
        raw=self.rfile.read(int(self.headers.get("Content-Length","0")))
        ledger.append({"boundary":"messages", "path":self.path,"request":json.loads(raw)})
        c=http.client.HTTPConnection("127.0.0.1",bridge_port,timeout=20)
        try:
            c.request("POST",self.path,raw,{"Content-Type":"application/json","Authorization":"Bearer sk-offline-fixture","anthropic-version":self.headers.get("anthropic-version","2023-06-01")})
            res=c.getresponse();body=res.read()
            ledger.append({"boundary":"bridge_response","status":res.status,"body":body.decode(errors="replace")})
            self.send_response(res.status);self.send_header("Content-Type",res.getheader("Content-Type","application/json"));self.send_header("Content-Length",str(len(body)));self.end_headers();self.wfile.write(body)
        finally:c.close()
stub=http.server.ThreadingHTTPServer(("127.0.0.1",0),Stub)
capture=http.server.ThreadingHTTPServer(("127.0.0.1",0),Capture)
with socket.socket() as reserve:
    reserve.bind(("127.0.0.1",0));bridge_port=reserve.getsockname()[1]
ports={"provider":stub.server_port,"capture":capture.server_port,"bridge":bridge_port}
base_policy = '(version 1)\n(allow default)\n(deny network*)\n(deny file-write*)\n'
base_policy += '(allow file-write* (subpath '+json.dumps(str(R))+') (literal "/dev/null") (literal '+json.dumps(str(bridge_log))+'))\n'
base_policy += '(deny file-read* (subpath '+json.dumps(str(pathlib.Path.home()))+'))\n(allow file-read-metadata (subpath '+json.dumps(str(pathlib.Path.home()))+'))\n'
base_policy += '(allow file-read* (subpath '+json.dumps(str(R))+') (literal '+json.dumps(str(CLI))+'))\n(deny mach-lookup (global-name "com.apple.securityd"))\n'
policy=base_policy+"\n".join('(allow network-outbound (remote ip "localhost:%d"))'%p for p in ports.values())+'\n(allow network-inbound (local ip "localhost:%d"))\n'%bridge_port
(R/"runtime.sb").write_text(policy)
# Negative controls use documentation-only external IPs and a benign local sentinel.
with tempfile.NamedTemporaryFile(mode="w+", dir=R.parent, prefix="mercury-isolation-", delete=True) as owned_sentinel:
    sentinel=pathlib.Path(owned_sentinel.name)
    owned_sentinel.write("outside-fixture")
    owned_sentinel.flush()
    check="import errno,socket,pathlib; "
    check+="s=socket.socket(); s.settimeout(1); assert s.connect_ex(('192.0.2.1',443)) == errno.EPERM; "
    check+="s=socket.socket(socket.AF_INET6); s.settimeout(1); assert s.connect_ex(('2001:db8::1',443)) == errno.EPERM; "
    check+="p=pathlib.Path("+repr(str(sentinel))+"); "
    check+="exec(\"try: p.read_text()\\nexcept PermissionError: pass\\nelse: raise AssertionError('read escaped')\"); "
    check+="exec(\"try: p.write_text('bad')\\nexcept PermissionError: pass\\nelse: raise AssertionError('write escaped')\"); print('runtime isolation negative controls passed')"
    guard=subprocess.run(["/usr/bin/sandbox-exec","-f",str(R/"runtime.sb"),"/opt/homebrew/bin/python3","-I","-c",check],cwd=R,env={"PATH":"/usr/bin:/bin"},capture_output=True,text=True,timeout=5)
    (OUT/"isolation.txt").write_text(guard.stdout+guard.stderr)
    assert guard.returncode==0, guard.stderr
    assert sentinel.read_text()=="outside-fixture"

(R/"proxy.yaml").write_text("model_list:\n  - model_name: mercury\n    litellm_params:\n      model: inception/mercury-3\n      api_base: http://127.0.0.1:%d/v1\n      api_key: offline-provider-fixture\n      max_retries: 0\nlitellm_settings:\n  drop_params: false\n  telemetry: false\n  num_retries: 0\nrouter_settings:\n  num_retries: 0\ngeneral_settings:\n  master_key: sk-offline-fixture\n"%stub.server_port)
for server in [stub,capture]:threading.Thread(target=server.serve_forever,daemon=True).start()
env={"PATH":"/opt/homebrew/bin:/usr/bin:/bin","HOME":str(R/"bridge-home"),"TMPDIR":str(R),"XDG_CACHE_HOME":str(R/"cache"),"LITELLM_LOCAL_MODEL_COST_MAP":"True","DO_NOT_TRACK":"1","LITELLM_LOG":"ERROR","PYTHON_DOTENV_DISABLED":"1","PYTHONUNBUFFERED":"1"}
(R/"bridge-home").mkdir(exist_ok=True)
args=["/usr/bin/sandbox-exec","-f",str(R/"runtime.sb"),str(R/"venv/bin/python"),"-X","importtime",str(R/"venv/bin/litellm"),"--config",str(R/"proxy.yaml"),"--host","127.0.0.1","--port",str(bridge_port)]
log=open(bridge_log,"x")
bridge=subprocess.Popen(args,cwd=R,env=env,stdout=log,stderr=subprocess.STDOUT,start_new_session=True)
print(json.dumps({"driver_pid":os.getpid(),"bridge_pid":bridge.pid,"ports":ports,"bridge_log":str(bridge_log)}),flush=True)
results=[]
try:
    ready=False
    for i in range(400):
        if bridge.poll() is not None:break
        try:
            req=urllib.request.Request("http://127.0.0.1:%d/v1/models"%bridge_port,headers={"Authorization":"Bearer sk-offline-fixture"})
            with urllib.request.urlopen(req,timeout=.2) as res:
                ready="mercury" in res.read().decode()
            if ready:break
        except Exception:pass
        time.sleep(.1)
    if not ready:raise RuntimeError("bridge did not become ready; exit="+str(bridge.poll()))
    for case_name,effort in [("low","low"),("high","high"),("missing","low")]:
        home=OUT/("cli-"+case_name);home.mkdir(exist_ok=True);(home/".claude").mkdir(exist_ok=True)
        (home/".claude/settings.json").write_text(json.dumps({"model":"mercury","effortLevel":effort,"permissions":{"allow":["Read","Glob","Grep"],"defaultMode":"dontAsk"}}))
        child_env={"PATH":"/opt/homebrew/bin:/usr/bin:/bin","HOME":str(home),"CLAUDE_CONFIG_DIR":str(home/".claude"),"TMPDIR":str(home),"XDG_CONFIG_HOME":str(home),"XDG_CACHE_HOME":str(home),"ANTHROPIC_AUTH_TOKEN":"sk-offline-fixture","ANTHROPIC_BASE_URL":"http://127.0.0.1:%d"%capture.server_port,"ANTHROPIC_MODEL":"mercury","ANTHROPIC_DEFAULT_HAIKU_MODEL":"mercury","ANTHROPIC_DEFAULT_SONNET_MODEL":"mercury","ANTHROPIC_DEFAULT_OPUS_MODEL":"mercury","CLAUDE_CODE_SUBAGENT_MODEL":"mercury","CLAUDE_CODE_MAX_OUTPUT_TOKENS":"1024","CLAUDE_CODE_MAX_RETRIES":"0","MAX_STRUCTURED_OUTPUT_RETRIES":"1","DISABLE_AUTOUPDATER":"1","DISABLE_TELEMETRY":"1","DISABLE_ERROR_REPORTING":"1"}
        cli=["/usr/bin/sandbox-exec","-f",str(R/"runtime.sb"),str(CLI),"--model","mercury","--effort",effort,"--print","--output-format","json","--bare","--safe-mode","--restricted","--setting-sources","","--settings",str(home/".claude/settings.json"),"--strict-mcp-config","--mcp-config",'{"mcpServers":{}}',"--disable-slash-commands","--no-chrome","--tools","Read,Glob,Grep","--allowedTools","Read,Glob,Grep","--permission-mode","dontAsk","--permission-prompts","none","--max-turns","1","--no-session-persistence"]
        before=len(ledger)
        call=subprocess.Popen(cli,cwd=home,env=child_env,stdin=subprocess.PIPE,stdout=subprocess.PIPE,stderr=subprocess.PIPE,text=True,start_new_session=True)
        try:
            stdout,stderr=call.communicate("Instructions: offline contract sentinel\nTask: reply offline-fixture-ok. Do not use tools.\n",timeout=35)
        except subprocess.TimeoutExpired:
            os.killpg(call.pid,signal.SIGKILL);stdout,stderr=call.communicate(timeout=5)
        call.stdout_text,call.stderr_text=stdout,stderr
        (OUT/(case_name+".stdout")).write_text(call.stdout_text);(OUT/(case_name+".stderr")).write_text(call.stderr_text)
        results.append({"case":case_name,"effort":effort,"exit":call.returncode,"ledger_start":before,"ledger_end":len(ledger)})
    (OUT/"result.json").write_text(json.dumps({"results":results,"ports":ports},indent=2))
finally:
    (OUT/"ledger.json").write_text(json.dumps(ledger,indent=2))
    if bridge.poll() is None:
        os.killpg(bridge.pid,signal.SIGTERM)
        try:bridge.wait(timeout=5)
        except subprocess.TimeoutExpired:os.killpg(bridge.pid,signal.SIGKILL);bridge.wait(timeout=5)
    for server in [capture,stub]:server.shutdown();server.server_close()
    log.close()
    print(json.dumps({"bridge_exit":bridge.returncode,"requests":len(ledger),"results":results}),flush=True)
