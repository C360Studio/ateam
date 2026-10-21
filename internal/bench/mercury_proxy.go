package bench

import (
	"bytes"
	"context"
	"crypto/subtle"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

// startMercuryProxy keeps the provider credential in Go and records bytes before
// LiteLLM can translate usage. Buffered delivery does not measure stream timing.
func startMercuryProxy(ctx context.Context, apiBase, key, token, capturePath string) (string, func() error, error) {
	endpoint, err := url.Parse(strings.TrimRight(apiBase, "/") + "/chat/completions")
	if err != nil || endpoint.Host == "" || endpoint.User != nil || endpoint.RawQuery != "" || endpoint.Fragment != "" || (endpoint.Scheme != "https" && endpoint.Scheme != "http") || key == "" || token == "" {
		return "", nil, fmt.Errorf("invalid Mercury proxy configuration")
	}
	journal, err := os.OpenFile(capturePath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return "", nil, fmt.Errorf("create Mercury capture: %w", err)
	}
	return serveMercuryProxy(ctx, endpoint, key, token, journal)
}

func serveMercuryProxy(ctx context.Context, endpoint *url.URL, key, token string, journal *os.File) (string, func() error, error) {
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		_ = journal.Close()
		return "", nil, fmt.Errorf("listen Mercury capture: %w", err)
	}
	owned, cancel := context.WithCancel(ctx)
	transport := &http.Transport{Proxy: nil, DisableKeepAlives: true, ForceAttemptHTTP2: false, TLSNextProto: map[string]func(string, *tls.Conn) http.RoundTripper{}}
	client := &http.Client{Transport: transport, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	var serial, admission sync.Mutex
	var handlers sync.WaitGroup
	stopped, failed := false, false
	sequence, written, bodyBytes := 0, 0, 0
	// Leave room for an error after a complete exchange, so a later bad request
	// cannot make the earlier usage appear to describe a successful attempt.
	failureLine := []byte("{\"type\":\"error\",\"error\":\"capture_failed\"}\n")
	writeRecord := func(record mercuryRecord) error {
		data, encodeErr := json.Marshal(record)
		if encodeErr != nil {
			return encodeErr
		}
		data = append(data, '\n')
		if written+len(data) > maxAdapterBytes-len(failureLine) {
			return fmt.Errorf("Mercury journal limit")
		}
		n, writeErr := journal.Write(data)
		written += n
		if writeErr != nil {
			return writeErr
		}
		if n != len(data) {
			return io.ErrShortWrite
		}
		return nil
	}
	fail := func() {
		if failed {
			return
		}
		failed = true
		if n, writeErr := journal.Write(failureLine); writeErr != nil || n != len(failureLine) {
			// Best effort: stop also reports failure independently of these bytes.
			_ = journal.Truncate(0)
		}
	}
	server := &http.Server{ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, MaxHeaderBytes: 8192}
	server.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		admission.Lock()
		if stopped {
			admission.Unlock()
			http.Error(w, "capture stopped", http.StatusServiceUnavailable)
			return
		}
		handlers.Add(1)
		admission.Unlock()
		defer handlers.Done()
		serial.Lock()
		defer serial.Unlock()
		reject := func() { fail(); http.Error(w, "capture failed", http.StatusBadGateway) }
		if r.Method == http.MethodHead && r.URL.Path == "/" && r.URL.RawQuery == "" {
			w.WriteHeader(http.StatusOK)
			return
		}
		if failed || owned.Err() != nil {
			reject()
			return
		}
		if r.Method != "POST" || r.URL.Path != "/v1/chat/completions" || r.URL.RawQuery != "" || r.URL.Fragment != "" || subtle.ConstantTimeCompare([]byte(r.Header.Get("Authorization")), []byte("Bearer "+token)) != 1 || r.Header.Get("X-Stainless-Retry-Count") != "0" || sequence >= 4 {
			reject()
			return
		}
		body, readErr := io.ReadAll(io.LimitReader(r.Body, maxAdapterBytes+1))
		if readErr != nil || len(body) > maxAdapterBytes || uniqueJSON(body) != nil {
			reject()
			return
		}
		// Maps preserve JSON's case-sensitive field names and reject aliases instead
		// of inheriting encoding/json's case-insensitive struct-field matching.
		var fields map[string]json.RawMessage
		var model, effort string
		var maxTokens int
		var stream, include bool
		var options map[string]json.RawMessage
		if json.Unmarshal(body, &fields) != nil || json.Unmarshal(fields["model"], &model) != nil || json.Unmarshal(fields["reasoning_effort"], &effort) != nil || json.Unmarshal(fields["max_tokens"], &maxTokens) != nil || json.Unmarshal(fields["stream"], &stream) != nil || json.Unmarshal(fields["stream_options"], &options) != nil || json.Unmarshal(options["include_usage"], &include) != nil || fields["max_completion_tokens"] != nil || fields["max_retries"] != nil || fields["allowed_openai_params"] != nil || model != "mercury-3" || effort != "low" || maxTokens != 8192 || !stream || !include {
			reject()
			return
		}
		sequence++
		retry := 0
		if writeRecord(mercuryRecord{Type: "intent", ID: sequence, Model: model, Effort: effort, MaxTokens: maxTokens, Stream: stream, IncludeUsage: include, Retries: &retry}) != nil {
			reject()
			return
		}
		exchange, cancelExchange := context.WithCancel(owned)
		stopDisconnect := context.AfterFunc(r.Context(), cancelExchange)
		defer func() { stopDisconnect(); cancelExchange() }()
		outbound, requestErr := http.NewRequestWithContext(exchange, http.MethodPost, endpoint.String(), bytes.NewReader(body))
		if requestErr != nil {
			reject()
			return
		}
		outbound.Header.Set("Authorization", "Bearer "+key)
		outbound.Header.Set("Content-Type", "application/json")
		outbound.Header.Set("Accept-Encoding", "identity")
		response, requestErr := client.Do(outbound)
		if requestErr != nil {
			reject()
			return
		}
		defer func() { _ = response.Body.Close() }()
		responseBody, readErr := io.ReadAll(io.LimitReader(response.Body, int64(maxAdapterBytes-bodyBytes+1)))
		bodyBytes += len(responseBody)
		if bodyBytes > maxAdapterBytes {
			reject()
			return
		}
		if writeRecord(mercuryRecord{Type: "response", ID: sequence, Status: response.StatusCode, EOF: readErr == nil, Body: string(responseBody)}) != nil {
			reject()
			return
		}
		encoding := response.Header.Get("Content-Encoding")
		if readErr != nil || response.StatusCode != http.StatusOK || (encoding != "" && encoding != "identity") {
			reject()
			return
		}
		if _, parseErr := parseMercurySSE(string(responseBody)); parseErr != nil {
			reject()
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		if _, writeErr := w.Write(responseBody); writeErr != nil {
			fail()
		}
	})
	served := make(chan struct{})
	go func() { defer close(served); _ = server.Serve(listener) }()
	var once sync.Once
	var stopErr error
	stop := func() error {
		once.Do(func() {
			cancel()
			admission.Lock()
			stopped = true
			admission.Unlock()
			_ = server.Close()
			handlers.Wait()
			<-served
			transport.CloseIdleConnections()
			if failed {
				stopErr = fmt.Errorf("Mercury capture failed")
			}
			if closeErr := journal.Close(); closeErr != nil {
				stopErr = errors.Join(stopErr, fmt.Errorf("close Mercury capture: %w", closeErr))
			}
		})
		return stopErr
	}
	return "http://" + listener.Addr().String() + "/v1", stop, nil
}
