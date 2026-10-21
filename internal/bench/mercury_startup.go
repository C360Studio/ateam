package bench

import (
	"bytes"
	"strconv"
	"strings"
)

// Composition is intentional: limitedBuffer.ReadFrom must not bypass Write.
// os/exec serializes stdout/stderr writes when they share this same writer.
type mercuryBridgeOutput struct {
	captured  limitedBuffer
	address   chan string
	scanned   int
	announced bool
}

func (b *mercuryBridgeOutput) Write(p []byte) (int, error) {
	n, err := b.captured.Write(p)
	for !b.announced {
		line, _, complete := bytes.Cut(b.captured.Bytes()[b.scanned:], []byte{'\n'})
		if !complete {
			break
		}
		b.scanned += len(line) + 1
		if address := mercuryStartupAddress(string(line)); address != "" {
			b.announced = true
			select {
			case b.address <- address:
			default:
			}
		}
	}
	return n, err
}

// This is the startup line emitted by the pinned LiteLLM/Uvicorn child after
// binding. Unknown formats fail closed under the existing attempt deadline.
func mercuryStartupAddress(line string) string {
	line = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "INFO:"))
	port, ok := strings.CutPrefix(line, "Uvicorn running on http://127.0.0.1:")
	if !ok {
		return ""
	}
	port, ok = strings.CutSuffix(port, " (Press CTRL+C to quit)")
	if !ok {
		return ""
	}
	number, err := strconv.Atoi(port)
	if err != nil || number < 1 || number > 65535 || strconv.Itoa(number) != port {
		return ""
	}
	return "http://127.0.0.1:" + port
}
