"""Finite research tests; no dependency imports, network or model calls.

Oracle: only explicit provider integers can be credited, once per capture exchange;
missing/invalid/interrupted evidence cannot complete the supplied inventory.
"""
import copy
import hashlib
import importlib.util
import itertools
import json
import pathlib
import sys
import unittest

sys.dont_write_bytecode = True
HERE = pathlib.Path(__file__).resolve().parent
SOURCE = pathlib.Path(sys.argv.pop(1)) if len(sys.argv) > 1 else HERE / 'usage_provenance.py'
spec = importlib.util.spec_from_file_location('usage_provenance', SOURCE)
u = importlib.util.module_from_spec(spec)
spec.loader.exec_module(u)


def chunk(choices, **extra):
    return dict(id='fixture-response', object='chat.completion.chunk', model='mercury-3',
                choices=choices, **extra)


def event(obj):
    return 'data: ' + json.dumps(obj) + '\n\n'


FINISH = event(chunk([dict(index=0, delta={}, finish_reason='stop')]))
DONE = 'data: [DONE]\n\n'
KNOWN = dict(prompt_tokens=20, completion_tokens=5, total_tokens=25,
             prompt_tokens_details={'cached_tokens': 3}, completion_tokens_details={'reasoning_tokens': 2})


def wire(usage=KNOWN):
    return FINISH + (event(chunk([], usage=usage)) if usage is not None else '') + DONE


def exchange(body=None, **overrides):
    body = wire() if body is None else body
    result = dict(exchange_id='x1', attempt_id='a1', body=body,
                  body_sha256=hashlib.sha256(body.encode()).hexdigest(), http_status=200, termination='eof')
    result.update(overrides)
    return result


