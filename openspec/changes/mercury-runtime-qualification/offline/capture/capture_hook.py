"""Opt-in research instrumentation using LiteLLM/HTTPX public hooks; no agent loop."""
import asyncio
import hashlib
import json
import os
from pathlib import Path
import re

import httpx
from litellm.integrations.custom_logger import CustomLogger
from openai import AsyncOpenAI

# // shortcut: bounded buffering is an offline accounting instrument; qualify streaming before production integration.
MAX_BYTES = 1_048_576
MAX_CHUNKS = 4096


class CaptureFailure(RuntimeError):
    """Distinct storage failure; explicit SDK retry control and the fatal latch are also required."""


def record(path, data, fault=""):
    with path.open('x') as out:
        json.dump(data, out, sort_keys=True)
        out.flush()
        if fault == 'end-file-sync':
            raise OSError('injected completion file fsync failure')
        os.fsync(out.fileno())
    descriptor = os.open(path.parent, os.O_RDONLY)
    try:
        if fault == 'end-dir-sync':
            raise OSError('injected completion directory fsync failure')
        os.fsync(descriptor)
    finally:
        os.close(descriptor)


class RecordedStream(httpx.AsyncByteStream):
    def __init__(self, original, owner, identity):
        self.original, self.owner, self.identity = original, owner, identity
        self.size, self.chunks, self.finished = 0, 0, False
        self.digest = hashlib.sha256()
        self.output = (owner.root / (identity + '.body')).open('xb')

    def finish(self, termination):
        if self.finished:
            return
        self.output.flush()
        os.fsync(self.output.fileno())
        record(self.owner.root / (self.identity + '.end.json'),
               dict(termination=termination, bytes=self.size, chunks=self.chunks,
                    body_sha256=self.digest.hexdigest()), fault=self.owner.fault if termination == "eof" else "")
        self.finished = True

    async def __aiter__(self):
        try:
            async for chunk in self.original:
                if self.size + len(chunk) > MAX_BYTES or self.chunks + 1 > MAX_CHUNKS:
                    raise CaptureFailure('capture size/chunk limit')
                if self.owner.fault == 'body':
                    raise CaptureFailure('injected body write failure')
                self.output.write(chunk)
                self.output.flush()
                os.fsync(self.output.fileno())
                self.digest.update(chunk)
                self.size += len(chunk)
                self.chunks += 1
                if self.chunks == 1:
                    record(self.owner.root / (self.identity + '.first.json'), dict(bytes=self.size))
                yield chunk
            self.finish('eof')
        except BaseException as error:
            if isinstance(error, asyncio.CancelledError):
                reason = 'cancelled'
            elif isinstance(error, (CaptureFailure, OSError)):
                reason = 'capture_error'
                self.owner.fail('capture_error')
            else:
                reason = 'transport_error'
            try:
                self.finish(reason)
            except OSError:
                self.owner.fail('capture_error')
            if isinstance(error, OSError) and not isinstance(error, httpx.HTTPError):
                raise CaptureFailure('capture storage failure') from error
            raise
        finally:
            self.output.close()

    async def aclose(self):
        try:
            if not self.finished and not self.output.closed:
                self.finish('closed')
        except OSError as error:
            self.owner.fail('capture_error')
            raise CaptureFailure('capture close failure') from error
        finally:
            self.output.close()
            await self.original.aclose()


class CaptureHook(CustomLogger):
    def __init__(self):
        self.root = Path(os.environ['CAPTURE_ROOT']).resolve(strict=True)
        self.attempt = os.environ['CAPTURE_ATTEMPT']
        if not re.fullmatch(r'[a-z0-9-]{1,40}', self.attempt):
            raise ValueError('invalid attempt ID')
        self.endpoint = os.environ['CAPTURE_ENDPOINT']
        self.fault = os.environ.get('CAPTURE_FAULT', '')
        if self.fault not in ('', 'intent', 'body', 'end-file-sync', 'end-dir-sync'):
            raise ValueError('invalid fault control')
        self.sequence, self.deployments = 0, 0
        self.failed = False
        self.handler = None

    def fail(self, reason):
        self.failed = True
        try:
            record(self.root / 'failure.json', dict(reason=reason))
        except OSError:
            pass  # Capture failure must still escape if storage cannot retain its marker.

    async def request(self, request):
        if self.failed:
            raise CaptureFailure('capture already failed')
        if request.method != 'POST' or str(request.url) != self.endpoint:
            self.fail('capture_error')
            raise CaptureFailure('unexpected HTTP destination')
        if self.sequence >= 8:
            self.fail('capture_error')
            raise CaptureFailure('capture request limit')
        self.sequence += 1
        identity = self.attempt + '-' + str(self.sequence)
        request.headers['x-ateam-exchange-id'] = identity
        request.headers['accept-encoding'] = 'identity'
        request.extensions['ateam_exchange_id'] = identity
        try:
            if self.fault == 'intent':
                raise OSError('injected intent write failure')
            record(self.root / (identity + '.intent.json'),
                   dict(attempt_id=self.attempt, exchange_id=identity, method=request.method,
                        request_sha256=hashlib.sha256(request.content).hexdigest(),
                        sdk_max_retries=None if self.handler is None else self.handler.max_retries,
                        sdk_retry_header=request.headers.get('x-stainless-retry-count')))
        except OSError as error:
            self.fail('capture_error')
            raise CaptureFailure('capture intent failure') from error

    async def response(self, response):
        identity = response.request.extensions['ateam_exchange_id']
        try:
            encoding = response.headers.get('content-encoding', 'identity')
            if encoding != 'identity':
                raise CaptureFailure('encoded response unsupported')
            record(self.root / (identity + '.headers.json'),
                   dict(status=response.status_code, content_type=response.headers.get('content-type')))
            response.stream = RecordedStream(response.stream, self, identity)
            # Bound and durably record through the wrapper before the SDK can stop at DONE.
            await response.aread()
        except (OSError, CaptureFailure) as error:
            self.fail('capture_error')
            raise CaptureFailure('capture response failure') from error

    async def async_pre_call_deployment_hook(self, kwargs, call_type):
        if self.failed:
            raise CaptureFailure('capture already failed')
        self.deployments += 1
        record(self.root / ('deployment-' + str(self.deployments) + '.json'),
               dict(attempt_id=self.attempt, call_type=str(call_type)))
        if self.handler is None:
            self.handler = AsyncOpenAI(api_key='offline-provider-fixture', base_url=self.endpoint.removesuffix('/chat/completions'), max_retries=0,
                                       http_client=httpx.AsyncClient(timeout=10, trust_env=False, follow_redirects=False,
                                                                     event_hooks={'request': [self.request], 'response': [self.response]}))
        kwargs['client'] = self.handler
        allowed = kwargs.get('allowed_openai_params')
        if allowed is None:
            allowed = []
        if type(allowed) is not list or any(type(value) is not str for value in allowed):
            self.fail('invalid_allowed_params')
            raise CaptureFailure('invalid allowed parameter list')
        kwargs['allowed_openai_params'] = allowed + ([] if 'max_retries' in allowed else ['max_retries'])
        kwargs['max_retries'] = 0
        kwargs['timeout'] = 10
        return kwargs


capture = CaptureHook()
