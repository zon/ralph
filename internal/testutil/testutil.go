package testutil

import "github.com/zon/ralph/internal/context"

// NewContext creates a standard test context with safe defaults.
// All tests should use this to ensure consistent configuration:
// - NoNotify: true (prevents real desktop notifications)
// - NoServices: false (allows service testing)
// - Verbose: false (reduces test output noise)
// - MaxIterations: 10 (reasonable default)
//
// Use options to customize specific fields as needed.
func NewContext(opts ...ContextOption) *context.Context {
	ctx := &context.Context{
		ProjectFile:   "",
		MaxIterations: 10,
		DryRun:        true,
		Verbose:       false,
		NoNotify:      true,
		NoServices:    false,
		Local:         true,
	}

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
		ctx.ProjectFile = path
	}
}

// WithMaxIterations sets the maximum iterations
func WithMaxIterations(max int) ContextOption {
	return func(ctx *context.Context) {
		ctx.MaxIterations = max
	}
}

// WithDryRun sets the dry-run flag
func WithDryRun(dryRun bool) ContextOption {
	return func(ctx *context.Context) {
		ctx.DryRun = dryRun
	}
}

// WithVerbose sets the verbose flag
func WithVerbose(verbose bool) ContextOption {
	return func(ctx *context.Context) {
		ctx.Verbose = verbose
	}
}

// WithNoNotify sets the no-notify flag
// Note: Tests should always use NoNotify: true (the default)
func WithNoNotify(noNotify bool) ContextOption {
	return func(ctx *context.Context) {
		ctx.NoNotify = noNotify
	}
}

// WithNoServices sets the no-services flag
func WithNoServices(noServices bool) ContextOption {
	return func(ctx *context.Context) {
		ctx.NoServices = noServices
	}
}

// WithInstructions sets the instructions file path
func WithInstructions(instructions string) ContextOption {
	return func(ctx *context.Context) {
		ctx.Instructions = instructions
	}
}
