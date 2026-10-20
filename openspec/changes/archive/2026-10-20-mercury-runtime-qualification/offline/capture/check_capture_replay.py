"""Independent expectations for the finite research capture-evidence contract."""
import copy
import importlib.util
import itertools
import json
from pathlib import Path
import sys
import unittest

sys.dont_write_bytecode=True
HERE=Path(__file__).resolve().parent
SOURCE=Path(sys.argv.pop(1)) if len(sys.argv)>1 else HERE/'replay_capture.py'
spec=importlib.util.spec_from_file_location('replay_capture',SOURCE)
m=importlib.util.module_from_spec(spec);spec.loader.exec_module(m)
BUNDLE=json.loads((HERE/'capture-observations.json').read_text())
KNOWN=next(c for c in BUNDLE['cases'] if c['case']=='known')


class CaptureReplay(unittest.TestCase):
    def test_retained_evidence(self):
        results=m.check(BUNDLE)
        self.assertEqual(sum(r['accepted_capture'] for r in results.values()),4)
        self.assertEqual(sum(r['provider_usage_complete'] for r in results.values()),2)
        self.assertEqual(sum(r['known_primary_subtotal']['prompt_tokens'] for r in results.values()),20)
        self.assertEqual(sum(r['known_primary_subtotal']['completion_tokens'] for r in results.values()),5)
        self.assertTrue(all(r['runtime_qualified'] is False for r in results.values()))

    def test_inventory_missing_duplicate_foreign_or_wrong_request(self):
        for change in ('missing-fixture','missing-capture','duplicate-fixture','duplicate-capture','foreign-id','wrong-request'):
            c=copy.deepcopy(KNOWN)
            if change=='missing-fixture': c['fixture']=[]
            if change=='missing-capture': c['captures']=[]
            if change=='duplicate-fixture': c['fixture']*=2
            if change=='duplicate-capture': c['captures']*=2
            if change=='foreign-id': c['captures'][0]['intent']['exchange_id']='foreign'
            if change=='wrong-request': c['captures'][0]['intent']['request_sha256']='wrong'
            with self.subTest(change=change):
                r=m.assess(c);self.assertFalse(r['accepted_capture']);self.assertFalse(r['provider_usage_complete'])
                self.assertEqual(r['known_primary_subtotal'],dict(prompt_tokens=0,completion_tokens=0))

    def test_eof_body_and_control_guards(self):
        for change in ('close','body','status','sdk-retries','wire-field','effort','bridge-type','isolation'):
            c=copy.deepcopy(KNOWN)
            if change=='close': c['captures'][0]['end']['termination']='closed'
            if change=='body': c['fixture'][0]['body']+='x'
            if change=='status': c['fixture'][0]['response_status']=429
            if change=='sdk-retries': c['captures'][0]['intent']['sdk_max_retries']=2
            if change=='wire-field': c['fixture'][0]['unexpected_retry_fields']=['max_retries']
            if change=='effort': c['fixture'][0]['request_controls']['reasoning_effort']='high'
            if change=='bridge-type': c['summary']['bridge_exit']=False
            if change=='isolation': c['controls']['cli']=False
            with self.subTest(change=change):self.assertFalse(m.assess(c)['accepted_capture'])
        c=copy.deepcopy(KNOWN); c['captures'][0]['body']+='x'
        with self.assertRaises(ValueError):m.assess(c)

    def test_failure_and_native_outcome_grid(self):
        count=0
        for exit_code,is_error,failure in itertools.product((0,1),(False,True),(None,{'reason':'capture_error'})):
            c=copy.deepcopy(KNOWN)
            c['summary']['cli_exit']=exit_code;c['native']['is_error']=is_error;c['failure']=failure
            r=m.assess(c)
            expected=exit_code==0 and not is_error and failure is None
            self.assertEqual(r['accepted_capture'],expected)
            self.assertEqual(r['provider_usage_complete'],expected)
            self.assertEqual(r['known_primary_subtotal'],dict(prompt_tokens=20 if expected else 0,completion_tokens=5 if expected else 0))
            count+=1
        self.assertEqual(count,8);print('native/failure grid: 8 completed')

    def test_case_order_and_replay_idempotence(self):
        baseline={c['case']:m.assess(c) for c in BUNDLE['cases']}
        for ordering in (BUNDLE['cases'],list(reversed(BUNDLE['cases'])),BUNDLE['cases'][3:]+BUNDLE['cases'][:3]):
            self.assertEqual({c['case']:m.assess(c) for c in ordering},baseline)
        self.assertEqual(m.check(copy.deepcopy(BUNDLE)),m.check(BUNDLE))
        print('history reorderings: 3 completed; all13 cases retained')

    def test_source_pin_and_case_inventory(self):
        for key in ('hook_sha256','driver_sha256'):
            b=copy.deepcopy(BUNDLE);b[key]='bad'
            with self.assertRaises(AssertionError):m.check(b)
        b=copy.deepcopy(BUNDLE);b['cases'].pop()
        with self.assertRaises(AssertionError):m.check(b)


if __name__=='__main__':unittest.main(verbosity=2)
