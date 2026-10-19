package bootstrap

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testReviewPolicy = `{"version":1,"models":[{"id":"test-dev","provider":"codex"},{"id":"test-review","provider":"claude","stronger_than":["test-dev"]}],"routes":{"developer":{"model":"test-dev","effort":"medium"},"reviewer":{"model":"test-review","effort":"high"}},"owners":["project-owner"],"review_authors":["review-bot"],"required_checks":["unit"],"cross_provider":false}`

func TestConfigureReviewLifecycle(t *testing.T) {
	root := t.TempDir()
	if _, err := InitRepo(context.Background(), root, Options{}); err != nil {
		t.Fatal(err)
	}
	written, err := ConfigureReview(root, []byte(testReviewPolicy))
	if err != nil || len(written) != 4 {
		t.Fatalf("setup wrote %v, error %v; want policy, two routes, Claude pointer", written, err)
	}
	data, err := os.ReadFile(filepath.Join(root, ".agents/routes/reviewer.md"))
	if err != nil || !strings.Contains(string(data), `Model: "test-review"`) || !strings.Contains(string(data), `Reasoning effort: "high"`) {
		t.Fatalf("reviewer route lost model/effort: %s, %v", data, err)
	}
	if v := CheckParity(root); len(v) != 0 {
		t.Fatalf("configured setup violates parity: %v", v)
	}
	written, err = ConfigureReview(root, []byte(testReviewPolicy))
	if err != nil || len(written) != 0 {
		t.Fatalf("identical setup must not rewrite files: %v %v", written, err)
	}
	contract := filepath.Join(root, ".agents/contracts/reviewer.md")
	if err := os.WriteFile(contract, []byte(reviewerContract+"\nProject rule: verify API compatibility.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := ConfigureReview(root, []byte(testReviewPolicy)); err != nil {
		t.Fatalf("local contract additions must be preserved: %v", err)
	}
	if v := CheckParity(root); len(v) != 0 {
		t.Fatalf("local contract additions must pass parity: %v", v)
	}
}

func TestConfigureReviewConflictNoWrites(t *testing.T) {
	for _, conflict := range []string{".agents/review-policy.json", ".agents/contracts/reviewer.md", ".agents/routes/reviewer.md", ".claude/skills/ateam-reviewer/SKILL.md", ".gemini/commands/ateam-reviewer.toml", ".claude/skills/ateam-developer/SKILL.md", ".gemini/commands/ateam-developer.toml"} {
		t.Run(conflict, func(t *testing.T) {
			root := t.TempDir()
			path := filepath.Join(root, conflict)
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, []byte("custom"), 0o644); err != nil {
				t.Fatal(err)
			}
			written, err := ConfigureReview(root, []byte(testReviewPolicy))
			if err == nil || len(written) != 0 {
				t.Fatalf("conflict must fail before writes: %v %v", written, err)
			}
			files := 0
			err = filepath.WalkDir(root, func(_ string, entry os.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if !entry.IsDir() {
					files++
				}
				return nil
			})
			if err != nil || files != 1 {
				t.Fatalf("conflict wrote files: count %d error %v", files, err)
			}
			data, err := os.ReadFile(path)
			if err != nil || string(data) != "custom" {
				t.Fatalf("custom file changed: %s %v", data, err)
			}
		})
	}
}

func TestConfigureReviewRefusesInvalidAndSymlinks(t *testing.T) {
	root := t.TempDir()
	if _, err := ConfigureReview(root, []byte(`{"version":100}`)); err == nil {
		t.Fatal("invalid policy accepted")
	}
	if entries, err := os.ReadDir(root); err != nil || len(entries) != 0 {
		t.Fatalf("invalid policy wrote files: %v %v", entries, err)
	}
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(root, ".agents")); err != nil {
		t.Fatal(err)
	}
	if _, err := ConfigureReview(root, []byte(testReviewPolicy)); err == nil {
		t.Fatal("symlink accepted")
	}
	if entries, err := os.ReadDir(outside); err != nil || len(entries) != 0 {
		t.Fatalf("symlink target changed: %v %v", entries, err)
	}
}

