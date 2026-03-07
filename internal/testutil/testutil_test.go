package testutil

import "testing"

func TestNewContext(t *testing.T) {
	ctx := NewContext()

	// Verify safe defaults
	if ctx.MaxIterations() != 10 {
		t.Errorf("Expected MaxIterations=10, got %d", ctx.MaxIterations())
	}

	if !ctx.IsDryRun() {
		t.Error("Expected IsDryRun()=true by default")
	}

	if ctx.IsVerbose() {
		t.Error("Expected IsVerbose()=false by default")
	}

	// NoNotify=true means ShouldNotify returns false
	if ctx.ShouldNotify() {
		t.Error("Expected ShouldNotify()=false when NoNotify=true")
	}

	// NoServices=false (default) means ShouldStartServices returns true
	if !ctx.ShouldStartServices() {
		t.Error("Expected ShouldStartServices()=true when NoServices=false")
	}

	if ctx.ProjectFile() != "" {
		t.Errorf("Expected ProjectFile='', got '%s'", ctx.ProjectFile())
	}
}

func TestNewContext_WithOptions(t *testing.T) {
	ctx := NewContext(
		WithProjectFile("/path/to/project.yaml"),
		WithMaxIterations(5),
		WithDryRun(false),
		WithVerbose(true),
	)

	if ctx.ProjectFile() != "/path/to/project.yaml" {
		t.Errorf("Expected ProjectFile='/path/to/project.yaml', got '%s'", ctx.ProjectFile())
	}

	if ctx.MaxIterations() != 5 {
		t.Errorf("Expected MaxIterations=5, got %d", ctx.MaxIterations())
	}

	if ctx.IsDryRun() {
		t.Error("Expected IsDryRun()=false")
	}

	if !ctx.IsVerbose() {
		t.Error("Expected IsVerbose()=true")
	}

	// NoNotify is still true from default, so ShouldNotify is false
	if ctx.ShouldNotify() {
		t.Error("Expected ShouldNotify()=false (default NoNotify=true)")
	}
}

func TestNewContext_MultipleOptions(t *testing.T) {
	// Test that multiple options can be applied
	ctx := NewContext(
		WithDryRun(false),
		WithMaxIterations(20),
		WithNoServices(true),
	)

	if ctx.IsDryRun() {
		t.Error("Expected IsDryRun()=false")
	}

	if ctx.MaxIterations() != 20 {
		t.Errorf("Expected MaxIterations=20, got %d", ctx.MaxIterations())
	}

	// WithNoServices(true), noServices becomes true, so ShouldStartServices returns false
	if ctx.ShouldStartServices() {
		t.Error("Expected ShouldStartServices()=false when NoServices=true")
	}
}
