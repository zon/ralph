package testutil

import (
	"os"

	"github.com/zon/ralph/internal/context"
	"github.com/zon/ralph/internal/output"
)

// NewContext creates a standard test context with safe defaults.
// All tests should use this to ensure consistent configuration:
// - NoNotify: true (prevents real desktop notifications)
// - NoServices: false (allows service testing)
// - Verbose: false (reduces test output noise)
//
// Use options to customize specific fields as needed. The output client is
// created after the options run so its debug mode follows the final verbosity.
func NewContext(opts ...ContextOption) *context.Context {
	ctx := context.NewContext()
	ctx.SetNoNotify(true)
	ctx.SetLocal(true)

	for _, opt := range opts {
		opt(ctx)
	}

	ctx.SetOutput(output.NewClient(os.Stdout, os.Stderr, ctx.IsVerbose()))

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

// WithVerbose sets the verbose flag
func WithVerbose(verbose bool) ContextOption {
	return func(ctx *context.Context) {
		ctx.SetVerbose(verbose)
	}
}

// WithNoServices sets the no-services flag
func WithNoServices(noServices bool) ContextOption {
	return func(ctx *context.Context) {
		ctx.SetNoServices(noServices)
	}
}
