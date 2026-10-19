package review

import (
	"fmt"
	"strings"
	"testing"
)

func testRecord() (Record, Expected) {
	e := Expected{RecordAuthor: "reviewer-login", Repository: "https://github.com/org/repo", Base: strings.Repeat("a", 40), Head: strings.Repeat("b", 40), PolicyDigest: strings.Repeat("c", 64)}
	r := Record{Version: 1, Repository: e.Repository, Base: e.Base, Head: e.Head, PolicyDigest: e.PolicyDigest, Implementers: []Run{{ID: "dev", Model: "small", Provider: "codex", Effort: "high"}}, Reviewer: Run{ID: "review", Model: "medium", Provider: "codex", Effort: "low"}, ReadOnly: true, Verdict: "approve", Findings: []Finding{}, Checks: []Check{{Name: "test", Commit: e.Head, Status: "passed", Source: "reviewer"}}, Limitations: []string{}}
	return r, e
}

// Model eligibility cannot substitute for a current substantive independent review.
func TestRecordRefusals(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*Record)
	}{
		{"version", func(r *Record) { r.Version = 2 }},
		{"repository", func(r *Record) { r.Repository += "other" }},
		{"base", func(r *Record) { r.Base = strings.Repeat("d", 40) }},
		{"head", func(r *Record) { r.Head = strings.Repeat("d", 40) }},
		{"policy", func(r *Record) { r.PolicyDigest = strings.Repeat("d", 64) }},
		{"writable", func(r *Record) { r.ReadOnly = false }},
		{"self-review", func(r *Record) { r.Reviewer.ID = "dev" }},
		{"missing-implementers", func(r *Record) { r.Implementers = nil }},
		{"duplicate-implementers", func(r *Record) { r.Implementers = append(r.Implementers, r.Implementers[0]) }},
		{"escalated-implementer", func(r *Record) {
			r.Implementers = append(r.Implementers, Run{ID: "dev2", Model: "large", Provider: "claude", Effort: "low"})
		}},
		{"equal-reviewer", func(r *Record) { r.Reviewer.Model = "small"; r.Reviewer.Effort = "ultra" }},
		{"unknown-developer", func(r *Record) { r.Implementers[0].Model = "unknown" }},
		{"changes-requested", func(r *Record) { r.Verdict = "request_changes" }},
		{"high", func(r *Record) { r.Findings = []Finding{{Severity: "high", Message: "bug"}} }},
		{"blocking", func(r *Record) { r.Findings = []Finding{{Severity: "blocking", Message: "bug"}} }},
		{"unknown-severity", func(r *Record) { r.Findings = []Finding{{Severity: "critical", Message: "bug"}} }},
		{"empty-finding", func(r *Record) { r.Findings = []Finding{{Severity: "low"}} }},
		{"empty-limitation", func(r *Record) { r.Limitations = []string{" "} }},
		{"missing-check", func(r *Record) { r.Checks = nil }},
		{"failed-check", func(r *Record) { r.Checks[0].Status = "failed" }},
		{"unavailable-check", func(r *Record) { r.Checks[0].Status = "unavailable" }},
		{"skipped-check", func(r *Record) { r.Checks[0].Status = "skipped" }},
		{"reported-check", func(r *Record) { r.Checks[0].Source = "implementer" }},
		{"stale-check", func(r *Record) { r.Checks[0].Commit = r.Base }},
		{"short-check", func(r *Record) { r.Checks[0].Commit = "abc" }},
		{"duplicate-check", func(r *Record) { r.Checks = append(r.Checks, r.Checks[0]) }},
		{"unknown-source", func(r *Record) { r.Checks[0].Source = "ci" }},
		{"unknown-status", func(r *Record) { r.Checks[0].Status = "success" }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r, e := testRecord()
			tc.mutate(&r)
			if err := Validate(testPolicy(), r, e, nil, ""); err == nil {
				t.Fatal("invalid review approved")
			}
		})
	}
	r, e := testRecord()
	r.Findings = []Finding{{Severity: "medium", Message: "follow-up"}}
	r.Limitations = []string{"runtime identity is attested"}
	r.Checks = append(r.Checks, Check{Name: "test", Commit: r.Head, Status: "passed", Source: "implementer"})
	if err := Validate(testPolicy(), r, e, nil, ""); err != nil {
		t.Fatal(err)
	}
	e.Head = "abc"
	if err := Validate(testPolicy(), r, e, nil, ""); err == nil {
		t.Fatal("short trusted commit accepted")
	}
}

func testException(r Record) Exception {
	return Exception{Version: 1, Repository: r.Repository, Head: r.Head, ReviewerModel: r.Reviewer.Model, PolicyDigest: r.PolicyDigest, Reason: "explicitly accept equal model for this commit"}
}

