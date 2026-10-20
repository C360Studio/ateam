"""macOS research driver: one owned native attempt, isolated synthetic provider, no paid calls."""
import hashlib
import http.server
import json
import os
from pathlib import Path
import select
import shutil
import signal
import socket
import subprocess
import sys
import threading
import time
import urllib.request

R, CLI, OUT = (Path(p).resolve() for p in sys.argv[1:4])
CASE = sys.argv[4]
CASES = ('known', 'zero', 'missing', 'partial', 'http-error', 'disconnect', 'cancel', 'intent-fail', 'body-fail', 'disabled', 'oversize', 'end-file-fail', 'end-dir-fail')
assert CASE in CASES
assert R.is_relative_to(Path.home().resolve()) and OUT.is_relative_to(R) and OUT != R
assert hashlib.sha256(CLI.read_bytes()).hexdigest() == '97a01e5bc74a199e67189435d0331ea3a24eac2e07db4b76d9148c5b0386138f'
for path in ('managed-settings.json', 'managed-settings.d', 'managed-mcp.json'):
    assert not (Path('/Library/Application Support/ClaudeCode') / path).exists(), 'managed policy needs review'
OUT.mkdir(parents=True, exist_ok=False)
EVIDENCE = OUT / 'evidence'; EVIDENCE.mkdir()
CLI_HOME = OUT / 'cli-home'; CLI_HOME.mkdir()
shutil.copyfile(Path(__file__).with_name('capture_hook.py'), OUT / 'capture_hook.py')
fixture = []
request_received = threading.Event()
cancel_ack = threading.Event()
connection_observed = threading.Event()
fixture_done = threading.Event()


def event(choices, **extra):
    obj = dict(id='capture-fixture', object='chat.completion.chunk', created=1, model='mercury-3', choices=choices, **extra)
    return ('data: ' + json.dumps(obj) + '\n\n').encode()


prefix = event([dict(index=0, delta={'role':'assistant', 'content':'offline-fixture-ok'}, finish_reason=None)])
finish = event([dict(index=0, delta={}, finish_reason='stop')])
usage = dict(prompt_tokens=20, completion_tokens=5, total_tokens=25,
             prompt_tokens_details={'cached_tokens':3}, completion_tokens_details={'reasoning_tokens':2})
if CASE == 'zero':
    usage = dict(prompt_tokens=0, completion_tokens=0, total_tokens=0,
                 prompt_tokens_details={'cached_tokens':0}, completion_tokens_details={'reasoning_tokens':0})
if CASE == 'partial':
    usage = dict(prompt_tokens=20, prompt_tokens_details={'cached_tokens':3})
body = prefix + finish + (b'' if CASE == 'missing' else event([], usage=usage)) + b'data: [DONE]\n\n'
if CASE == 'http-error':
    body = b'{"error":{"message":"offline rate limit","type":"rate_limit"}}'
if CASE == 'oversize':
    body = b':' + b'x' * 1_048_576 + b'\n\n' + body


class Stub(http.server.BaseHTTPRequestHandler):
    protocol_version = 'HTTP/1.1'

    def log_message(self, *_):
        pass

    def do_POST(self):
        self.connection.settimeout(5)
        length = int(self.headers.get('Content-Length', '0'))
        if not 0 < length <= 1_048_576 or len(fixture) >= 8:
            self.send_error(413); return
        raw = self.rfile.read(length)
        entry = dict(exchange_id=self.headers.get('x-ateam-exchange-id'), request_sha256=hashlib.sha256(raw).hexdigest(),
                     request=json.loads(raw), path=self.path, response_body=body.decode(), response_status=429 if CASE == 'http-error' else 200)
        fixture.append(entry)
        request_received.set()
        self.send_response(entry['response_status'])
        self.send_header('Content-Type', 'application/json' if CASE == 'http-error' else 'text/event-stream')
        # Explicit chunked transfer proves capture is not dependent on Content-Length.
        self.send_header('Transfer-Encoding', 'chunked')
        self.send_header('Connection', 'close')
        self.end_headers()
        try:
            if CASE in ('disconnect', 'cancel'):
                self.wfile.write(('%x\r\n' % len(prefix)).encode() + prefix + b'\r\n'); self.wfile.flush()
                entry['response_body'] = prefix.decode()
                if CASE == 'cancel':
                    if not cancel_ack.wait(10):
                        raise TimeoutError('native cancellation acknowledgement absent')
                    ready, _, _ = select.select([self.connection], [], [], 2)
                    entry['connection_closed_after_cli_exit'] = bool(ready and self.connection.recv(1, socket.MSG_PEEK) == b'')
                    connection_observed.set()
                # Deliberately omit terminating chunk: observable incomplete transport.
                self.close_connection = True
                return
            # Split the final usage over wire chunks; no timing inference from chunk sizes.
            for part in (body[:117], body[117:]):
                self.wfile.write(('%x\r\n' % len(part)).encode() + part + b'\r\n'); self.wfile.flush()
            self.wfile.write(b'0\r\n\r\n'); self.wfile.flush()
        except (BrokenPipeError, ConnectionResetError) as error:
            entry['fixture_write_error'] = type(error).__name__
        finally:
            fixture_done.set()