func TestReviewSetupDrift(t *testing.T) {
	for _, path := range []string{".agents/review-policy.json", ".agents/contracts/developer.md", ".agents/contracts/reviewer.md", ".agents/routes/developer.md", ".agents/routes/reviewer.md", ".claude/skills/ateam-reviewer/SKILL.md"} {
		t.Run(path, func(t *testing.T) {
			root := t.TempDir()
			if _, err := ConfigureReview(root, []byte(testReviewPolicy)); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(root, path), []byte("drift"), 0o644); err != nil {
				t.Fatal(err)
			}
			requireViolation(t, CheckParity(root), path)
		})
	}
	root := t.TempDir()
	if _, err := InitRepo(context.Background(), root, Options{}); err != nil {
		t.Fatal(err)
	}
	if violations := checkReviewSetup(root); len(violations) != 0 {
		t.Fatalf("unconfigured repo must remain compatible: %v", violations)
	}
}

func TestConfigureReviewGeminiAndCeiling(t *testing.T) {
	root := t.TempDir()
	policy := strings.ReplaceAll(testReviewPolicy, `"provider":"claude"`, `"provider":"gemini"`)
	policy = strings.ReplaceAll(policy, `,"stronger_than":["test-dev"]`, "")
	if _, err := ConfigureReview(root, []byte(policy)); err != nil {
		t.Fatalf("ceiling policy must install for explicit exception review: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(root, ".gemini/commands/ateam-reviewer.toml"))
	if err != nil || !strings.Contains(string(data), ".agents/routes/reviewer.md fully") {
		t.Fatalf("missing Gemini pointer: %s %v", data, err)
	}
	if violations := checkReviewSetup(root); len(violations) != 0 {
		t.Fatalf("structurally valid ceiling config should pass doctor: %v", violations)
	}
}

func TestCanonicalReviewContracts(t *testing.T) {
	for role, expected := range map[string]string{"developer": developerContract, "reviewer": reviewerContract} {
		data, err := os.ReadFile(filepath.Join(repoRoot(t), ".agents/contracts", role+".md"))
		if err != nil || string(data) != expected {
			t.Fatalf("%s contract template drift: %v", role, err)
		}
	}
}

func TestConfigureReviewUnsupportedProvider(t *testing.T) {
	root := t.TempDir()
	policy := strings.ReplaceAll(testReviewPolicy, `"provider":"claude"`, `"provider":"unknown"`)
	if _, err := ConfigureReview(root, []byte(policy)); err == nil {
		t.Fatal("unsupported provider accepted")
	}
	if entries, err := os.ReadDir(root); err != nil || len(entries) != 0 {
		t.Fatalf("unsupported provider wrote files: %v %v", entries, err)
	}
}

func TestConfigureReviewProviderMatrix(t *testing.T) {
	checked := 0
	for _, developer := range []string{"codex", "claude", "gemini"} {
		for _, reviewer := range []string{"codex", "claude", "gemini"} {
			t.Run(developer+"/"+reviewer, func(t *testing.T) {
				root := t.TempDir()
				policy := strings.Replace(testReviewPolicy, `"provider":"codex"`, `"provider":"`+developer+`"`, 1)
				policy = strings.Replace(policy, `"id":"test-review","provider":"claude"`, `"id":"test-review","provider":"`+reviewer+`"`, 1)
				if _, err := ConfigureReview(root, []byte(policy)); err != nil {
					t.Fatal(err)
				}
				for role, provider := range map[string]string{"developer": developer, "reviewer": reviewer} {
					for _, candidate := range []struct{ name, path string }{
						{"claude", ".claude/skills/ateam-" + role + "/SKILL.md"},
						{"gemini", ".gemini/commands/ateam-" + role + ".toml"},
					} {
						_, err := os.Stat(filepath.Join(root, candidate.path))
						if provider == candidate.name && err != nil {
							t.Fatalf("missing applicable pointer %s: %v", candidate.path, err)
						}
						if provider != candidate.name && !os.IsNotExist(err) {
							t.Fatalf("unexpected pointer %s: %v", candidate.path, err)
						}
						if provider != candidate.name {
							// Model a stale pointer left by a provider switch.
							path := filepath.Join(root, candidate.path)
							if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
								t.Fatal(err)
							}
							if err := os.WriteFile(path, []byte("previous provider pointer"), 0600); err != nil {
								t.Fatal(err)
							}
							requireViolation(t, checkReviewSetup(root), candidate.path)
							if written, err := ConfigureReview(root, []byte(policy)); err == nil || len(written) != 0 {
								t.Fatalf("stale provider pointer must block setup: %v %v", written, err)
							}
							if err := os.Remove(path); err != nil {
								t.Fatal(err)
							}
							checked++
						}
					}
				}
				if written, err := ConfigureReview(root, []byte(policy)); err != nil || len(written) != 0 {
					t.Fatalf("retry changed setup: %v %v", written, err)
				}
				if v := checkReviewSetup(root); len(v) != 0 {
					t.Fatalf("generated setup drifted: %v", v)
				}
			})
		}
	}
	if checked != 24 {
		t.Fatalf("inactive-pointer assertions ran %d times, want 24", checked)
	}
	t.Logf("completed %d inactive-pointer install/reject/reconcile histories across nine provider pairs", checked)
}

// The listed paths come from the supported provider/role contract, independent
// of reviewFiles. Existing inactive pointers must never be silently accepted.
func TestReviewInactivePointers(t *testing.T) {
	for _, rel := range []string{".claude/skills/ateam-developer/SKILL.md", ".gemini/commands/ateam-developer.toml", ".gemini/commands/ateam-reviewer.toml"} {
		for _, kind := range []string{"file", "dangling-symlink"} {
			t.Run(rel+"/"+kind, func(t *testing.T) {
				root := t.TempDir()
				if _, err := ConfigureReview(root, []byte(testReviewPolicy)); err != nil {
					t.Fatal(err)
				}
				path := filepath.Join(root, rel)
				if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
					t.Fatal(err)
				}
				if kind == "file" {
					if err := os.WriteFile(path, []byte("custom inactive pointer"), 0600); err != nil {
						t.Fatal(err)
					}
				} else {
					if err := os.Symlink("missing-target", path); err != nil {
						t.Fatal(err)
					}
				}
				requireViolation(t, checkReviewSetup(root), rel)
				if written, err := ConfigureReview(root, []byte(testReviewPolicy)); err == nil || len(written) != 0 {
					t.Fatalf("inactive pointer must block setup without writes: %v %v", written, err)
				}
				if _, err := os.Lstat(path); err != nil {
					t.Fatalf("inactive pointer was removed: %v", err)
				}
			})
		}
	}
}