func TestOwnerException(t *testing.T) {
	r, e := testRecord()
	r.Reviewer.Model = "small"
	x := testException(r)
	if err := Validate(testPolicy(), r, e, &x, "owner"); err != nil {
		t.Fatalf("scoped owner exception rejected: %v", err)
	}
	for _, author := range []string{"", "attacker", "Owner"} {
		if err := Validate(testPolicy(), r, e, &x, author); err == nil {
			t.Fatalf("unauthorized author %q accepted", author)
		}
	}
	cases := []struct {
		name   string
		mutate func(*Exception)
	}{
		{"version", func(x *Exception) { x.Version = 2 }},
		{"repository", func(x *Exception) { x.Repository = "other" }},
		{"head", func(x *Exception) { x.Head = strings.Repeat("d", 40) }},
		{"model", func(x *Exception) { x.ReviewerModel = "large" }},
		{"policy", func(x *Exception) { x.PolicyDigest = strings.Repeat("d", 64) }},
		{"reason", func(x *Exception) { x.Reason = " " }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			bad := x
			tc.mutate(&bad)
			if err := Validate(testPolicy(), r, e, &bad, "owner"); err == nil {
				t.Fatal("unscoped exception approved")
			}
		})
	}
	t.Run("cannot-waive-independence", func(t *testing.T) {
		bad := r
		bad.Reviewer.ID = "dev"
		if err := Validate(testPolicy(), bad, e, &x, "owner"); err == nil {
			t.Fatal("exception waived independence")
		}
	})
	t.Run("cannot-waive-cross-provider", func(t *testing.T) {
		p := testPolicy()
		p.CrossProvider = true
		if err := Validate(p, r, e, &x, "owner"); err == nil {
			t.Fatal("exception waived cross-provider")
		}
	})
	t.Run("cannot-waive-evidence", func(t *testing.T) {
		bad := r
		bad.Checks = nil
		if err := Validate(testPolicy(), bad, e, &x, "owner"); err == nil {
			t.Fatal("exception waived required checks")
		}
	})
	t.Run("cannot-waive-verdict", func(t *testing.T) {
		bad := r
		bad.Verdict = "request_changes"
		if err := Validate(testPolicy(), bad, e, &x, "owner"); err == nil {
			t.Fatal("exception waived verdict")
		}
	})
}

func TestRecordParsing(t *testing.T) {
	r, _ := testRecord()
	if _, err := ParseRecord(mustJSON(t, r)); err != nil {
		t.Fatal(err)
	}
	x := testException(r)
	if _, err := ParseException(mustJSON(t, x)); err != nil {
		t.Fatal(err)
	}
	for _, data := range []string{`{"owner":"owner"}`, `{"reason":"first","reason":"second"}`, `{"head":null}`, `{"reviewer_model":{}}`} {
		if _, err := ParseException([]byte(data)); err == nil {
			t.Fatalf("invalid exception JSON accepted: %s", data)
		}
	}
	if _, err := ParseRecord([]byte(`{"reviewer":{"ID":"spoof"}}`)); err == nil {
		t.Fatal("case-insensitive nested field accepted")
	}
}

func TestRecordAuthor(t *testing.T) {
	r, e := testRecord()
	for _, author := range []string{"", "attacker", "owner", "Reviewer-Login"} {
		e.RecordAuthor = author
		if err := Validate(testPolicy(), r, e, nil, ""); err == nil {
			t.Fatalf("unauthorized record author %q approved", author)
		}
	}
	e.RecordAuthor = "reviewer-login"
	if err := Validate(testPolicy(), r, e, nil, ""); err != nil {
		t.Fatalf("authorized reviewer refused: %v", err)
	}
}

// Enumerated histories use the rule that approval survives only if no identity changed.
func TestReviewHistory(t *testing.T) {
	checks := 0
	for sequence := 0; sequence < 256; sequence++ {
		r, e := testRecord()
		r.Reviewer.Model = "small"
		x := testException(r)
		fresh := true
		for step := 0; step < 4; step++ {
			switch (sequence >> (step * 2)) & 3 {
			case 0: // no change
			case 1:
				e.Head = fmt.Sprintf("%040x", step+1)
				fresh = false
			case 2:
				e.PolicyDigest = fmt.Sprintf("%064x", step+1)
				fresh = false
			case 3: // explicit fresh review and owner approval of the new identities
				r.Head = e.Head
				r.PolicyDigest = e.PolicyDigest
				r.Checks[0].Commit = e.Head
				x = testException(r)
				fresh = true
			}
			err := Validate(testPolicy(), r, e, &x, "owner")
			if (err == nil) != fresh {
				t.Fatalf("history %d step %d: fresh=%v got %v", sequence, step, fresh, err)
			}
			checks++
		}
	}
	if checks != 1024 {
		t.Fatalf("assertion activation: got %d checks", checks)
	}
	t.Logf("completed %d history checks over 256 four-step histories", checks)
}
