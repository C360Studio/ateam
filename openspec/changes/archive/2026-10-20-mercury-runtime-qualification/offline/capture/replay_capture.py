"""Finite no-network capture-evidence checker. Never qualifies a production runtime."""
import hashlib
import importlib.util
import json
from pathlib import Path
import sys

sys.dont_write_bytecode = True
HERE = Path(__file__).resolve().parent
spec = importlib.util.spec_from_file_location('usage_provenance', HERE.parent/'usage_provenance.py')
usage = importlib.util.module_from_spec(spec); spec.loader.exec_module(usage)


def assess(case):
    fixtures, captures = case['fixture'], case['captures']
    if type(fixtures) is not list or type(captures) is not list or max(len(fixtures), len(captures)) > 8:
        raise ValueError('invalid supplied inventory')
    fixture_ids = [f['exchange_id'] for f in fixtures]
    capture_ids = [c['intent']['exchange_id'] for c in captures]
    valid_ids = all(type(i) is str and i for i in fixture_ids + capture_ids)
    matched = (valid_ids and len(set(fixture_ids)) == len(fixtures) and
               len(set(capture_ids)) == len(captures) and set(fixture_ids) == set(capture_ids))
    wire_complete = bool(captures)
    control_valid = len(fixtures) == 1 and bool(captures)
    reports = []
    for capture in captures:
        intent, headers, end = capture['intent'], capture['headers'], capture['end']
        identity = intent['exchange_id']
        rows = [f for f in fixtures if f['exchange_id'] == identity]
        if len(rows) != 1:
            matched = False; wire_complete = False; continue
        fixture = rows[0]
        matched = matched and intent['request_sha256'] == fixture['request_sha256'] and intent['attempt_id'] == case['case']
        expected = dict(model='mercury-3',stream=True,max_tokens=1024,reasoning_effort='low',stream_options={'include_usage':True})
        control_valid = (control_valid and type(intent.get('sdk_max_retries')) is int and intent['sdk_max_retries'] == 0 and
                         intent.get('sdk_retry_header') == '0' and fixture['request_controls'] == expected and
                         fixture['unexpected_retry_fields'] == [] and fixture['path'] == '/v1/chat/completions')
        raw = capture['body']
        body_valid = (type(raw) is str and type(fixture['body']) is str and raw == fixture['body'] and
                      len(raw.encode()) == capture['body_bytes'] == fixture['body_bytes'] and
                      hashlib.sha256(raw.encode()).hexdigest() == capture['body_sha256'] == fixture['body_sha256'])
        terminal_valid = (type(end) is dict and end.get('termination') == 'eof' and body_valid and
                          end.get('bytes') == capture['body_bytes'] and end.get('body_sha256') == capture['body_sha256'] and
                          type(headers) is dict and headers.get('status') == fixture['response_status'])
        wire_complete = wire_complete and terminal_valid
        if type(raw) is str and type(headers) is dict:
            reports.append(usage.inspect_exchange(dict(exchange_id=identity,attempt_id=case['case'],body=raw,
                body_sha256=capture['body_sha256'],http_status=headers['status'],termination='eof' if terminal_valid else 'disconnect')))
    native, summary = case['native'], case['summary']
    native_success = (type(summary['cli_exit']) is int and summary['cli_exit'] == 0 and type(native) is dict and
                      native.get('type') == 'result' and native.get('subtype') == 'success' and native.get('is_error') is False)
    supervised = (summary.get('native_timeout') is not True and type(summary['bridge_exit']) is int and summary['bridge_exit'] in (0,-15) and
                  case['controls'].get('bridge') is True and case['controls'].get('cli') is True)
    accepted = bool(matched and wire_complete and control_valid and native_success and supervised and case['failure'] is None)
    complete = accepted and len(reports) == len(captures) and all(r['primary_complete'] for r in reports)
    subtotal = dict(prompt_tokens=0, completion_tokens=0)
    if accepted:
        for report in reports:
            if report['primary_complete']:
                for key in subtotal: subtotal[key] += report['fields'][key]
    return dict(case=case['case'],observed_requests=len(fixtures),inventory_matches=bool(matched),
                eof_and_body_match=bool(wire_complete),controls_valid=bool(control_valid),native_success=native_success,
                accepted_capture=accepted,provider_usage_complete=bool(complete),known_primary_subtotal=subtotal,
                usage=reports,runtime_qualified=False)


def check(bundle):
    expected = {'known':(1,True,True),'zero':(1,True,True),'missing':(1,True,False),'partial':(1,True,False),
                'http-error':(1,False,False),'disconnect':(1,False,False),'cancel':(1,False,False),
                'intent-fail':(0,False,False),'body-fail':(1,False,False),'end-file-fail':(1,False,False),
                'end-dir-fail':(1,False,False),'disabled':(1,False,False),'oversize':(1,False,False)}
    assert bundle['version']==1 and bundle['synthetic_provider'] is True and bundle['paid_calls']==0
    assert hashlib.sha256((HERE/'capture_hook.py').read_bytes()).hexdigest()==bundle['hook_sha256'], 'hook pin'
    assert hashlib.sha256((HERE/'capture_reproduce.py').read_bytes()).hexdigest()==bundle['driver_sha256'], 'driver pin'
    assert len(bundle['cases'])==len(expected) and {c['case'] for c in bundle['cases']}==set(expected), 'case inventory'
    results={}
    for case in bundle['cases']:
        assert case['summary']['hook_sha256']==bundle['hook_sha256'], 'mixed hook versions'
        result=assess(case); results[case['case']]=result
        assert (result['observed_requests'],result['accepted_capture'],result['provider_usage_complete'])==expected[case['case']], case['case']
    assert results['known']['known_primary_subtotal']==dict(prompt_tokens=20,completion_tokens=5)
    assert results['known']['usage'][0]['fields']==dict(prompt_tokens=20,completion_tokens=5,total_tokens=25,cached_tokens=3,reasoning_tokens=2)
    assert results['zero']['usage'][0]['fields']==dict(prompt_tokens=0,completion_tokens=0,total_tokens=0,cached_tokens=0,reasoning_tokens=0)
    assert results['missing']['usage'][0]['fields']=={}
    assert results['partial']['usage'][0]['fields']['completion_tokens'] is None
    for name in ('end-file-fail','end-dir-fail'):
        assert results[name]['eof_and_body_match'] and not results[name]['accepted_capture'], 'EOF metadata alone is insufficient'
    cancel=next(c for c in bundle['cases'] if c['case']=='cancel')
    assert cancel['summary']['cancelled_after_durable_first_chunk'] is True
    assert cancel['fixture'][0]['connection_closed_after_cli_exit'] is False, 'retained cancellation blocker changed'
    retries=bundle['prior_trials'][2]
    assert len(retries['fixture'])==3 and [c['intent']['sdk_retry_header'] for c in retries['captures']]==['0','1','2']
    assert all(c['intent']['sdk_max_retries']==2 for c in retries['captures'])
    assert assess(retries)['accepted_capture'] is False, 'unexpected retries must refuse this profile'
    return results


if __name__=='__main__':
    path=Path(sys.argv[1]) if len(sys.argv)>1 else HERE/'capture-observations.json'
    results=check(usage.strict_json(path.read_text()))
    print(json.dumps({key:{k:v for k,v in row.items() if k!='usage'} for key,row in results.items()},indent=2))
    print('PASS: 13 native fixture cases replay; buffered capture demonstrated, runtime remains unqualified')
