package web

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"syscall"

	"github.com/c360studio/ateam/internal/review"
)

// ModelRouting describes configured intent, not runtime identity or PR approval.
type ModelRouting struct {
	Status       string           `json:"status"`
	Reason       string           `json:"reason,omitempty"`
	Developer    *ConfiguredModel `json:"developer,omitempty"`
	Reviewer     *ConfiguredModel `json:"reviewer,omitempty"`
	PolicyDigest string           `json:"policy_digest,omitempty"`
}

// ConfiguredModel keeps model identity and reasoning effort separate.
type ConfiguredModel struct {
	Model    string `json:"model"`
	Provider string `json:"provider"`
	Effort   string `json:"effort"`
}

func readModelRouting(repoRoot string) ModelRouting {
	unavailable := ModelRouting{Status: "unavailable", Reason: "Model routing policy could not be read. Check that it is a readable regular file inside this repository."}
	root, err := os.OpenRoot(repoRoot)
	if err != nil {
		return unavailable
	}
	defer root.Close()
	const path = ".agents/review-policy.json"
	info, err := root.Lstat(path)
	if os.IsNotExist(err) {
		return ModelRouting{Status: "unconfigured", Reason: "No model routing policy is configured for this checkout."}
	}
	if err != nil || !info.Mode().IsRegular() {
		return unavailable
	}
	data, err := readRoutingFile(root, path)
	if err != nil {
		return unavailable
	}
	if len(data) > review.MaxJSONBytes {
		return ModelRouting{Status: "invalid", Reason: "Invalid model routing policy: exceeds 1 MiB."}
	}
	policy, err := review.ParsePolicy(data)
	if err != nil {
		return ModelRouting{Status: "invalid", Reason: "Invalid model routing policy: " + err.Error()}
	}
	configured := func(route review.Route) *ConfiguredModel {
		model, _ := policy.Model(route.Model)
		return &ConfiguredModel{Model: model.ID, Provider: model.Provider, Effort: route.Effort}
	}
	result := ModelRouting{Status: "eligible", Developer: configured(policy.Routes.Developer), Reviewer: configured(policy.Routes.Reviewer), PolicyDigest: review.PolicyDigest(data)}
	if err := policy.ValidateRouting(); err != nil {
		result.Status = "blocked"
		result.Reason = err.Error()
	}
	return result
}

func (s *Server) handleModelRouting(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if !s.validateReadRequest(w, r) {
		return
	}
	s.mu.Lock()
	root := s.currentRepo()
	s.mu.Unlock()
	writeJSON(w, http.StatusOK, readModelRouting(root))
}

// readRoutingFile validates the opened descriptor because the path can change
// after Lstat. The caller classifies oversized policy separately from I/O errors.
func readRoutingFile(root *os.Root, path string) ([]byte, error) {
	// Nonblocking open prevents a concurrent replacement with a FIFO from
	// stalling the dashboard before descriptor validation can reject it.
	file, err := root.OpenFile(path, os.O_RDONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("policy is not a regular file")
	}
	return io.ReadAll(io.LimitReader(file, review.MaxJSONBytes+1))
}
