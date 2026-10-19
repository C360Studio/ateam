package review

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

const forgePRFixture = `{"state":"open","merged":false,"number":7,"base":{"sha":"1111111111111111111111111111111111111111"},"head":{"sha":"2222222222222222222222222222222222222222"},"draft":false,"user":{"login":"developer"}}`

func TestForgeProviders(t *testing.T) {
	for _, provider := range []string{"github", "gitea"} {
		t.Run(provider, func(t *testing.T) {
			prefix, auth := "/repos/team/project", "Bearer secret"
			if provider == "gitea" {
				prefix, auth = "/api/v1/repos/team/project", "token secret"
			}
			var base string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet || r.Header.Get("Authorization") != auth {
					t.Errorf("unexpected request: %s, authorization %q", r.Method, r.Header.Get("Authorization"))
				}
				switch r.URL.Path {
				case prefix + "/pulls/7":
					fmt.Fprint(w, strings.Replace(forgePRFixture, `"base":{`, `"base":{"ref":"main","repo":{"full_name":"team/project"},`, 1))
				case prefix:
					fmt.Fprint(w, `{"full_name":"team/project","default_branch":"main"}`)
				case prefix + "/branches/main":
					if provider == "github" {
						fmt.Fprint(w, `{"name":"main","commit":{"sha":"1111111111111111111111111111111111111111"}}`)
					} else {
						fmt.Fprint(w, `{"name":"main","commit":{"id":"1111111111111111111111111111111111111111"}}`)
					}
				case prefix + "/issues/comments/19":
					field, parent := "issue_url", base+prefix+"/issues/7"
					if provider == "gitea" {
						field, parent = "pull_request_url", base+"/team/project/pulls/7"
					}
					fmt.Fprintf(w, `{"id":19,%q:%q,"body":"exact body\n","user":{"login":"reviewer"},"created_at":"2026-10-19T10:00:00Z","updated_at":"2026-10-19T10:00:00Z"}`, field, parent)
				default:
					t.Errorf("unexpected endpoint %s", r.URL.Path)
					http.NotFound(w, r)
				}
			}))
			defer server.Close()
			base = server.URL
			cfg := ForgeConfig{Provider: provider, BaseURL: base, Owner: "team", Repo: "project", Token: "secret"}
			pr, err := ReadPullRequest(context.Background(), server.Client(), cfg, 7)
			if err != nil {
				t.Fatal(err)
			}
			want := PullRequest{Base: "1111111111111111111111111111111111111111", Head: "2222222222222222222222222222222222222222", Author: "developer"}
			if pr != want {
				t.Fatalf("PR = %+v, want %+v", pr, want)
			}
			comment, err := ReadComment(context.Background(), server.Client(), cfg, 7, 19)
			if err != nil {
				t.Fatal(err)
			}
			if comment != (Comment{Author: "reviewer", Body: "exact body\n", CreatedAt: "2026-10-19T10:00:00Z", UpdatedAt: "2026-10-19T10:00:00Z"}) {
				t.Fatalf("comment = %+v", comment)
			}
		})
	}
}

func TestForgeRejectsPartialPullRequests(t *testing.T) {
	for _, provider := range []string{"github", "gitea"} {
		t.Run(provider, func(t *testing.T) {
			cases := map[string]string{
				"closed":         strings.Replace(forgePRFixture, `"state":"open"`, `"state":"closed"`, 1),
				"merged":         strings.Replace(forgePRFixture, `"merged":false`, `"merged":true`, 1),
				"missing merged": strings.Replace(forgePRFixture, `"merged":false,`, "", 1),
				"empty":          "{}", "null": "null", "invalid JSON": "{", "trailing JSON": forgePRFixture + "{}",
				"wrong number":   strings.Replace(forgePRFixture, `"number":7`, `"number":8`, 1),
				"missing draft":  strings.Replace(forgePRFixture, `"draft":false,`, "", 1),
				"null draft":     strings.Replace(forgePRFixture, `"draft":false`, `"draft":null`, 1),
				"short SHA":      strings.Replace(forgePRFixture, "1111111111111111111111111111111111111111", "abcdef0", 1),
				"invalid SHA":    strings.Replace(forgePRFixture, "2222222222222222222222222222222222222222", "zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz", 1),
				"missing author": strings.Replace(forgePRFixture, `"developer"`, `""`, 1),
			}
			for name, body := range cases {
				t.Run(name, func(t *testing.T) {
					server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { fmt.Fprint(w, body) }))
					defer server.Close()
					got, err := ReadPullRequest(context.Background(), server.Client(), ForgeConfig{Provider: provider, BaseURL: server.URL, Owner: "team", Repo: "project"}, 7)
					if err == nil || got != (PullRequest{}) {
						t.Fatalf("partial response accepted: %+v, %v", got, err)
					}
				})
			}
		})
	}
}

