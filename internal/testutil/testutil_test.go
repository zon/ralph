package testutil

import "testing"

func TestNewContext(t *testing.T) {
	ctx := NewContext()

	// Verify safe defaults
	if ctx.MaxIterations != 10 {
		t.Errorf("Expected MaxIterations=10, got %d", ctx.MaxIterations)
	}

	if !ctx.DryRun {
		t.Error("Expected DryRun=true by default")
	}

	if ctx.Verbose {
		t.Error("Expected Verbose=false by default")
	}

	if !ctx.NoNotify {
		t.Error("Expected NoNotify=true by default (critical for tests)")
	}

	if ctx.NoServices {
		t.Error("Expected NoServices=false by default")
	}

	if ctx.ProjectFile != "" {
		t.Errorf("Expected ProjectFile='', got '%s'", ctx.ProjectFile)
	}
}

func TestNewContext_WithOptions(t *testing.T) {
	ctx := NewContext(
		WithProjectFile("/path/to/project.yaml"),
		WithMaxIterations(5),
		WithDryRun(false),
		WithVerbose(true),
	)

	if ctx.ProjectFile != "/path/to/project.yaml" {
		t.Errorf("Expected ProjectFile='/path/to/project.yaml', got '%s'", ctx.ProjectFile)
	}

	if ctx.MaxIterations != 5 {
		t.Errorf("Expected MaxIterations=5, got %d", ctx.MaxIterations)
	}

	if ctx.DryRun {
		t.Error("Expected DryRun=false")
	}

	if !ctx.Verbose {
		t.Error("Expected Verbose=true")
	}

	// NoNotify should still be true (safe default preserved)
	if !ctx.NoNotify {
		t.Error("Expected NoNotify=true (safe default)")
	}
}

func TestNewContext_MultipleOptions(t *testing.T) {
	// Test that multiple options can be applied
	ctx := NewContext(
		WithDryRun(false),
		WithMaxIterations(20),
		WithNoServices(true),
	)

	if ctx.DryRun {
		t.Error("Expected DryRun=false")
	}

	if ctx.MaxIterations != 20 {
		t.Errorf("Expected MaxIterations=20, got %d", ctx.MaxIterations)
	}

	if !ctx.NoServices {
		t.Error("Expected NoServices=true")
	}
}
