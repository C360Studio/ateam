"""Run copied-source mutations; leave tracked research sources byte-for-byte unchanged."""
import hashlib
import pathlib
import shutil
import subprocess
import sys
import tempfile

HERE = pathlib.Path(__file__).resolve().parent
SOURCE = HERE / 'usage_provenance.py'
TEST = HERE / 'check_usage_provenance.py'
source = SOURCE.read_text()
original_hash = hashlib.sha256(SOURCE.read_bytes()).hexdigest()


def run(test, target):
    return subprocess.run([sys.executable, '-I', str(test), str(target)],
                          capture_output=True, text=True, timeout=20)


def passed(result):
    if result.returncode:
        raise AssertionError(result.stdout + result.stderr)


passed(run(TEST, SOURCE))
mutations = [
    ('missing-to-zero', 'value = obj.get(key)', 'value = obj.get(key, 0)', 'test_zero_partial_and_absent_matrix'),
    ('ignore-cancellation', "result['done'] and end == 'eof' and", "result['done'] and", 'test_transport_outcomes_retain_observed_counts'),
    ('double-count-reasoning', "subtotal[key] += row['fields'][key]", "subtotal[key] += row['fields'][key] + (row['fields'].get('reasoning_tokens') or 0)", 'test_retained_provider_evidence'),
    ('accept-duplicate-usage', 'if saw_usage or not finished or choices:', 'if not finished or choices:', 'test_duplicate_usage_and_orderings'),
    ('dedupe-provider-id', "row.update(exchange_id=identity, attempt_id=exchange['attempt_id'])", "if any(old['response_id'] == row['response_id'] for old in rows):\n            continue\n        row.update(exchange_id=identity, attempt_id=exchange['attempt_id'])", 'test_history_permutations_reference_inventory'),
    ('lose-reasoning-detail', 'values[inner] = count(detail or {}, inner)', "values[inner] = 0 if inner == 'reasoning_tokens' else count(detail or {}, inner)", 'test_retained_provider_evidence'),
    ('ignore-body-hash', "if digest != exchange['body_sha256']:", 'if False:', 'test_manifest_and_size_boundaries'),
    ('bool-is-count', 'type(value) is not int', 'not isinstance(value, int)', 'test_invalid_counts_and_totals'),
    ('ignore-byte-limit', "if type(body) is not str or len(body.encode('utf-8')) > MAX_BYTES:", 'if type(body) is not str:', 'test_manifest_and_size_boundaries'),
]
with tempfile.TemporaryDirectory() as temp:
    for name, before, after, expected in mutations:
        assert source.count(before) == 1, name
        target = pathlib.Path(temp) / (name + '.py')
        target.write_text(source.replace(before, after))
        result = run(TEST, target)
        output = result.stdout + result.stderr
        assert result.returncode != 0 and 'FAIL: ' + expected in output, name + ': invalid/survivor/inconclusive\n' + output
        print(name + ': detection')
passed(run(TEST, SOURCE))
assert hashlib.sha256(SOURCE.read_bytes()).hexdigest() == original_hash
print('reader: baseline/after pass, SHA-256 unchanged ' + original_hash)

if sys.platform == 'darwin':
    sentinel = HERE / 'check_sentinel.py'
    test = HERE / 'check_sentinel_cwd.py'
    original = sentinel.read_bytes()
    passed(run(test, sentinel))
    with tempfile.TemporaryDirectory() as temp:
        target = pathlib.Path(temp) / sentinel.name
        guard = "assert test_root.is_relative_to(pathlib.Path.home().resolve()), 'cwd must be under the user home for the file-denial control'"
        assert original.decode().count(guard) == 1
        target.write_text(original.decode().replace(guard, 'pass  # mutant omits cwd precondition'))
        shutil.copyfile(HERE / 'reproduce.py', pathlib.Path(temp) / 'reproduce.py')
        result = run(test, target)
        assert result.returncode != 0 and 'FAIL: test_outside_home_refused_before_policy' in result.stderr, result.stderr
        assert 'FAIL: test_documented_worktree_succeeds' not in result.stderr, result.stderr
        print('omit-cwd-precondition: detection')
    passed(run(test, sentinel))
    assert sentinel.read_bytes() == original
    print('sentinel: baseline/after pass, SHA-256 unchanged ' + hashlib.sha256(original).hexdigest())
else:
    print('SKIP: macOS sentinel mutation')
