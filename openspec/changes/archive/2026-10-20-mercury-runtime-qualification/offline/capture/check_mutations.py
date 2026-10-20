"""Non-destructive copied-source mutation checks for these finite research tests."""
import hashlib
import os
from pathlib import Path
import shutil
import subprocess
import sys
import tempfile

sys.dont_write_bytecode = True
suite = sys.argv[1]
assert suite in ('hook', 'replay')
source = Path(__file__).resolve().parent
filename = 'capture_hook.py' if suite == 'hook' else 'replay_capture.py'
testfile = 'check_capture_' + suite + '.py'
original = (source / filename).read_bytes()
mutations = [
    ('retry-override', "allowed + ([] if 'max_retries' in allowed else ['max_retries'])", 'allowed', 'test_retry_override_preserves_allowlist_and_rejects_bad_shape'),
    ('size-limit', 'self.size + len(chunk) > MAX_BYTES', 'False', 'test_size_and_chunk_limits_before_yield'),
    ('before-yield', '                self.output.flush()\n', '', 'test_recorded_before_yield_and_close'),
    ('incomplete-eof', 'self.finish(reason)', "self.finish('eof')", 'test_close_cancel_and_transport_error_are_incomplete'),
    ('fatal-latch', '        if self.failed:\n', '        if False:\n', 'test_request_limit_and_fatal_latch'),
    ('buffer-before-sdk', 'await response.aread()', 'pass', 'test_buffered_httpx_hook_records_eof_before_consumer'),
] if suite == 'hook' else [
    ('failure-veto', "and case['failure'] is None", '', 'test_failure_and_native_outcome_grid'),
    ('native-veto', 'and native_success and supervised', 'and supervised', 'test_failure_and_native_outcome_grid'),
    ('eof-guard', "end.get('termination') == 'eof'", 'True', 'test_eof_body_and_control_guards'),
    ('control-guard', 'and control_valid and native_success', 'and native_success', 'test_eof_body_and_control_guards'),
    ('bridge-type', "type(summary['bridge_exit']) is int and ", '', 'test_eof_body_and_control_guards'),
    ('request-match', "intent['request_sha256'] == fixture['request_sha256']", 'True', 'test_inventory_missing_duplicate_foreign_or_wrong_request'),
]


def run(target):
    return subprocess.run([sys.executable, '-I', str(source / testfile), str(target)],
                          capture_output=True, text=True, timeout=60)


baseline = run(source / filename)
assert baseline.returncode == 0, baseline.stdout + baseline.stderr
print('baseline: pass', flush=True)
with tempfile.TemporaryDirectory(dir=os.environ.get('CAPTURE_ROOT')) as temporary:
    root = Path(temporary)
    capture = root / 'capture'; capture.mkdir()
    if suite == 'replay':
        shutil.copyfile(source.parent / 'usage_provenance.py', root / 'usage_provenance.py')
        for name in ('capture_hook.py', 'capture_reproduce.py'):
            shutil.copyfile(source / name, capture / name)
    for name, before, after, test in mutations:
        text = original.decode()
        assert before in text, name
        target = capture / filename
        target.write_text(text.replace(before, after))
        result = run(target)
        output = result.stdout + result.stderr
        # A named failing assertion is required; import/runtime errors are invalid, not detections.
        assert result.returncode == 1 and 'FAIL: ' + test in output and 'AssertionError' in output and 'ERROR:' not in output, name + '\n' + output
        print(name + ': detection (' + test + ')', flush=True)
after = run(source / filename)
assert after.returncode == 0, after.stdout + after.stderr
assert (source / filename).read_bytes() == original, 'original source changed'
print('after: pass; original SHA256 ' + hashlib.sha256(original).hexdigest())