func TestReviewUnconfiguredArtifacts(t *testing.T) {
	for _, role := range []string{"developer", "reviewer"} {
		for _, operation := range []string{"delete", "drift"} {
			t.Run(role+"/"+operation, func(t *testing.T) {
				root := t.TempDir()
				if _, err := InitRepo(context.Background(), root, Options{}); err != nil {
					t.Fatal(err)
				}
				rel := ".agents/contracts/" + role + ".md"
				path := filepath.Join(root, rel)
				if operation == "delete" {
					if err := os.Remove(path); err != nil {
						t.Fatal(err)
					}
				} else {
					if err := os.WriteFile(path, []byte("drift"), 0600); err != nil {
						t.Fatal(err)
					}
				}
				requireViolation(t, CheckParity(root), rel)
			})
		}
		for _, rel := range []string{".agents/routes/" + role + ".md", ".claude/skills/ateam-" + role + "/SKILL.md", ".gemini/commands/ateam-" + role + ".toml"} {
			t.Run(rel, func(t *testing.T) {
				root := t.TempDir()
				if _, err := InitRepo(context.Background(), root, Options{}); err != nil {
					t.Fatal(err)
				}
				path := filepath.Join(root, rel)
				if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink("missing-policy", path); err != nil {
					t.Fatal(err)
				}
				requireViolation(t, CheckParity(root), rel)
			})
		}
	}
}

func TestReviewAbsentInactivePointerUnderLinkedDirectory(t *testing.T) {
	root := t.TempDir()
	if err := os.Symlink(t.TempDir(), filepath.Join(root, ".gemini")); err != nil {
		t.Fatal(err)
	}
	// Neither role uses Gemini. A shared directory without reserved role
	// pointers is unrelated configuration and must not become a conflict.
	if _, err := ConfigureReview(root, []byte(testReviewPolicy)); err != nil {
		t.Fatalf("unrelated linked provider directory blocked setup: %v", err)
	}
	if violations := checkReviewSetup(root); len(violations) != 0 {
		t.Fatalf("unrelated linked provider directory caused drift: %v", violations)
	}
}