func TestForgeRejectsUnboundComments(t *testing.T) {
	for _, provider := range []string{"github", "gitea"} {
		t.Run(provider, func(t *testing.T) {
			for _, name := range []string{"wrong PR", "wrong host", "wrong repo", "missing parent", "wrong ID", "missing author", "missing body"} {
				t.Run(name, func(t *testing.T) {
					var body string
					server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { fmt.Fprint(w, body) }))
					defer server.Close()
					field, parentPrefix, parentKind := "issue_url", server.URL+"/repos", "issues"
					if provider == "gitea" {
						field, parentPrefix, parentKind = "pull_request_url", server.URL, "pulls"
					}
					parent := parentPrefix + "/team/project/" + parentKind + "/7"
					id, author, text := 19, "reviewer", "evidence"
					switch name {
					case "wrong PR":
						parent = parentPrefix + "/team/project/" + parentKind + "/8"
					case "wrong host":
						parent = "https://untrusted.test/team/project/" + parentKind + "/7"
					case "wrong repo":
						parent = parentPrefix + "/other/project/" + parentKind + "/7"
					case "missing parent":
						parent = ""
					case "wrong ID":
						id = 20
					case "missing author":
						author = ""
					case "missing body":
						text = ""
					}
					body = fmt.Sprintf(`{"id":%d,%q:%q,"user":{"login":%q},"body":%q,"created_at":"2026-10-19T10:00:00Z","updated_at":"2026-10-19T10:00:00Z"}`, id, field, parent, author, text)
					got, err := ReadComment(context.Background(), server.Client(), ForgeConfig{Provider: provider, BaseURL: server.URL, Owner: "team", Repo: "project"}, 7, 19)
					if err == nil || got != (Comment{}) {
						t.Fatalf("unbound comment accepted: %+v, %v", got, err)
					}
				})
			}
		})
	}
}

func TestForgeHTTPFailures(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status int
		body   string
	}{
		{"unauthorized", 401, `{"token":"must not be echoed"}`},
		{"server failure", 500, "server failure"},
		{"empty success", 204, ""},
		{"oversized", 200, strings.Repeat(" ", maxForgeResponse) + forgePRFixture},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(tc.status); fmt.Fprint(w, tc.body) }))
			defer server.Close()
			_, err := ReadPullRequest(context.Background(), server.Client(), ForgeConfig{Provider: "github", BaseURL: server.URL, Owner: "team", Repo: "project"}, 7)
			if err == nil {
				t.Fatal("accepted failed transport")
			}
			if strings.Contains(err.Error(), "must not be echoed") {
				t.Fatal("response body leaked in error")
			}
		})
	}
}

func TestForgeRedirectDoesNotForwardCredentials(t *testing.T) {
	var requests atomic.Int32
	destination := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { requests.Add(1); fmt.Fprint(w, forgePRFixture) }))
	defer destination.Close()
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, destination.URL, http.StatusFound) }))
	defer source.Close()
	client := source.Client()
	callbackCalled := false
	client.CheckRedirect = func(*http.Request, []*http.Request) error { callbackCalled = true; return nil }
	_, err := ReadPullRequest(context.Background(), client, ForgeConfig{Provider: "github", BaseURL: source.URL, Owner: "team", Repo: "project", Token: "secret"}, 7)
	if err == nil || requests.Load() != 0 || callbackCalled {
		t.Fatalf("redirect followed: err=%v requests=%d callback=%v", err, requests.Load(), callbackCalled)
	}
	if client.CheckRedirect == nil {
		t.Fatal("caller client modified")
	}
}

