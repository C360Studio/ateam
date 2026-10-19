package review

import (
	"encoding/hex"
	"fmt"
	"strings"
)

// Run is an attested actual model invocation, not proof of runtime identity.
type Run struct {
	ID       string `json:"id"`
	Model    string `json:"model"`
	Provider string `json:"provider"`
	Effort   string `json:"effort"`
}

// Record binds a substantive review to a repository, commits and governing policy.
type Record struct {
	Version      int       `json:"version"`
	Repository   string    `json:"repository"`
	Base         string    `json:"base"`
	Head         string    `json:"head"`
	PolicyDigest string    `json:"policy_digest"`
	Implementers []Run     `json:"implementers"`
	Reviewer     Run       `json:"reviewer"`
	ReadOnly     bool      `json:"read_only"`
	Verdict      string    `json:"verdict"`
	Findings     []Finding `json:"findings"`
	Checks       []Check   `json:"checks"`
	Limitations  []string  `json:"limitations"`
}

// Finding records a review concern and its severity.
type Finding struct {
	Severity string `json:"severity"`
	Message  string `json:"message"`
}

// Check distinguishes reviewer-executed checks from implementer reports.
type Check struct {
	Name   string `json:"name"`
	Commit string `json:"commit"`
	Status string `json:"status"`
	Source string `json:"source"`
}

// Expected carries trusted repository and revision identities supplied by the caller.
type Expected struct {
	Repository   string
	Base         string
	Head         string
	PolicyDigest string
	RecordAuthor string
}

// Exception waives only superiority; the authenticated author is supplied separately.
type Exception struct {
	Version       int    `json:"version"`
	Repository    string `json:"repository"`
	Head          string `json:"head"`
	ReviewerModel string `json:"reviewer_model"`
	PolicyDigest  string `json:"policy_digest"`
	Reason        string `json:"reason"`
}

// ParseRecord strictly decodes a review attestation. Validate establishes approval.
func ParseRecord(data []byte) (Record, error) {
	var record Record
	err := decodeJSON(data, &record)
	return record, err
}

// ParseException strictly decodes an exception without trusting an embedded owner identity.
func ParseException(data []byte) (Exception, error) {
	var exception Exception
	err := decodeJSON(data, &exception)
	return exception, err
}

// Eligible checks actual model identities, independence and all configured comparisons.
func Eligible(p Policy, implementers []Run, reviewer Run) error {
	if err := ValidatePolicy(p); err != nil {
		return err
	}
	return eligible(p, implementers, reviewer, false)
}

func eligible(p Policy, implementers []Run, reviewer Run, waiveSuperiority bool) error {
	if err := p.validateRun(reviewer); err != nil {
		return fmt.Errorf("reviewer: %w", err)
	}
	if len(implementers) == 0 {
		return fmt.Errorf("all actual implementers must be attested")
	}
	seen := map[string]bool{reviewer.ID: true}
	for _, dev := range implementers {
		if err := p.validateRun(dev); err != nil {
			return fmt.Errorf("implementer: %w", err)
		}
		if seen[dev.ID] {
			return fmt.Errorf("reviewer must be independent and run IDs unique")
		}
		seen[dev.ID] = true
		if p.CrossProvider && dev.Provider == reviewer.Provider {
			return fmt.Errorf("cross-provider review required for every implementer")
		}
		if !waiveSuperiority && !p.stronger(reviewer.Model, dev.Model) {
			return fmt.Errorf("reviewer model %q is not approved stronger than %q; an explicit owner exception is required", reviewer.Model, dev.Model)
		}
	}
	return nil
}

func (p Policy) validateRun(run Run) error {
	model, known := p.Model(run.Model)
	if !identity(run.ID) || !identity(run.Effort) || !known || run.Provider != model.Provider {
		return fmt.Errorf("unknown model, mismatched provider or unresolved run identity/effort")
	}
	return nil
}

