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
// Use options to customize specific fields as needed.
func NewContext(opts ...ContextOption) *context.Context {
	ctx := context.NewContext()
	ctx.SetOutput(output.NewClient(os.Stdout, os.Stderr, false))
	ctx.SetNoNotify(true)
	ctx.SetLocal(true)

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

// WithLocal sets the local flag
func WithLocal(local bool) ContextOption {
	return func(ctx *context.Context) {
		ctx.SetLocal(local)
	}
}

// WithFollow sets the follow flag
func WithFollow(follow bool) ContextOption {
	return func(ctx *context.Context) {
		ctx.SetFollow(follow)
	}
}

// WithInstructions sets the instructions file path
func WithInstructions(instructions string) ContextOption {
	return func(ctx *context.Context) {
		ctx.SetInstructions(instructions)
	}
}