func TestForgeCancellation(t *testing.T) {
	started := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) { close(started); <-r.Context().Done() }))
	defer server.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	result := make(chan error, 1)
	go func() {
		_, err := ReadPullRequest(ctx, server.Client(), ForgeConfig{Provider: "github", BaseURL: server.URL, Owner: "team", Repo: "project"}, 7)
		result <- err
	}()
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		cancel()
		t.Fatal("request never reached server")
	}
	cancel()
	select {
	case err := <-result:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("cancellation not propagated: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("cancellation did not finish request")
	}
}

func TestForgeRejectsInvalidConfiguration(t *testing.T) {
	for _, tc := range []struct{ name, provider, base, owner, repo string }{
		{"provider", "local", "https://example.test", "team", "project"},
		{"cleartext", "github", "http://example.test", "team", "project"},
		{"userinfo", "github", "https://secret@example.test", "team", "project"},
		{"query", "github", "https://example.test?token=secret", "team", "project"},
		{"fragment", "github", "https://example.test#fragment", "team", "project"},
		{"owner traversal", "github", "https://example.test", "..", "project"},
		{"repo slash", "github", "https://example.test", "team", "other/project"},
		{"relative base", "github", "/example", "team", "project"},
		{"relative path", "github", "https://example.test/../other", "team", "project"},
		{"escaped path", "github", "https://example.test/%2e%2e", "team", "project"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := ForgeConfig{Provider: tc.provider, BaseURL: tc.base, Owner: tc.owner, Repo: tc.repo}
			if _, err := ReadPullRequest(context.Background(), nil, cfg, 7); err == nil {
				t.Fatal("invalid configuration accepted")
			}
		})
	}
	cfg := ForgeConfig{Provider: "github", BaseURL: "https://example.test", Owner: "team", Repo: "project"}
	if _, err := ReadPullRequest(context.Background(), nil, cfg, 0); err == nil {
		t.Fatal("zero PR accepted")
	}
	if _, err := ReadComment(context.Background(), nil, cfg, 7, 0); err == nil {
		t.Fatal("zero comment ID accepted")
	}
}

// Gitea's issue comment converter uses pull_request_url for PR comments,
// with an HTML pull URL, and leaves issue_url empty.
func TestForgeGiteaPullComment(t *testing.T) {
	var base string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/gitea/api/v1/repos/team/project/issues/comments/19" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		fmt.Fprintf(w, `{"id":19,"issue_url":"","pull_request_url":%q,"body":"evidence","user":{"login":"reviewer"},"created_at":"2026-10-19T10:00:00Z","updated_at":"2026-10-19T10:00:00Z"}`, base+"/team/project/pulls/7")
	}))
	defer server.Close()
	base = server.URL + "/gitea"
	comment, err := ReadComment(context.Background(), server.Client(), ForgeConfig{Provider: "gitea", BaseURL: base, Owner: "team", Repo: "project"}, 7, 19)
	if err != nil || comment.Author != "reviewer" || comment.Body != "evidence" {
		t.Fatalf("genuine Gitea pull comment rejected: %+v, %v", comment, err)
	}
}

