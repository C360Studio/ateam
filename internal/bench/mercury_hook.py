"""Pinned LiteLLM callback: capture provider bytes; accounting belongs to Go."""
import json
import os

import httpx
from litellm.integrations.custom_logger import CustomLogger
from openai import AsyncOpenAI

LIMIT = 1024 * 1024


class BoundedStream(httpx.AsyncByteStream):
    def __init__(self, source, owner):
        self.source, self.owner = source, owner

    async def __aiter__(self):
        async for chunk in self.source:
            self.owner.body_bytes += len(chunk)
            if self.owner.body_bytes > LIMIT:
                raise ValueError("body_limit")
            yield chunk

    async def aclose(self):
        await self.source.aclose()


class Capture(CustomLogger):
    def __init__(self):
        self.output = open(os.environ["MERCURY_CAPTURE_PATH"], "x", encoding="utf-8")
        self.base = os.environ["MERCURY_API_BASE"].rstrip("/")
        self.sequence = self.written = self.body_bytes = 0
        self.failed, self.client = False, None

    def record(self, record):
        encoded = json.dumps(record, ensure_ascii=True, separators=(",", ":")) + "\n"
        self.written += len(encoded)
        if self.written > LIMIT:
            raise ValueError("journal_limit")
        self.output.write(encoded)
        self.output.flush()

    def fail(self):
        self.failed = True
        try:
            self.record(dict(type="error", error="capture_failed"))
        except Exception:
            pass  # Missing response also fails Go accounting if the journal cannot be written.

    async def request(self, request):
        try:
            if self.failed or self.sequence >= 4 or request.method != "POST" or str(request.url) != self.base + "/chat/completions":
                raise ValueError("request_guard")
            data = json.loads(request.content)
            retry = int(request.headers.get("x-stainless-retry-count", "-1"))
            if (data.get("model") != "mercury-3" or data.get("reasoning_effort") != "low"
                    or type(data.get("max_tokens")) is not int or data["max_tokens"] != 8192
                    or data.get("stream") is not True or data.get("stream_options", {}).get("include_usage") is not True
                    or retry != 0 or self.client.max_retries != 0):
                raise ValueError("request_controls")
            self.sequence += 1
            request.extensions["mercury_id"] = self.sequence
            request.headers["accept-encoding"] = "identity"
            self.record(dict(type="intent", id=self.sequence, model=data.get("model"),
                             reasoning_effort=data.get("reasoning_effort"), max_tokens=data.get("max_tokens"),
                             stream=data.get("stream"), include_usage=data.get("stream_options", {}).get("include_usage"),
                             retry_count=retry))
        except Exception:
            self.fail()
            raise

    async def response(self, response):
        try:
            if response.headers.get("content-encoding", "identity") != "identity":
                raise ValueError("encoded_response")
            response.stream = BoundedStream(response.stream, self)
            # Buffer before SDK translation/DONE handling; stream timing is not benchmark evidence.
            body = await response.aread()
            self.record(dict(type="response", id=response.request.extensions["mercury_id"],
                             status=response.status_code, eof=True, body=body.decode("utf-8")))
        except BaseException:
            self.fail()
            raise

    async def async_pre_call_deployment_hook(self, kwargs, call_type):
        if self.failed:
            raise ValueError("capture_failed")
        if self.client is None:
            self.client = AsyncOpenAI(api_key=os.environ["INCEPTION_API_KEY"], base_url=self.base, max_retries=0,
                                      http_client=httpx.AsyncClient(trust_env=False, follow_redirects=False,
                                                                   event_hooks={"request": [self.request], "response": [self.response]}))
        allowed = kwargs.get("allowed_openai_params")
        if allowed is None:
            allowed = []
        if type(allowed) is not list or any(type(item) is not str for item in allowed):
            self.fail()
            raise ValueError("invalid_allowed_params")
        kwargs["allowed_openai_params"] = allowed + ([] if "max_retries" in allowed else ["max_retries"])
        kwargs["client"], kwargs["max_retries"] = self.client, 0
        return kwargs


capture = Capture()
