// Package review validates explicitly approved model comparisons and review attestations.
package review

import (
	"crypto/sha256"
	"fmt"
	"strings"
)

// Policy declares model relationships; model names and effort never imply rank.
type Policy struct {
	Version        int      `json:"version"`
	Models         []Model  `json:"models,omitempty"`
	Routes         Routes   `json:"routes"`
	Owners         []string `json:"owners,omitempty"`
	ReviewAuthors  []string `json:"review_authors,omitempty"`
	RequiredChecks []string `json:"required_checks,omitempty"`
	CrossProvider  bool     `json:"cross_provider"`
	// ModelTiers and EffortBands rank runs for merge-check (ADR 0002). A policy may
	// declare only these; Models, Routes and the lists above serve the dormant
	// route/check path and are then absent.
	ModelTiers  ModelTiers  `json:"model_tiers"`
	EffortBands EffortBands `json:"effort_bands"`
	// ReviewRecord names where the review record lives: RecordPRComment (the
	// default when absent) or RecordAIP (owner ruling on #130).
	ReviewRecord string `json:"review_record,omitempty"`
}

// Review record modes. Under RecordAIP the record is an archived REVIEW-002
// review-implementation.md with a matching approval.md, not a PR comment.
const (
	RecordPRComment = "pr-comment"
	RecordAIP       = "aip"
)

// ModelTiers lists "provider:model" names from strongest tier to weakest.
type ModelTiers struct {
	Frontier  []string `json:"frontier,omitempty"`
	Workhorse []string `json:"workhorse,omitempty"`
	Light     []string `json:"light,omitempty"`
}

// EffortBands maps each provider's native effort values onto three bands.
type EffortBands struct {
	Claude Bands `json:"claude"`
	Codex  Bands `json:"codex"`
	Gemini Bands `json:"gemini"`
}

// Bands lists native effort values from the lowest band to the highest.
type Bands struct {
	Low  []string `json:"low,omitempty"`
	Mid  []string `json:"mid,omitempty"`
	High []string `json:"high,omitempty"`
}

// HasBands reports whether the policy maps any native effort value.
func (p Policy) HasBands() bool {
	n := 0
	for _, b := range []Bands{p.EffortBands.Claude, p.EffortBands.Codex, p.EffortBands.Gemini} {
		n += len(b.Low) + len(b.Mid) + len(b.High)
	}
	return n > 0
}

// HasRoutes reports whether the policy configures the dormant route/check path.
func (p Policy) HasRoutes() bool { return len(p.Models) > 0 }

// HasTiers reports whether the policy ranks runs by tier and effort band.
func (p Policy) HasTiers() bool {
	return len(p.ModelTiers.Frontier)+len(p.ModelTiers.Workhorse)+len(p.ModelTiers.Light) > 0
}

// Model identifies a provider model and its explicitly weaker models.
type Model struct {
	ID           string   `json:"id"`
	Provider     string   `json:"provider"`
	StrongerThan []string `json:"stronger_than,omitempty"`
}

// Routes separates implementation and review model selections.
type Routes struct {
	Developer Route `json:"developer"`
	Reviewer  Route `json:"reviewer"`
}

// Route records effort separately from model strength.
type Route struct {
	Model  string `json:"model"`
	Effort string `json:"effort"`
}

// ParsePolicy strictly decodes and validates a governing policy.
func ParsePolicy(data []byte) (Policy, error) {
	var p Policy
	if err := decodeJSON(data, &p); err != nil {
		return p, err
	}
	return p, ValidatePolicy(p)
}

// PolicyDigest binds evidence to the exact governing policy bytes.
func PolicyDigest(data []byte) string { return fmt.Sprintf("%x", sha256.Sum256(data)) }

// Model looks up an exact model identity.
func (p Policy) Model(id string) (Model, bool) {
	for _, model := range p.Models {
		if model.ID == id {
			return model, true
		}
	}
	return Model{}, false
}