func TestForgeTargetAuthority(t *testing.T) {
	for _, provider := range []string{"github", "gitea"} {
		t.Run(provider, func(t *testing.T) {
			for _, tc := range []struct {
				name, target, repo, defaultBranch, currentSHA string
				ok                                            bool
			}{
				{"default", "main", "team/project", "main", "1111111111111111111111111111111111111111", true},
				{"wrong target", "attacker-policy", "team/project", "main", "1111111111111111111111111111111111111111", false},
				{"wrong repo", "main", "other/project", "main", "1111111111111111111111111111111111111111", false},
				{"missing target", "", "team/project", "main", "1111111111111111111111111111111111111111", false},
				{"wrong repository metadata", "main", "team/project", "main", "1111111111111111111111111111111111111111", false},
				{"wrong branch response", "main", "team/project", "main", "1111111111111111111111111111111111111111", false},
				{"missing current SHA", "main", "team/project", "main", "", false},
				{"missing default", "main", "team/project", "", "1111111111111111111111111111111111111111", false},
				{"stale SHA", "main", "team/project", "main", "3333333333333333333333333333333333333333", false},
				{"slash branch", "release/main", "team/project", "release/main", "1111111111111111111111111111111111111111", true},
			} {
				t.Run(tc.name, func(t *testing.T) {
					prefix := "/repos/team/project"
					if provider == "gitea" {
						prefix = "/api/v1" + prefix
					}
					pr := strings.Replace(forgePRFixture, `"base":{`, fmt.Sprintf(`"base":{"ref":%q,"repo":{"full_name":%q},`, tc.target, tc.repo), 1)
					server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						switch r.URL.Path {
						case prefix + "/pulls/7":
							fmt.Fprint(w, pr)
						case prefix:
							fullName := "team/project"
							if tc.name == "wrong repository metadata" {
								fullName = "other/project"
							}
							fmt.Fprintf(w, `{"full_name":%q,"default_branch":%q}`, fullName, tc.defaultBranch)
						case prefix + "/branches/" + tc.defaultBranch:
							key := "sha"
							if provider == "gitea" {
								key = "id"
							}
							if r.URL.EscapedPath() != prefix+"/branches/"+url.PathEscape(tc.defaultBranch) {
								t.Errorf("branch path not escaped: %s", r.URL.EscapedPath())
							}
							name := tc.defaultBranch
							if tc.name == "wrong branch response" {
								name = "other"
							}
							fmt.Fprintf(w, `{"name":%q,"commit":{%q:%q}}`, name, key, tc.currentSHA)
						default:
							t.Errorf("unexpected request %s", r.URL.Path)
							http.NotFound(w, r)
						}
					}))
					defer server.Close()
					_, err := ReadPullRequest(context.Background(), server.Client(), ForgeConfig{Provider: provider, BaseURL: server.URL, Owner: "team", Repo: "project"}, 7)
					if (err == nil) != tc.ok {
						t.Fatalf("target authority: success=%v, want %v (%v)", err == nil, tc.ok, err)
					}
				})
			}
		})
	}
}

func TestForgeRejectEditedComments(t *testing.T) {
	for _, provider := range []string{"github", "gitea"} {
		t.Run(provider, func(t *testing.T) {
			for _, tc := range []struct{ name, created, updated string }{
				{"edited", "2026-10-19T10:00:00Z", "2026-10-19T10:00:01Z"},
				{"missing created", "", "2026-10-19T10:00:00Z"},
				{"missing updated", "2026-10-19T10:00:00Z", ""},
				{"invalid timestamps", "yesterday", "yesterday"},
			} {
				t.Run(tc.name, func(t *testing.T) {
					var base string
					server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
						field, parent := "issue_url", base+"/repos/team/project/issues/7"
						if provider == "gitea" {
							field, parent = "pull_request_url", base+"/team/project/pulls/7"
						}
						fmt.Fprintf(w, `{"id":19,%q:%q,"body":"evidence","user":{"login":"reviewer"},"created_at":%q,"updated_at":%q}`, field, parent, tc.created, tc.updated)
					}))
					defer server.Close()
					base = server.URL
					_, err := ReadComment(context.Background(), server.Client(), ForgeConfig{Provider: provider, BaseURL: base, Owner: "team", Repo: "project"}, 7, 19)
					if err == nil {
						t.Fatal("edited or unprovable comment accepted")
					}
				})
			}
		})
	}
}