stub = http.server.ThreadingHTTPServer(('127.0.0.1', 0), Stub)
with socket.socket() as reserve:
    reserve.bind(('127.0.0.1', 0)); bridge_port = reserve.getsockname()[1]
endpoint = 'http://127.0.0.1:%d/v1/chat/completions' % stub.server_port
bridge_log = Path.cwd() / '.enjoy-logs' / (OUT.name + '-bridge.log')
bridge_log.parent.mkdir(exist_ok=True)


def policy(read_roots, write_roots, outbound, inbound=None):
    text = '(version 1)\n(allow default)\n(deny network*)\n(deny file-write*)\n'
    text += '(allow file-write* (literal "/dev/null") ' + ' '.join('(subpath '+json.dumps(str(p))+')' for p in write_roots) + ')\n'
    text += '(deny file-read* (subpath '+json.dumps(str(Path.home()))+'))\n'
    text += '(allow file-read-metadata (subpath '+json.dumps(str(Path.home()))+'))\n'
    text += '(allow file-read* (literal '+json.dumps(str(CLI))+') ' + ' '.join('(subpath '+json.dumps(str(p))+')' for p in read_roots) + ')\n'
    text += '(deny mach-lookup (global-name "com.apple.securityd"))\n'
    text += '\n'.join('(allow network-outbound (remote ip "localhost:%d"))' % p for p in outbound) + '\n'
    if inbound is not None:
        text += '(allow network-inbound (local ip "localhost:%d"))\n' % inbound
    return text


(OUT/'bridge.sb').write_text(policy([R], [OUT, bridge_log], [stub.server_port], bridge_port))
(OUT/'cli.sb').write_text(policy([CLI_HOME], [CLI_HOME], [bridge_port]))
# Native process cannot modify evidence; both native and bridge must deny external egress.
sentinel = EVIDENCE/'ownership-sentinel'; sentinel.write_text('preserve')
for name in ('bridge', 'cli'):
    check = "import errno,socket,pathlib; s=socket.socket();s.settimeout(1);assert s.connect_ex(('192.0.2.1',443))==errno.EPERM; s=socket.socket(socket.AF_INET6);s.settimeout(1);assert s.connect_ex(('2001:db8::1',443))==errno.EPERM;"
    if name == 'cli':
        check += "p=pathlib.Path("+repr(str(sentinel))+");exec(\"try: p.write_text('bad')\\nexcept PermissionError: pass\\nelse: raise AssertionError('capture write escaped')\");"
    check += "print('isolation control passed')"
    control = subprocess.run(['/usr/bin/sandbox-exec','-f',str(OUT/(name+'.sb')),'/opt/homebrew/bin/python3','-I','-c',check],
                             env={'PATH':'/usr/bin:/bin'}, cwd=CLI_HOME, capture_output=True,text=True,timeout=5)
    (OUT/(name+'-isolation.txt')).write_text(control.stdout+control.stderr)
    assert control.returncode == 0, control.stderr
assert sentinel.read_text() == 'preserve'

config = 'model_list:\n  - model_name: mercury\n    litellm_params:\n      model: inception/mercury-3\n      api_base: http://127.0.0.1:%d/v1\n      api_key: offline-provider-fixture\n      max_retries: 0\nlitellm_settings:\n  drop_params: false\n  telemetry: false\n  num_retries: 0\n' % stub.server_port
if CASE != 'disabled':
    config += '  callbacks: [capture_hook.capture]\n'
config += 'router_settings:\n  num_retries: 0\ngeneral_settings:\n  master_key: sk-offline-fixture\n'
(OUT/'proxy.yaml').write_text(config)
bridge_home = OUT/'bridge-home'; bridge_home.mkdir()
env = dict(PATH='/opt/homebrew/bin:/usr/bin:/bin', HOME=str(bridge_home), TMPDIR=str(OUT),
           XDG_CACHE_HOME=str(OUT/'cache'), LITELLM_LOCAL_MODEL_COST_MAP='True', DO_NOT_TRACK='1',
           LITELLM_LOG='ERROR', PYTHON_DOTENV_DISABLED='1', PYTHONUNBUFFERED='1',
           CAPTURE_ROOT=str(EVIDENCE), CAPTURE_ATTEMPT=CASE, CAPTURE_ENDPOINT=endpoint,
           CAPTURE_FAULT={'intent-fail':'intent','body-fail':'body','end-file-fail':'end-file-sync','end-dir-fail':'end-dir-sync'}.get(CASE,''))
threading.Thread(target=stub.serve_forever, daemon=True).start()
log = bridge_log.open('x')
bridge = subprocess.Popen(['/usr/bin/sandbox-exec','-f',str(OUT/'bridge.sb'),str(R/'venv/bin/python'),
                           str(R/'venv/bin/litellm'),'--config',str(OUT/'proxy.yaml'),'--host','127.0.0.1','--port',str(bridge_port)],
                          cwd=OUT,env=env,stdout=log,stderr=subprocess.STDOUT,start_new_session=True)