// Validate accepts only current, independent approving evidence under trusted policy.
// Implementer completeness, read-only behavior and runtime model identity remain attestations.
func Validate(p Policy, r Record, expected Expected, exception *Exception, exceptionAuthor string) error {
	if err := ValidatePolicy(p); err != nil {
		return err
	}
	authorized := false
	for _, author := range p.ReviewAuthors {
		if author == expected.RecordAuthor {
			authorized = true
		}
	}
	if !authorized {
		return fmt.Errorf("record author is not an authenticated governing-policy reviewer")
	}
	if err := validateBinding(r, expected); err != nil {
		return err
	}
	if !r.ReadOnly {
		return fmt.Errorf("reviewer must attest read-only review")
	}
	if exception != nil {
		if err := validateException(p, *exception, exceptionAuthor, r); err != nil {
			return err
		}
	}
	if err := eligible(p, r.Implementers, r.Reviewer, exception != nil); err != nil {
		return err
	}
	if r.Verdict != "approve" {
		return fmt.Errorf("review verdict is not approve")
	}
	for _, finding := range r.Findings {
		if strings.TrimSpace(finding.Message) == "" {
			return fmt.Errorf("finding message is required")
		}
		switch finding.Severity {
		case "blocking", "high":
			return fmt.Errorf("unresolved %s finding blocks approval", finding.Severity)
		case "medium", "low", "info":
		default:
			return fmt.Errorf("unknown finding severity %q", finding.Severity)
		}
	}
	for _, limitation := range r.Limitations {
		if strings.TrimSpace(limitation) == "" {
			return fmt.Errorf("empty limitation")
		}
	}
	return validateChecks(p, r)
}

func validateBinding(r Record, expected Expected) error {
	if !identity(expected.Repository) || !commitID(expected.Base) || !commitID(expected.Head) || !digestID(expected.PolicyDigest) {
		return fmt.Errorf("trusted repository, full base/head commits and policy digest are required")
	}
	if r.Version != 1 {
		return fmt.Errorf("unsupported review record version %d", r.Version)
	}
	if r.Repository != expected.Repository || r.Base != expected.Base || r.Head != expected.Head || r.PolicyDigest != expected.PolicyDigest {
		return fmt.Errorf("review record has stale or mismatched repository/base/head/policy binding")
	}
	return nil
}

func validateException(p Policy, e Exception, author string, r Record) error {
	owner := false
	for _, candidate := range p.Owners {
		if author == candidate {
			owner = true
		}
	}
	if !owner {
		return fmt.Errorf("exception author is not an authenticated governing-policy owner")
	}
	if e.Version != 1 || e.Repository != r.Repository || e.Head != r.Head || e.ReviewerModel != r.Reviewer.Model || e.PolicyDigest != r.PolicyDigest || strings.TrimSpace(e.Reason) == "" {
		return fmt.Errorf("exception requires current repository/head/reviewer/policy scope and a reason")
	}
	return nil
}

func validateChecks(p Policy, r Record) error {
	seen := map[string]bool{}
	passed := map[string]bool{}
	for _, check := range r.Checks {
		if !identity(check.Name) || !commitID(check.Commit) {
			return fmt.Errorf("check name and full commit are required")
		}
		if check.Source != "reviewer" && check.Source != "implementer" {
			return fmt.Errorf("unknown check source %q", check.Source)
		}
		key := check.Name + "\x00" + check.Source
		if seen[key] {
			return fmt.Errorf("duplicate check %q from %s", check.Name, check.Source)
		}
		seen[key] = true
		switch check.Status {
		case "passed", "failed", "unavailable", "skipped":
		default:
			return fmt.Errorf("unknown check status %q", check.Status)
		}
		if check.Source == "reviewer" && check.Commit == r.Head && check.Status == "passed" {
			passed[check.Name] = true
		}
	}
	for _, required := range p.RequiredChecks {
		if !passed[required] {
			return fmt.Errorf("required check %q lacks reviewer-executed passing evidence at current head", required)
		}
	}
	return nil
}

func commitID(s string) bool { return (len(s) == 40 || len(s) == 64) && lowerHex(s) }
func digestID(s string) bool { return len(s) == 64 && lowerHex(s) }
func lowerHex(s string) bool {
	_, err := hex.DecodeString(s)
	return err == nil && s == strings.ToLower(s)
}
