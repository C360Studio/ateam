package web

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// testBoardHost is the Host a browser sends to the board when it is opened at its default address.
const testBoardHost = "127.0.0.1:8420"

// testBoardURL is a request target httptest reads the loopback Host from, for tests that build many requests.
const testBoardURL = "http://" + testBoardHost

func getState(h http.Handler, host, origin string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/api/state", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	req.Host = host
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

// The correct behavior: with no token, the board answers a loopback peer only when the browser addressed it by a
// loopback name. The wrong behavior to catch: a page at attacker.example whose DNS name was re-pointed at 127.0.0.1
// reaches the board from a loopback peer with Host and Origin both naming attacker.example, so the Host must be
// checked, not only compared with the Origin.
//
// spec: openspec/specs/harness/spec.md#Requirement: Board state and authenticated API access
func TestTokenlessBoardAnswersOnlyLoopbackHosts(t *testing.T) {
	t.Setenv("ATEAM_API_TOKEN", "")
	hosts := []struct {
		host string
		want int
	}{
		{"127.0.0.1:8420", http.StatusOK},
		{"127.0.0.1", http.StatusOK},
		{"localhost:8420", http.StatusOK},
		{"localhost", http.StatusOK},
		{"LocalHost:8420", http.StatusOK},
		{"[::1]:8420", http.StatusOK},
		{"[::1]", http.StatusOK},
		{"attacker.example:8420", http.StatusForbidden},
		{"attacker.example", http.StatusForbidden},
		{"127.0.0.1.attacker.example:8420", http.StatusForbidden},
		{"localhost.attacker.example:8420", http.StatusForbidden},
		{"attacker.example@127.0.0.1:8420", http.StatusForbidden},
		{"192.0.2.7:8420", http.StatusForbidden},
		{"example.com", http.StatusForbidden},
		{"", http.StatusForbidden},
	}
	srv, err := NewServer(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	h := srv.Handler()
	for _, tc := range hosts {
		t.Run("read "+tc.host, func(t *testing.T) {
			if rec := getState(h, tc.host, ""); rec.Code != tc.want {
				t.Fatalf("GET /api/state with Host %q = %d, want %d: %s", tc.host, rec.Code, tc.want, rec.Body.String())
			}
		})
		t.Run("approve "+tc.host, func(t *testing.T) {
			files := &fakeFiles{tree: approveTree(approveTasks)}
			srv, h := approveServer(t, files)
			rec := postApprove(srv, h, `{"change":"alpha","branch":"claude/alpha"}`, func(r *http.Request) {
				r.Host = tc.host
				r.Header.Set("Origin", "http://"+tc.host)
				r.Header.Set("Sec-Fetch-Site", "same-origin")
			})
			if rec.Code != tc.want {
				t.Fatalf("approve with Host %q = %d, want %d: %s", tc.host, rec.Code, tc.want, rec.Body.String())
			}
			if tc.want != http.StatusOK && (files.calls != 0 || files.writes != nil) {
				t.Errorf("refused approve made %d forge calls and wrote %+v, want none", files.calls, files.writes)
			}
		})
	}
}

// The Origin of a request with a loopback Host must be loopback too, so a loopback Host header alone is not enough.
// Reads carry no other Origin check, which is why this is tested on GET.
//
// spec: openspec/specs/harness/spec.md#Requirement: Board state and authenticated API access
func TestTokenlessBoardRefusesForeignOrigin(t *testing.T) {
	t.Setenv("ATEAM_API_TOKEN", "")
	srv, err := NewServer(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	h := srv.Handler()
	for origin, want := range map[string]int{
		"http://localhost:8420":      http.StatusOK,
		"http://127.0.0.1:5173":      http.StatusOK,
		"http://[::1]:8420":          http.StatusOK,
		"http://attacker.example":    http.StatusForbidden,
		"https://attacker.example:1": http.StatusForbidden,
		"http://localhost.example":   http.StatusForbidden,
		"null":                       http.StatusForbidden,
		"://":                        http.StatusForbidden,
	} {
		if rec := getState(h, "localhost:8420", origin); rec.Code != want {
			t.Errorf("GET /api/state with Host localhost:8420 and Origin %q = %d, want %d", origin, rec.Code, want)
		}
	}

	files := &fakeFiles{tree: approveTree(approveTasks)}
	psrv, ph := approveServer(t, files)
	rec := postApprove(psrv, ph, `{"change":"alpha","branch":"claude/alpha"}`, func(r *http.Request) {
		r.Host = "localhost:8420"
		r.Header.Set("Origin", "http://attacker.example:8420")
	})
	if rec.Code != http.StatusForbidden || files.calls != 0 {
		t.Errorf("approve with a foreign Origin = %d after %d forge calls, want 403 and none", rec.Code, files.calls)
	}
}

// With a token the page that re-points its name cannot send the token, so a remote client with the token may use any
// Host: the Host check belongs to the tokenless branch only.
//
// spec: openspec/specs/harness/spec.md#Requirement: Board state and authenticated API access
func TestTokenedBoardAcceptsAnyHost(t *testing.T) {
	t.Setenv("ATEAM_API_TOKEN", "tok-secret-99")
	files := &fakeFiles{tree: approveTree(approveTasks)}
	srv, h := approveServer(t, files)

	req := httptest.NewRequest(http.MethodGet, "/api/state", nil)
	req.RemoteAddr = "203.0.113.5:9000"
	req.Host = "board.example.com"
	req.Header.Set("X-ATeam-Token", "tok-secret-99")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/state from a remote host with the token = %d, want 200: %s", rec.Code, rec.Body.String())
	}

	rec = postApprove(srv, h, `{"change":"alpha","branch":"claude/alpha"}`, func(r *http.Request) {
		r.RemoteAddr = "203.0.113.5:9000"
		r.Host = "board.example.com"
		r.Header.Set("Origin", "https://board.example.com")
		r.Header.Set("Sec-Fetch-Site", "same-origin")
		r.Header.Set("X-ATeam-Token", "tok-secret-99")
	})
	if rec.Code != http.StatusOK || len(files.writes) != 2 {
		t.Fatalf("approve from a remote host with the token = %d with %d writes, want 200 and 2: %s", rec.Code, len(files.writes), rec.Body.String())
	}

	rec = postApprove(srv, h, `{"change":"alpha","branch":"claude/alpha"}`, func(r *http.Request) {
		r.RemoteAddr = "203.0.113.5:9000"
		r.Host = "board.example.com"
	})
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("approve from a remote host without the token = %d, want 401", rec.Code)
	}
}
