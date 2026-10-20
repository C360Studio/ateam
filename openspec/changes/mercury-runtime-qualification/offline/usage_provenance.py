"""Offline research contract for supplied Chat Completions wire evidence, not a capture tool."""
import hashlib
import json

MAX_BYTES = 1_048_576
MAX_EXCHANGES = 256
MAX_COUNT = 2**53 - 1


def strict_json(text):
    def pairs(items):
        result = {}
        for key, value in items:
            if key in result:
                raise ValueError('duplicate JSON key')
            result[key] = value
        return result

    def constant(_):
        raise ValueError('nonfinite JSON number')

    return json.loads(text, object_pairs_hook=pairs, parse_constant=constant)


def count(obj, key):
    value = obj.get(key)
    if value is not None and (type(value) is not int or not 0 <= value <= MAX_COUNT):
        raise ValueError('invalid token count: ' + key)
    return value


def usage_fields(raw):
    if type(raw) is not dict:
        raise ValueError('usage is not an object')
    values = {key: count(raw, key) for key in ('prompt_tokens', 'completion_tokens', 'total_tokens')}
    for outer, inner in [('prompt_tokens_details', 'cached_tokens'),
                         ('completion_tokens_details', 'reasoning_tokens')]:
        detail = raw.get(outer)
        if detail is not None and type(detail) is not dict:
            raise ValueError('invalid detail object')
        values[inner] = count(detail or {}, inner)
    p, c, t = (values[k] for k in ('prompt_tokens', 'completion_tokens', 'total_tokens'))
    if t is not None and any(v is not None and v > t for v in (p, c)):
        raise ValueError('primary exceeds total')
    if p is not None and c is not None and t is not None and p + c != t:
        raise ValueError('inconsistent total')
    for subset, primary in [('cached_tokens', p), ('reasoning_tokens', c)]:
        if primary is not None and values[subset] is not None and values[subset] > primary:
            raise ValueError('subset exceeds primary')
    return values


