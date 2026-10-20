"""Pinned-dependency offline tests; all HTTP uses MockTransport, no model/network calls."""
import asyncio
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import sys
import tempfile
import unittest

sys.dont_write_bytecode = True
SOURCE = Path(sys.argv.pop(1)) if len(sys.argv) > 1 else Path(__file__).with_name('capture_hook.py')
spec = importlib.util.spec_from_file_location('capture_hook', SOURCE)
m = importlib.util.module_from_spec(spec); spec.loader.exec_module(m)
import httpx


class Stream(httpx.AsyncByteStream):
    def __init__(self, parts, error=None):
        self.parts, self.error, self.closed = parts, error, False

    async def __aiter__(self):
        for part in self.parts:
            yield part
        if self.error:
            raise self.error

    async def aclose(self):
        self.closed = True


class CaptureTests(unittest.IsolatedAsyncioTestCase):
    def setUp(self):
        self.temporary = tempfile.TemporaryDirectory(dir=os.environ['CAPTURE_ROOT'])
        self.root = Path(self.temporary.name)
        os.environ['CAPTURE_ROOT'] = str(self.root)
        self.owner = m.CaptureHook()
        os.environ['CAPTURE_ROOT'] = str(self.root.parent)
        self.identity = 'test-1'

    async def asyncTearDown(self):
        if self.owner.handler is not None:
            await self.owner.handler.close()

    def tearDown(self):
        self.temporary.cleanup()

    def end(self):
        return json.loads((self.root / 'test-1.end.json').read_text())

    async def test_retry_override_preserves_allowlist_and_rejects_bad_shape(self):
        allowed = ['reasoning_effort']
        result = await self.owner.async_pre_call_deployment_hook({'allowed_openai_params':allowed}, None)
        self.assertEqual(result['allowed_openai_params'], ['reasoning_effort', 'max_retries'])
        self.assertEqual(allowed, ['reasoning_effort'])
        self.assertEqual(result['max_retries'], 0)
        self.assertEqual(result['client'].max_retries, 0)
        with self.assertRaises(m.CaptureFailure):
            await self.owner.async_pre_call_deployment_hook({'allowed_openai_params':False}, None)
        self.assertTrue(self.owner.failed)

    async def test_recorded_before_yield_and_close(self):
        original = Stream([b'abc', b'def'])
        stream = m.RecordedStream(original, self.owner, self.identity)
        got = bytearray()
        async for part in stream:
            got.extend(part)
            self.assertEqual((self.root/'test-1.body').read_bytes(), got)
        await stream.aclose()
        self.assertTrue(original.closed)
        self.assertEqual(self.end(), dict(termination='eof', bytes=6, chunks=2, body_sha256=hashlib.sha256(b'abcdef').hexdigest()))

    async def test_chunk_partitions_have_same_body_and_digest(self):
        for width in (1, 2, 3, 7):
            with self.subTest(width=width):
                identity = 'part-' + str(width)
                stream = m.RecordedStream(Stream([b'abcdefg'[i:i+width] for i in range(0,7,width)]), self.owner, identity)
                self.assertEqual(b''.join([part async for part in stream]), b'abcdefg')
                await stream.aclose()
                end = json.loads((self.root/(identity+'.end.json')).read_text())
                self.assertEqual(end['body_sha256'], hashlib.sha256(b'abcdefg').hexdigest())
                self.assertEqual(end['termination'], 'eof')

    async def test_close_cancel_and_transport_error_are_incomplete(self):
        for name, error in [('cancel', asyncio.CancelledError()), ('transport', httpx.ReadError('fixture'))]:
            stream = m.RecordedStream(Stream([b'prefix'], error), self.owner, name)
            with self.assertRaises(type(error)):
                async for _ in stream: pass
            await stream.aclose()
            end = json.loads((self.root/(name+'.end.json')).read_text())
            self.assertEqual(end['termination'], 'cancelled' if name=='cancel' else 'transport_error')
            self.assertEqual((self.root/(name+'.body')).read_bytes(), b'prefix')
        stream = m.RecordedStream(Stream([b'not-consumed']), self.owner, 'close')
        await stream.aclose()
        self.assertEqual(json.loads((self.root/'close.end.json').read_text())['termination'], 'closed')

    async def test_size_and_chunk_limits_before_yield(self):
        for name, parts in [('bytes', [b'x' * m.MAX_BYTES, b'y']), ('chunks', [b'x'] * (m.MAX_CHUNKS + 1))]:
            stream = m.RecordedStream(Stream(parts), self.owner, name)
            received = bytearray()
            with self.assertRaisesRegex(m.CaptureFailure, 'limit'):
                async for part in stream: received.extend(part)
            await stream.aclose()
            self.assertEqual(len(received), m.MAX_BYTES if name=='bytes' else m.MAX_CHUNKS)
            self.assertTrue(self.owner.failed)
            self.assertEqual(json.loads((self.root/(name+'.end.json')).read_text())['termination'], 'capture_error')

    async def test_intent_before_dispatch_and_endpoint_guard(self):
        request = httpx.Request('POST', self.owner.endpoint, content=b'{}')
        await self.owner.request(request)
        identity = request.headers['x-ateam-exchange-id']
        intent = json.loads((self.root/(identity+'.intent.json')).read_text())
        self.assertEqual(intent['request_sha256'], hashlib.sha256(b'{}').hexdigest())
        self.assertEqual(request.extensions['ateam_exchange_id'], identity)
        self.assertEqual(request.headers['accept-encoding'], 'identity')
        with self.assertRaisesRegex(m.CaptureFailure, 'destination'):
            await self.owner.request(httpx.Request('POST', 'http://127.0.0.1:1/unexpected'))
        self.assertTrue(self.owner.failed)

    async def test_request_limit_and_fatal_latch(self):
        for _ in range(8):
            await self.owner.request(httpx.Request('POST', self.owner.endpoint, content=b'{}'))
        with self.assertRaisesRegex(m.CaptureFailure, 'request limit'):
            await self.owner.request(httpx.Request('POST', self.owner.endpoint, content=b'{}'))
        with self.assertRaisesRegex(m.CaptureFailure, 'already failed'):
            await self.owner.request(httpx.Request('POST', self.owner.endpoint, content=b'{}'))
        self.assertEqual(len(list(self.root.glob('*.intent.json'))), 8)

    async def test_faults_prevent_success_even_with_valid_eof_json(self):
        for fault in ('body', 'end-file-sync', 'end-dir-sync'):
            owner = self.owner
            owner.fault = fault; owner.failed = False
            identity = fault
            stream = m.RecordedStream(Stream([b'whole-body']), owner, identity)
            with self.assertRaises(m.CaptureFailure):
                async for _ in stream: pass
            await stream.aclose()
            self.assertTrue(owner.failed)
            self.assertTrue((self.root/'failure.json').exists())
            end = json.loads((self.root/(identity+'.end.json')).read_text())
            self.assertEqual(end['termination'], 'capture_error' if fault=='body' else 'eof')
        self.owner.fault = 'intent'; self.owner.failed = False
        with self.assertRaisesRegex(m.CaptureFailure, 'intent failure'):
            await self.owner.request(httpx.Request('POST', self.owner.endpoint, content=b'{}'))
        self.assertEqual(len(list(self.root.glob('*.intent.json'))), 0)

    async def test_buffered_httpx_hook_records_eof_before_consumer(self):
        async def transport(request):
            return httpx.Response(200, stream=Stream([b'data: [DONE]\n\n', b': trailing\n\n']))
        async with httpx.AsyncClient(transport=httpx.MockTransport(transport),
                                    event_hooks={'request':[self.owner.request], 'response':[self.owner.response]}) as client:
            async with client.stream('POST', self.owner.endpoint, content=b'{}') as response:
                identity = response.request.headers['x-ateam-exchange-id']
                self.assertTrue((self.root/(identity+'.end.json')).exists(), 'EOF must precede consumer')
                self.assertEqual(json.loads((self.root/(identity+'.end.json')).read_text())['termination'], 'eof')
                self.assertEqual(await response.aread(), b'data: [DONE]\n\n: trailing\n\n')

    async def test_httpx_failure_and_encoding_refusal(self):
        for status, headers in [(429, {}), (200, {'content-encoding':'gzip'})]:
            self.owner.failed = False
            async def transport(request):
                return httpx.Response(status, headers=headers, stream=Stream([b'error']))
            async with httpx.AsyncClient(transport=httpx.MockTransport(transport), event_hooks={'request':[self.owner.request], 'response':[self.owner.response]}) as client:
                if status==429:
                    response = await client.post(self.owner.endpoint, content=b'{}')
                    self.assertEqual(response.content, b'error')
                    self.assertEqual(json.loads(next(self.root.glob('*.end.json')).read_text())['termination'], 'eof')
                else:
                    with self.assertRaises(m.CaptureFailure):
                        await client.post(self.owner.endpoint, content=b'{}')
                    self.assertTrue(self.owner.failed)


if __name__ == '__main__':
    unittest.main(verbosity=2)
