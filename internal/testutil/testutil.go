package testutil

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/zon/ralph/internal/context"
)

// Contains reports whether substr is in s.
func Contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

// NewContext creates a standard test context with safe defaults.
// All tests should use this to ensure consistent configuration:
// - NoNotify: true (prevents real desktop notifications)
// - NoServices: false (allows service testing)
// - Verbose: false (reduces test output noise)
// - MaxIterations: 10 (reasonable default)
//
// Use options to customize specific fields as needed.
func NewContext(opts ...ContextOption) *context.Context {
	ctx := &context.Context{}
	ctx.SetDryRun(true)
	ctx.SetNoNotify(true)
	ctx.SetLocal(true)
	ctx.SetMaxIterations(10)

	for _, opt := range opts {
		opt(ctx)
	}

	return ctx
}

// ContextOption is a function that modifies a context
type ContextOption func(*context.Context)

// WithProjectFile sets the project file path
func WithProjectFile(path string) ContextOption {
	return func(ctx *context.Context) {
		ctx.SetProjectFile(path)
	}
}

// WithMaxIterations sets the maximum iterations
func WithMaxIterations(max int) ContextOption {
	return func(ctx *context.Context) {
		ctx.SetMaxIterations(max)
	}
}

// WithDryRun sets the dry-run flag
func WithDryRun(dryRun bool) ContextOption {
	return func(ctx *context.Context) {
		ctx.SetDryRun(dryRun)
	}
}

// WithVerbose sets the verbose flag
func WithVerbose(verbose bool) ContextOption {
	return func(ctx *context.Context) {
		ctx.SetVerbose(verbose)
	}
}

// WithNoNotify sets the no-notify flag
// Note: Tests should always use NoNotify: true (the default)
func WithNoNotify(noNotify bool) ContextOption {
	return func(ctx *context.Context) {
		ctx.SetNoNotify(noNotify)
	}
}

// WithNoServices sets the no-services flag
func WithNoServices(noServices bool) ContextOption {
	return func(ctx *context.Context) {
		ctx.SetNoServices(noServices)
	}
}

// WithInstructions sets the instructions file path
func WithInstructions(instructions string) ContextOption {
	return func(ctx *context.Context) {
		ctx.SetInstructions(instructions)
	}
}

// E2EConfig holds configuration for end-to-end tests resolved from environment variables.
type E2EConfig struct {
	// Repo is the owner/repo of the dedicated test repository (e.g. "zon/ralph-mock").
	Repo string
	// Branch is the branch the workflow container will clone (typically "main").
	Branch string
	// DebugBranch is the ralph source branch to use inside the container via `go run`.
	DebugBranch string
	// Namespace is the Argo Workflows Kubernetes namespace.
	Namespace string
	// Timeout is the maximum time to wait for a single workflow to complete.
	Timeout time.Duration
}

// NewE2EConfig reads E2E configuration from environment variables, applying safe defaults.
// It calls t.Fatal if the required RALPH_E2E_REPO variable is not set.
//
// Environment variables:
//
//	RALPH_E2E_REPO          owner/repo of the test repository (default: "zon/ralph-mock")
//	RALPH_E2E_BRANCH        branch to clone in the container (default: "main")
//	RALPH_E2E_DEBUG_BRANCH  ralph source branch for go run mode (default: "main")
//	RALPH_E2E_NAMESPACE     Argo namespace (default: "ralph-mock")
//	RALPH_E2E_TIMEOUT       per-workflow poll timeout as a Go duration (default: "10m")
func NewE2EConfig(t *testing.T) *E2EConfig {
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

// NewE2EContext creates an execution context suitable for E2E tests. It resolves
// configuration from environment variables via NewE2EConfig and returns a context
// with DryRun disabled, Local disabled (remote workflow submission), and the
// test repository and branch overrides populated.
//
// opts may be used to override individual fields after the E2E defaults are applied.
func NewE2EContext(t *testing.T, opts ...ContextOption) (*context.Context, *E2EConfig) {
	t.Helper()

	cfg := NewE2EConfig(t)

	ctx := &context.Context{}
	ctx.SetRepo(cfg.Repo)
	ctx.SetBranch(cfg.Branch)
	ctx.SetDebugBranch(cfg.DebugBranch)
	ctx.SetDryRun(false)
	ctx.SetLocal(false)
	ctx.SetVerbose(true)
	ctx.SetNoNotify(true)
	ctx.SetNoServices(true)

	for _, opt := range opts {
		opt(ctx)
	}

	return ctx, cfg
}