class Contract(unittest.TestCase):
    def test_retained_provider_evidence(self):
        observations = json.loads((HERE / 'observations.json').read_text())
        rows = [exchange(c['provider_sse'], exchange_id=c['case'], attempt_id=c['case'])
                for c in observations['cases']]
        result = u.inspect_history(rows)
        self.assertEqual(result['known_primary_subtotal'], {'prompt_tokens': 40, 'completion_tokens': 10})
        self.assertEqual(result['unknown_exchange_ids'], ['missing'])
        self.assertFalse(result['supplied_inventory_complete'])
        for row in result['exchanges'][:2]:
            self.assertEqual(row['fields'], dict(prompt_tokens=20, completion_tokens=5, total_tokens=25,
                                               cached_tokens=3, reasoning_tokens=2))
            self.assertTrue(row['details_complete'])
        self.assertEqual(result['exchanges'][2]['fields'], {})
        self.assertFalse(result['runtime_qualified'])
        self.assertEqual(u.inspect_history(rows), result)  # deterministic replay

    def test_zero_partial_and_absent_matrix(self):
        # Exhaustive 3^4 grid: absence, explicit zero, positive. Independent expectations.
        cases = 0
        for p, c, cache, reason in itertools.product((None, 0, 2), repeat=4):
            raw = {}
            for key, value in [('prompt_tokens', p), ('completion_tokens', c)]:
                if value is not None:
                    raw[key] = value
            for key, inner, value in [('prompt_tokens_details', 'cached_tokens', cache),
                                      ('completion_tokens_details', 'reasoning_tokens', reason)]:
                if value is not None:
                    raw[key] = {inner: value}
            result = u.inspect_exchange(exchange(wire(raw)))
            invalid = (cache is not None and p is not None and cache > p or
                       reason is not None and c is not None and reason > c)
            with self.subTest(raw=raw):
                self.assertEqual(result['usage_state'], 'invalid' if invalid else 'reported')
                self.assertEqual(result['primary_complete'], not invalid and p is not None and c is not None)
                if not invalid:
                    self.assertEqual(result['fields'], dict(prompt_tokens=p, completion_tokens=c,
                                                           total_tokens=None, cached_tokens=cache, reasoning_tokens=reason))
                    self.assertEqual(result['details_complete'], all(x is not None for x in (p, c, cache, reason)))
            cases += 1
        self.assertEqual(cases, 81)
        print('generated counter grid: 81 completed')

    def test_invalid_counts_and_totals(self):
        for value in (-1, True, '5', 1.2, 1e400, 2**53):
            for key in ('prompt_tokens', 'completion_tokens', 'total_tokens'):
                raw = dict(KNOWN, **{key: value})
                with self.subTest(key=key, value=value):
                    self.assertEqual(u.inspect_exchange(exchange(wire(raw)))['usage_state'], 'invalid')
        for raw in ([], 0, {'prompt_tokens': 20, 'total_tokens': 5}, {'prompt_tokens': 20, 'completion_tokens': 5, 'total_tokens': 26},
                    dict(KNOWN, prompt_tokens_details=[]), dict(KNOWN, completion_tokens_details={'reasoning_tokens': True})):
            self.assertEqual(u.inspect_exchange(exchange(wire(raw)))['usage_state'], 'invalid')
        self.assertEqual(u.inspect_exchange(exchange(wire({'prompt_tokens': 0, 'completion_tokens': 0,
                                                          'total_tokens': 0})))['fields']['prompt_tokens'], 0)

    def test_null_is_unknown_not_zero(self):
        result = u.inspect_exchange(exchange(wire({'prompt_tokens': None, 'completion_tokens': 0})))
        self.assertIsNone(result['fields']['prompt_tokens'])
        self.assertEqual(result['fields']['completion_tokens'], 0)
        self.assertFalse(result['primary_complete'])
        self.assertEqual(u.inspect_exchange(exchange(wire(None)))['usage_state'], 'missing')

    def test_duplicate_json_keys_and_nonfinite(self):
        for raw in ('{"prompt_tokens":20,"prompt_tokens":0}', '{"prompt_tokens":NaN}',
                    '{"prompt_tokens":Infinity}', '{"prompt_tokens":-Infinity}'):
            body = FINISH + 'data: {"id":"fixture-response","object":"chat.completion.chunk","model":"mercury-3","choices":[],"usage":' + raw + '}\n\n' + DONE
            self.assertEqual(u.inspect_exchange(exchange(body))['usage_state'], 'invalid')

    def test_overflow_anywhere_refused(self):
        for token in ('1e999', '-1e999'):
            for before, after in [('"total_tokens": 25', '"total_tokens": 25, "extra": '+token),
                                  ('"total_tokens": 25', '"total_tokens": 25, "extra": {"nested": ['+token+']}'),
                                  ('"object": "chat.completion.chunk"', '"created": '+token+', "object": "chat.completion.chunk"')]:
                body = wire().replace(before, after)
                with self.subTest(token=token, after=after):
                    result = u.inspect_exchange(exchange(body))
                    self.assertEqual(result['usage_state'], 'invalid')
                    self.assertFalse(result['primary_complete'])

    def test_partial_total_subset_lower_bounds(self):
        raws = [{'prompt_tokens':20, 'total_tokens':20, 'completion_tokens_details':{'reasoning_tokens':2}},
                {'completion_tokens':5, 'total_tokens':5, 'prompt_tokens_details':{'cached_tokens':3}},
                {'total_tokens':4, 'prompt_tokens_details':{'cached_tokens':3}, 'completion_tokens_details':{'reasoning_tokens':2}}]
        for raw in raws:
            with self.subTest(raw=raw):
                self.assertEqual(u.inspect_exchange(exchange(wire(raw)))['usage_state'], 'invalid')
        possible = {'prompt_tokens':20, 'total_tokens':25, 'completion_tokens_details':{'reasoning_tokens':2}}
        result = u.inspect_exchange(exchange(wire(possible)))
        self.assertEqual(result['usage_state'], 'reported')
        self.assertIsNone(result['fields']['completion_tokens'])
        self.assertFalse(result['primary_complete'])

    def test_duplicate_usage_and_orderings(self):
        usage = event(chunk([], usage=KNOWN))
        for parts in itertools.permutations((FINISH, usage, DONE)):
            result = u.inspect_exchange(exchange(''.join(parts)))
            self.assertEqual(result['primary_complete'], parts == (FINISH, usage, DONE))
        for second in (KNOWN, dict(KNOWN, completion_tokens=6, total_tokens=26)):
            result = u.inspect_exchange(exchange(FINISH + usage + event(chunk([], usage=second)) + DONE))
            self.assertEqual(result['usage_state'], 'invalid')
            self.assertEqual(len(result['raw_usage']), 2)
            self.assertFalse(result['primary_complete'])
        print('event orderings: 6 completed; duplicate usage: 2 completed')

    def test_transport_outcomes_retain_observed_counts(self):
        for end, done in itertools.product(('eof', 'cancelled', 'timeout', 'disconnect'), (True, False)):
            result = u.inspect_exchange(exchange(wire() if done else wire()[:-len(DONE)], termination=end))
            self.assertEqual(result['fields']['completion_tokens'], 5)
            self.assertEqual(result['primary_complete'], end == 'eof' and done)
        print('transport/DONE grid: 8 completed')

    def test_http_failures_and_retry_history(self):
        for status in (401, 402, 429, 500, 503):
            rows = [exchange('{"error":"fixture"}', exchange_id='failed', http_status=status),
                    exchange(exchange_id='retry')]
            result = u.inspect_history(rows)
            self.assertEqual(result['known_primary_subtotal'], {'prompt_tokens': 20, 'completion_tokens': 5})
            self.assertEqual(result['unknown_exchange_ids'], ['failed'])
            self.assertEqual(len(result['exchanges']), 2)
            self.assertFalse(result['supplied_inventory_complete'])

    def test_history_permutations_reference_inventory(self):
        # Test-owned reference records; equal provider IDs must never dedupe requests.
        rows = [exchange(exchange_id='known'), exchange(wire(None), exchange_id='missing'),
                exchange(exchange_id='cancelled', termination='cancelled')]
        for ordered in itertools.permutations(rows):
            result = u.inspect_history(list(ordered))
            self.assertEqual(result['known_primary_subtotal'], {'prompt_tokens': 20, 'completion_tokens': 5})
            self.assertEqual(result['unknown_exchange_ids'], [r['exchange_id'] for r in ordered if r['exchange_id'] != 'known'])
            self.assertEqual([r['exchange_id'] for r in result['exchanges']], [r['exchange_id'] for r in ordered])
        result = u.inspect_history([exchange(exchange_id='first'), exchange(exchange_id='second')])
        self.assertEqual(result['known_primary_subtotal'], {'prompt_tokens': 40, 'completion_tokens': 10})
        self.assertTrue(result['supplied_inventory_complete'])
        self.assertFalse(result['runtime_qualified'])
        print('history permutations: 6 completed; repeated provider ID: 2 exchanges counted')

    def test_bad_framing_identity_and_shape(self):
        bad = [wire() + DONE, wire().replace('\n\n', '\n', 1), wire()[:-1],
               wire().replace('data: ', 'event: ', 1), wire().replace('mercury-3', 'other', 1),
               wire().replace('fixture-response', 'other', 1),
               wire().replace('"index": 0', '"index": true'),
               wire().replace('"delta": {}', '"delta": []'),
               wire().replace('"stop"', '"unknown"'),
               'data: []\n\n', 'data: {bad}\n\n',
               wire().replace('"choices": []', '"choices": {}'),
               wire().replace('chat.completion.chunk', 'error', 1),
               FINISH + event(chunk([], usage=None)) + DONE,
               wire().replace('\n', '\r')]
        for body in bad:
            with self.subTest(body=body[:80]):
                result = u.inspect_exchange(exchange(body))
                self.assertEqual(result['usage_state'], 'invalid')
                self.assertFalse(result['primary_complete'])
        self.assertTrue(u.inspect_exchange(exchange(wire().replace('\n', '\r\n')))['primary_complete'])
        self.assertTrue(u.inspect_exchange(exchange(': keepalive\n\n' + wire()))['primary_complete'])

    def test_manifest_and_size_boundaries(self):
        missing = exchange(); del missing['http_status']
        for rows in ([], [None], [missing], [exchange(estimate=5)], [exchange()] * 2, [exchange(exchange_id='')], [exchange(attempt_id=5)],
                     [exchange(body_sha256='wrong')], [exchange(http_status=True)],
                     [exchange(termination='unknown')], [exchange('x' * (u.MAX_BYTES + 1))],
                     [exchange(exchange_id=str(i)) for i in range(u.MAX_EXCHANGES + 1)]):
            with self.assertRaises(ValueError):
                u.inspect_history(rows)
        result = u.inspect_history([exchange(exchange_id=str(i)) for i in range(u.MAX_EXCHANGES)])
        self.assertEqual(len(result['exchanges']), 256)
        self.assertEqual(result['known_primary_subtotal'], {'prompt_tokens': 5120, 'completion_tokens': 1280})
        # At byte cap (comments supported); one byte over refused above.
        body = ':' + 'x' * (u.MAX_BYTES - len(wire()) - 3) + '\n\n' + wire()
        self.assertEqual(len(body.encode()), u.MAX_BYTES)
        self.assertTrue(u.inspect_exchange(exchange(body))['primary_complete'])
        self.assertEqual(u.inspect_exchange(exchange(':\n\n' * 4096))['usage_state'], 'invalid')
        print('boundary activation: 256/257 exchanges, 1048576/1048577 bytes, 4096 frames')

    def test_input_unchanged(self):
        rows = [exchange()]
        snapshot = copy.deepcopy(rows)
        u.inspect_history(rows)
        self.assertEqual(rows, snapshot)


if __name__ == '__main__':
    unittest.main(verbosity=2)