def inspect_exchange(exchange, expected_model='mercury-3'):
    """Inspect a bounded, complete supplied body; retain counts even on incomplete transport.

    Supported profile: UTF-8 SSE, LF/CRLF, one JSON data line per event,
    one choice index 0, one finish then one optional usage-only event, then DONE.
    Repeated usage is ambiguous even when identical. No cumulative/delta guessing.
    """
    keys = {'exchange_id', 'attempt_id', 'body', 'body_sha256', 'http_status', 'termination'}
    if type(exchange) is not dict or set(exchange) != keys:
        raise ValueError('invalid exchange schema')
    body = exchange['body']
    if type(body) is not str or len(body.encode('utf-8')) > MAX_BYTES:
        raise ValueError('body must be bounded UTF-8 text')
    digest = hashlib.sha256(body.encode('utf-8')).hexdigest()
    if digest != exchange['body_sha256']:
        raise ValueError('body hash mismatch')
    status, end = exchange['http_status'], exchange['termination']
    if type(status) is not int or not 100 <= status <= 599:
        raise ValueError('invalid HTTP status')
    if end not in ('eof', 'cancelled', 'timeout', 'disconnect'):
        raise ValueError('invalid termination')
    result = dict(source='supplied-provider-wire', body_sha256=digest, http_status=status,
                  termination=end, usage_state='missing', raw_usage=[], fields={},
                  done=False, primary_complete=False, details_complete=False, error=None)
    if status != 200:
        result['error'] = 'non-200 response; error-body usage is unsupported'
        return result
    finished = False
    response_id = None
    saw_usage = False
    try:
        text = body.replace('\r\n', '\n')
        if '\r' in text:
            raise ValueError('unsupported bare CR framing')
        frames = text.split('\n\n')
        if len(frames) > 4096:
            raise ValueError('too many SSE frames')
        # The final unterminated fragment cannot be treated as a dispatched SSE event.
        trailing = frames.pop()
        for frame in frames:
            if not frame or all(line.startswith(':') for line in frame.split('\n')):
                continue
            if result['done']:
                raise ValueError('data after DONE')
            if '\n' in frame or not frame.startswith('data: '):
                raise ValueError('unsupported SSE frame')
            data = frame[6:]
            if data == '[DONE]':
                if not finished:
                    raise ValueError('DONE before finish')
                result['done'] = True
                continue
            chunk = strict_json(data)
            if type(chunk) is not dict:
                raise ValueError('chunk is not an object')
            identity = chunk.get('id')
            if type(identity) is not str or not identity or chunk.get('model') != expected_model:
                raise ValueError('missing or unexpected response identity')
            if response_id is not None and response_id != identity:
                raise ValueError('conflicting response IDs')
            response_id = identity
            if chunk.get('object') != 'chat.completion.chunk' or 'error' in chunk:
                raise ValueError('unsupported chunk kind')
            choices = chunk.get('choices')
            if type(choices) is not list:
                raise ValueError('invalid choices')
            raw = chunk.get('usage')
            if raw is not None:
                result['raw_usage'].append(raw)
                if saw_usage or not finished or choices:
                    raise ValueError('ambiguous or misplaced usage')
                saw_usage = True
                result['fields'] = usage_fields(raw)
                result['usage_state'] = 'reported'
            elif choices:
                if finished or len(choices) != 1:
                    raise ValueError('unexpected choice after finish or multiple choices')
                choice = choices[0]
                if type(choice) is not dict or type(choice.get('index')) is not int or choice['index'] != 0:
                    raise ValueError('invalid choice index')
                if type(choice.get('delta')) is not dict:
                    raise ValueError('invalid delta')
                reason = choice.get('finish_reason')
                if reason not in (None, 'stop', 'length', 'tool_calls'):
                    raise ValueError('unsupported finish reason')
                finished = reason is not None
            else:
                raise ValueError('empty metadata event unsupported')
        if trailing:
            raise ValueError('unterminated SSE event')
        fields = result['fields']
        result['primary_complete'] = (result['done'] and end == 'eof' and
                                      fields.get('prompt_tokens') is not None and
                                      fields.get('completion_tokens') is not None)
        result['details_complete'] = (result['primary_complete'] and
                                      fields.get('cached_tokens') is not None and
                                      fields.get('reasoning_tokens') is not None)
    except (ValueError, RecursionError) as error:
        result['usage_state'] = 'invalid'
        result['error'] = str(error)
    result['response_id'] = response_id
    return result


def inspect_history(exchanges):
    """No exchange is deduplicated by provider response ID, including retries.

    Completeness is only for the supplied inventory, whose authenticity and
    exhaustiveness must be proved separately by a future capture integration.
    """
    if type(exchanges) is not list or not 0 < len(exchanges) <= MAX_EXCHANGES:
        raise ValueError('history must contain 1..256 exchanges')
    seen = set()
    rows = []
    subtotal = {'prompt_tokens': 0, 'completion_tokens': 0}
    for exchange in exchanges:
        if type(exchange) is not dict:
            raise ValueError('exchange must be an object')
        for key in ('exchange_id', 'attempt_id'):
            if type(exchange.get(key)) is not str or not exchange[key] or len(exchange[key]) > 128:
                raise ValueError('invalid capture identity')
        identity = exchange['exchange_id']
        if identity in seen:
            raise ValueError('duplicate capture exchange ID')
        seen.add(identity)
        row = inspect_exchange(exchange)
        row.update(exchange_id=identity, attempt_id=exchange['attempt_id'])
        rows.append(row)
        if row['primary_complete']:
            for key in subtotal:
                subtotal[key] += row['fields'][key]
    return dict(exchanges=rows, known_primary_subtotal=subtotal,
                supplied_inventory_complete=all(row['primary_complete'] for row in rows),
                unknown_exchange_ids=[row['exchange_id'] for row in rows if not row['primary_complete']],
                runtime_qualified=False)
