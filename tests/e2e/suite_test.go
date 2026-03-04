//go:build e2e

// Package e2e contains end-to-end tests that submit real Argo Workflows against a
// dedicated test repository and require a live Kubernetes cluster with Argo installed.
//
// Run with:
//
//	go test -tags e2e -timeout 15m ./tests/e2e/...
//
// Environment variables (all optional):
//
//	RALPH_E2E_REPO          owner/repo of the test repository (default: "zon/ralph-mock")
//	RALPH_E2E_BRANCH        branch to clone inside the workflow container (default: "main")
//	RALPH_E2E_DEBUG_BRANCH  ralph source branch to use via `go run` inside the container
//	                        (e.g. your current feature branch — defaults to "main")
//	RALPH_E2E_NAMESPACE     Argo namespace (default: "ralph-mock")
//	RALPH_E2E_TIMEOUT       per-workflow poll timeout as a Go duration (default: "10m")
package e2e

import (
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"
)

// E2EConfig holds configuration resolved from environment variables.
type E2EConfig struct {
	// Repo is the owner/repo of the dedicated test repository (e.g. "zon/ralph-mock").
	Repo string
	// Branch is the branch the workflow container will clone (typically "main").
	Branch string
	// DebugBranch is the ralph source branch to use inside the container via `go run`.
	// Set this to the branch under test so the workflow runs the code you're developing.
	DebugBranch string
	// Namespace is the Argo Workflows Kubernetes namespace.
	Namespace string
	// Timeout is the maximum time to wait for a single workflow to complete.
	Timeout time.Duration
}

// resolveConfig reads E2E configuration from environment variables, applying defaults.
// It fails the test immediately if required variables are missing.
func resolveConfig(t *testing.T) *E2EConfig {
	t.Helper()

	repo := os.Getenv("RALPH_E2E_REPO")
	if repo == "" {
		repo = "zon/ralph-mock"
	}

	branch := os.Getenv("RALPH_E2E_BRANCH")
	if branch == "" {
		branch = "main"
	}

	debugBranch := os.Getenv("RALPH_E2E_DEBUG_BRANCH")
	if debugBranch == "" {
		debugBranch = "main"
	}

	namespace := os.Getenv("RALPH_E2E_NAMESPACE")
	if namespace == "" {
		namespace = "ralph-mock"
	}

	timeout := 10 * time.Minute
	if raw := os.Getenv("RALPH_E2E_TIMEOUT"); raw != "" {
		d, err := time.ParseDuration(raw)
		if err != nil {
			t.Fatalf("invalid RALPH_E2E_TIMEOUT %q: %v", raw, err)
		}
		timeout = d
	}

	return &E2EConfig{
		Repo:        repo,
		Branch:      branch,
		DebugBranch: debugBranch,
		Namespace:   namespace,
		Timeout:     timeout,
	}
}

// TestMain validates that required tools (argo, gh) are on PATH before any test runs.
func TestMain(m *testing.M) {
	for _, tool := range []string{"argo", "gh"} {
		if _, err := exec.LookPath(tool); err != nil {
			fmt.Fprintf(os.Stderr, "E2E tests require %q on PATH: %v\n", tool, err)
			os.Exit(1)
		}
	}
	os.Exit(m.Run())
}
