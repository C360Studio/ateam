"""Check this finite synthetic evidence set; passing means the blocker reproduces."""

import json
import pathlib
import sys


def events(text):
    """Read the data lines of these single-line JSON SSE fixtures."""
    return [
        json.loads(line[6:])
        for line in text.splitlines()
        if line.startswith("data: ") and line != "data: [DONE]"
    ]


def check(document):
    assert document["synthetic_provider"] is True, "not synthetic evidence"
    assert document["paid_calls"] == 0, "unexpected paid calls"
    cases = document["cases"]
    assert [case["case"] for case in cases] == ["low", "high", "missing"], "case set"
    for case in cases:
        name = case["case"]
        expected_effort = "high" if name == "high" else "low"
        assert case["requested_effort"] == expected_effort, "requested effort"
        assert case["messages_request"]["thinking"]["type"] == "adaptive", "thinking shape"
        assert case["messages_request"]["output_config"]["effort"] == expected_effort, "incoming effort"
        assert case["provider_request"]["reasoning_effort"] == expected_effort, "forwarded effort"
        assert case["messages_request"]["max_tokens"] == 1024, "incoming output allowance"
        assert case["provider_request"]["max_tokens"] == 1024, "forwarded output allowance"
        assert "max_completion_tokens" not in case["provider_request"], "output field shape"
        assert case["messages_request"]["model"] == "mercury", "CLI alias"
        assert case["provider_request"]["model"] == "mercury-3", "provider model"
        assert case["instruction_sentinel_at_both_boundaries"] is True, "instruction delivery"
        assert case["messages_count"] == case["provider_count"] == 1, "request count"
        assert case["cli_exit"] == 0 and case["bridge_http_status"] == 200, "execution failed"

        provider = events(case["provider_sse"])
        bridge = events(case["bridge_sse"])
        deltas = [item for item in bridge if item.get("type") == "message_delta"]
        assert len(deltas) == 1, "terminal bridge count"
        bridge_usage = deltas[0]["usage"]
        terminal = case["cli_terminal"]
        assert terminal["type"] == "result" and terminal["subtype"] == "success", "CLI status"
        assert terminal["is_error"] is False and terminal["num_turns"] == 1, "CLI completion"
        assert terminal["result"] == "offline-fixture-ok", "fixture response"
        models = terminal["modelUsage"]
        assert list(models) == ["mercury"], "CLI model buckets"
        usage = models["mercury"]
        assert usage["provider"] == "firstParty" and usage["costBasis"] == "unknown", "CLI provenance shape"

        if name == "missing":
            assert len(provider) == 2, "missing fixture framing"
            assert all("usage" not in item for item in provider), "missing fixture gained usage"
            assert bridge_usage == {"input_tokens": 26, "output_tokens": 4}, "fallback counters changed"
            assert usage["inputTokens"] == 26 and usage["outputTokens"] == 4, "CLI fallback counters changed"
            assert usage["cacheReadInputTokens"] == usage["cacheCreationInputTokens"] == 0, "fallback subsets"
        else:
            assert len(provider) == 3 and provider[-1]["choices"] == [], "known fixture framing"
            assert provider[-1]["usage"] == {
                "prompt_tokens": 20,
                "completion_tokens": 5,
                "total_tokens": 25,
                "prompt_tokens_details": {"cached_tokens": 3},
                "completion_tokens_details": {"reasoning_tokens": 2},
            }, "provider oracle changed"
            assert bridge_usage == {
                "input_tokens": 17,
                "output_tokens": 5,
                "cache_read_input_tokens": 3,
            }, "bridge known counters changed"
            assert usage["inputTokens"] == 17 and usage["cacheReadInputTokens"] == 3, "CLI known input"
            assert usage["outputTokens"] == 5, "CLI known output"
        assert usage["thinkingTokens"] == 0, "CLI reasoning projection changed"
    assert document["status"] == "blocked-provider-usage-provenance", "wrong qualification verdict"


if __name__ == "__main__":
    source = pathlib.Path(sys.argv[1]) if len(sys.argv) > 1 else pathlib.Path(__file__).with_name("observations.json")
    check(json.loads(source.read_text()))
    print("PASS: retained evidence reproduces; provider-usage contract remains BLOCKED")