// ValidatePolicy rejects incomplete, ambiguous or contradictory configuration.
func ValidatePolicy(p Policy) error {
	if p.Version != 1 {
		return fmt.Errorf("unsupported policy version %d", p.Version)
	}
	switch p.ReviewRecord {
	case "", RecordPRComment, RecordAIP:
	default:
		return fmt.Errorf("review_record %q must be %q or %q", p.ReviewRecord, RecordPRComment, RecordAIP)
	}
	// Supplied tiers or bands are validated together, with or without models.
	if p.HasTiers() || p.HasBands() {
		if err := p.validateTiers(); err != nil {
			return err
		}
	}
	if !p.HasRoutes() {
		if !p.HasTiers() {
			return fmt.Errorf("policy must contain model tiers or 1..256 models")
		}
		if p.Routes != (Routes{}) || len(p.Owners)+len(p.ReviewAuthors)+len(p.RequiredChecks) != 0 {
			return fmt.Errorf("routes, owners, review_authors and required_checks require models")
		}
		return nil
	}
	if len(p.Models) > 256 {
		return fmt.Errorf("policy must contain 1..256 models")
	}
	seen := map[string]bool{}
	for _, model := range p.Models {
		if !identity(model.ID) || !identity(model.Provider) || seen[model.ID] {
			return fmt.Errorf("invalid or duplicate model %q", model.ID)
		}
		seen[model.ID] = true
	}
	for _, model := range p.Models {
		edges := map[string]bool{}
		for _, weaker := range model.StrongerThan {
			if !seen[weaker] || edges[weaker] {
				return fmt.Errorf("model %q has unknown or duplicate comparison %q", model.ID, weaker)
			}
			edges[weaker] = true
		}
	}
	if err := p.validateAcyclic(); err != nil {
		return err
	}
	for _, route := range []Route{p.Routes.Developer, p.Routes.Reviewer} {
		if !seen[route.Model] || !identity(route.Effort) {
			return fmt.Errorf("route has unknown model or unresolved effort")
		}
	}
	if err := uniqueNames("owners", p.Owners); err != nil {
		return err
	}
	if err := uniqueNames("review_authors", p.ReviewAuthors); err != nil {
		return err
	}
	return uniqueNames("required_checks", p.RequiredChecks)
}

// validateTiers rejects malformed, duplicate or unknown-provider tier and band entries.
func (p Policy) validateTiers() error {
	seen := map[string]bool{}
	for _, tier := range [][]string{p.ModelTiers.Frontier, p.ModelTiers.Workhorse, p.ModelTiers.Light} {
		for _, entry := range tier {
			provider, model, ok := strings.Cut(entry, ":")
			key := strings.ToLower(entry)
			if !ok || !supportedProvider(provider) || !identity(model) || seen[key] {
				return fmt.Errorf("model tier entry %q must be a unique claude:, codex: or gemini: model", entry)
			}
			seen[key] = true
		}
	}
	for provider, bands := range map[string]Bands{"claude": p.EffortBands.Claude, "codex": p.EffortBands.Codex, "gemini": p.EffortBands.Gemini} {
		values := map[string]bool{}
		for _, band := range [][]string{bands.Low, bands.Mid, bands.High} {
			for _, value := range band {
				if !identity(value) || values[strings.ToLower(value)] {
					return fmt.Errorf("%s effort value %q must be unique", provider, value)
				}
				values[strings.ToLower(value)] = true
			}
		}
		if len(values) == 0 {
			return fmt.Errorf("effort_bands must map %s effort values", provider)
		}
	}
	return nil
}

func supportedProvider(provider string) bool {
	return provider == "claude" || provider == "codex" || provider == "gemini"
}

func (p Policy) validateAcyclic() error {
	colors := map[string]int{}
	var visit func(string) error
	visit = func(id string) error {
		if colors[id] == 1 {
			return fmt.Errorf("model comparisons contain a cycle at %q", id)
		}
		if colors[id] == 2 {
			return nil
		}
		colors[id] = 1
		model, _ := p.Model(id)
		for _, weaker := range model.StrongerThan {
			if err := visit(weaker); err != nil {
				return err
			}
		}
		colors[id] = 2
		return nil
	}
	for _, model := range p.Models {
		if err := visit(model.ID); err != nil {
			return err
		}
	}
	return nil
}

func uniqueNames(label string, values []string) error {
	if len(values) == 0 {
		return fmt.Errorf("%s must not be empty", label)
	}
	seen := map[string]bool{}
	for _, value := range values {
		if !identity(value) || seen[value] {
			return fmt.Errorf("%s contains invalid or duplicate value %q", label, value)
		}
		seen[value] = true
	}
	return nil
}

func identity(s string) bool {
	return s != "" && len(s) <= 512 && s == strings.TrimSpace(s) && !strings.EqualFold(s, "inherit") && !strings.ContainsAny(s, "\x00\r\n\t")
}

func (p Policy) stronger(reviewer, developer string) bool {
	seen := map[string]bool{}
	var visit func(string) bool
	visit = func(id string) bool {
		if seen[id] {
			return false
		}
		seen[id] = true
		model, _ := p.Model(id)
		for _, weaker := range model.StrongerThan {
			if weaker == developer || visit(weaker) {
				return true
			}
		}
		return false
	}
	return reviewer != developer && visit(reviewer)
}

// ValidateRouting checks eligibility of the configured routes before implementation.
func (p Policy) ValidateRouting() error {
	dev, _ := p.Model(p.Routes.Developer.Model)
	reviewer, _ := p.Model(p.Routes.Reviewer.Model)
	return Eligible(p, []Run{{ID: "developer", Model: dev.ID, Provider: dev.Provider, Effort: p.Routes.Developer.Effort}}, Run{ID: "reviewer", Model: reviewer.ID, Provider: reviewer.Provider, Effort: p.Routes.Reviewer.Effort})
}