print(json.dumps(dict(driver_pid=os.getpid(), bridge_pid=bridge.pid, fixture_port=stub.server_port, bridge_port=bridge_port, bridge_log=str(bridge_log))), flush=True)
cli = None
summary = dict(case=CASE, hook_sha256=hashlib.sha256((OUT/'capture_hook.py').read_bytes()).hexdigest())


def terminate(process):
    if process is not None and process.poll() is None:
        os.killpg(process.pid, signal.SIGTERM)
        try:
            process.wait(timeout=5)
        except subprocess.TimeoutExpired:
            os.killpg(process.pid, signal.SIGKILL); process.wait(timeout=5)


try:
    deadline = time.monotonic() + 60
    while time.monotonic() < deadline:
        if bridge.poll() is not None:
            raise RuntimeError('bridge exited before readiness')
        try:
            request = urllib.request.Request('http://127.0.0.1:%d/v1/models' % bridge_port, headers={'Authorization':'Bearer sk-offline-fixture'})
            with urllib.request.urlopen(request, timeout=.25) as response:
                if 'mercury' in response.read().decode(): break
        except Exception:
            pass
        time.sleep(.1)
    else:
        raise TimeoutError('bridge readiness deadline')
    (CLI_HOME/'.claude').mkdir()
    settings = CLI_HOME/'.claude/settings.json'
    settings.write_text(json.dumps(dict(model='mercury', effortLevel='low', permissions=dict(allow=['Read','Glob','Grep'], defaultMode='dontAsk'))))
    child_env = dict(PATH='/opt/homebrew/bin:/usr/bin:/bin', HOME=str(CLI_HOME), CLAUDE_CONFIG_DIR=str(CLI_HOME/'.claude'),
                     TMPDIR=str(CLI_HOME), XDG_CONFIG_HOME=str(CLI_HOME), XDG_CACHE_HOME=str(CLI_HOME),
                     ANTHROPIC_AUTH_TOKEN='sk-offline-fixture', ANTHROPIC_BASE_URL='http://127.0.0.1:%d'%bridge_port,
                     ANTHROPIC_MODEL='mercury', ANTHROPIC_DEFAULT_HAIKU_MODEL='mercury', ANTHROPIC_DEFAULT_SONNET_MODEL='mercury',
                     ANTHROPIC_DEFAULT_OPUS_MODEL='mercury', CLAUDE_CODE_SUBAGENT_MODEL='mercury',
                     CLAUDE_CODE_MAX_OUTPUT_TOKENS='1024', CLAUDE_CODE_MAX_RETRIES='0', MAX_STRUCTURED_OUTPUT_RETRIES='1',
                     DISABLE_AUTOUPDATER='1', DISABLE_TELEMETRY='1', DISABLE_ERROR_REPORTING='1')
    args = ['/usr/bin/sandbox-exec','-f',str(OUT/'cli.sb'),str(CLI),'--model','mercury','--effort','low','--print',
            '--output-format','json','--bare','--safe-mode','--restricted','--setting-sources','','--settings',str(settings),
            '--strict-mcp-config','--mcp-config','{"mcpServers":{}}','--disable-slash-commands','--no-chrome',
            '--tools','Read,Glob,Grep','--allowedTools','Read,Glob,Grep','--permission-mode','dontAsk','--permission-prompts','none',
            '--max-turns','1','--no-session-persistence']
    stdout = (OUT/'cli.stdout').open('x'); stderr = (OUT/'cli.stderr').open('x')
    cli = subprocess.Popen(args, cwd=CLI_HOME, env=child_env, stdin=subprocess.PIPE, stdout=stdout, stderr=stderr, text=True, start_new_session=True)
    cli.stdin.write('Offline capture qualification: reply offline-fixture-ok. Do not use tools.\n'); cli.stdin.close()
    if CASE == 'cancel':
        assert request_received.wait(15), 'provider request absent'
        deadline = time.monotonic() + 5
        while not list(EVIDENCE.glob('*.first.json')) and time.monotonic() < deadline:
            time.sleep(.02)
        assert list(EVIDENCE.glob('*.first.json')), 'durable first chunk absent'
        terminate(cli); cancel_ack.set()
        assert connection_observed.wait(4), 'fixture cancellation observation absent'
        summary['cancelled_after_durable_first_chunk'] = True
    else:
        try:
            cli.wait(timeout=35)
        except subprocess.TimeoutExpired:
            summary['native_timeout'] = True
            terminate(cli)
    summary['cli_exit'] = cli.returncode
    stdout.close(); stderr.close()
finally:
    terminate(cli)
    cancel_ack.set()
    terminate(bridge)
    stub.shutdown(); stub.server_close()
    log.close()
    summary['bridge_exit'] = bridge.returncode
    (OUT/'fixture.json').write_text(json.dumps(fixture, indent=2))
    (OUT/'summary.json').write_text(json.dumps(summary, indent=2))
    print(json.dumps(dict(summary=summary, fixture_requests=len(fixture))), flush=True)
