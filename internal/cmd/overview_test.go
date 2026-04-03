package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEmbeddedOverviewInstructions(t *testing.T) {
	if overviewInstructions == "" {
		t.Error("overviewInstructions should not be empty")
	}
	if !contains(overviewInstructions, "{{.OverviewPath}}") {
		t.Error("overviewInstructions should contain OverviewPath template variable")
	}
}

func TestEmbeddedComponentReviewInstructions(t *testing.T) {
	if componentReviewInstructions == "" {
		t.Error("componentReviewInstructions should not be empty")
	}
	if !contains(componentReviewInstructions, "{{.ComponentName}}") {
		t.Error("componentReviewInstructions should contain ComponentName template variable")
	}
	if !contains(componentReviewInstructions, "{{.ConfigContent}}") {
		t.Error("componentReviewInstructions should contain ConfigContent template variable")
	}
}

func TestBuildOverviewPrompt(t *testing.T) {
	tests := []struct {
		name         string
		overviewPath string
		wantContains string
	}{
		{
			name:         "prompt contains overview path",
			overviewPath: "projects/review-2024-01-01-overview.json",
			wantContains: "projects/review-2024-01-01-overview.json",
		},
		{
			name:         "prompt contains JSON format instruction",
			overviewPath: "/tmp/overview.json",
			wantContains: "JSON format",
		},
		{
			name:         "prompt contains modules field",
			overviewPath: "overview.json",
			wantContains: "modules",
		},
		{
			name:         "prompt contains apps field",
			overviewPath: "overview.json",
			wantContains: "apps",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prompt := buildOverviewPrompt(tt.overviewPath)
			if !contains(prompt, tt.wantContains) {
				t.Errorf("buildOverviewPrompt() = %q, want it to contain %q", prompt, tt.wantContains)
			}
		})
	}
}

func TestBuildComponentPrompt(t *testing.T) {
	tests := []struct {
		name         string
		content      string
		projectDoc   string
		component    OverviewComponent
		summaryPath  string
		wantContains []string
	}{
		{
			name:       "prompt contains component name",
			content:    "Check for security issues",
			projectDoc: "Project documentation",
			component: OverviewComponent{
				Name:    "auth",
				Path:    "internal/auth",
				Summary: "Handles authentication and authorization",
			},
			summaryPath:  "tmp/summary-auth-0.txt",
			wantContains: []string{"auth", "internal/auth", "Handles authentication and authorization", "Check for security issues", "tmp/summary-auth-0.txt", "projects/"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prompt := buildComponentPrompt(tt.content, tt.projectDoc, tt.component, tt.summaryPath)
			for _, want := range tt.wantContains {
				if !contains(prompt, want) {
					t.Errorf("buildComponentPrompt() = %q, want it to contain %q", prompt, want)
				}
			}
		})
	}
}

func TestLoadOverview_Valid(t *testing.T) {
	tests := []struct {
		name        string
		jsonContent string
		wantModules int
		wantApps    int
	}{
		{
			name: "parses valid JSON with modules and apps",
			jsonContent: `{
  "modules": [
    {
      "name": "auth",
      "path": "internal/auth",
      "summary": "Handles authentication"
    },
    {
      "name": "api",
      "path": "internal/api",
      "summary": "REST API handlers"
    }
  ],
  "apps": [
    {
      "name": "ralph",
      "path": "cmd/ralph",
      "summary": "Main CLI entry point"
    }
  ]
}`,
			wantModules: 2,
			wantApps:    1,
		},
		{
			name: "parses JSON with only modules",
			jsonContent: `{
  "modules": [
    {
      "name": "core",
      "path": "internal/core",
      "summary": "Core business logic"
    }
  ],
  "apps": []
}`,
			wantModules: 1,
			wantApps:    0,
		},
		{
			name: "parses JSON with only apps",
			jsonContent: `{
  "modules": [],
  "apps": [
    {
      "name": "worker",
      "path": "cmd/worker",
      "summary": "Background worker"
    }
  ]
}`,
			wantModules: 0,
			wantApps:    1,
		},
		{
			name:        "parses empty modules and apps lists",
			jsonContent: `{"modules": [], "apps": []}`,
			wantModules: 0,
			wantApps:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			filePath := filepath.Join(tmpDir, "overview.json")
			if err := os.WriteFile(filePath, []byte(tt.jsonContent), 0644); err != nil {
				t.Fatalf("failed to write test file: %v", err)
			}

			overview, err := loadOverview(filePath)
			if err != nil {
				t.Fatalf("loadOverview() error = %v", err)
			}

			if len(overview.Modules) != tt.wantModules {
				t.Errorf("got %d modules, want %d", len(overview.Modules), tt.wantModules)
			}
			if len(overview.Apps) != tt.wantApps {
				t.Errorf("got %d apps, want %d", len(overview.Apps), tt.wantApps)
			}
		})
	}
}

func TestLoadOverview_Missing(t *testing.T) {
	tests := []struct {
		name     string
		filePath string
		wantErr  bool
	}{
		{
			name:     "returns error for missing file",
			filePath: "/nonexistent/path/overview.json",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := loadOverview(tt.filePath)
			if (err != nil) != tt.wantErr {
				t.Errorf("loadOverview() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLoadOverview_WithColons(t *testing.T) {
	tests := []struct {
		name        string
		jsonContent string
		wantModules int
		wantApps    int
	}{
		{
			name: "parses JSON with colons in summary",
			jsonContent: `{
  "modules": [
    {
      "name": "http",
      "path": "internal/http",
      "summary": "Handles HTTP: requests, responses, and middleware"
    },
    {
      "name": "config",
      "path": "internal/config",
      "summary": "Loads config from: env vars, files, and remote sources"
    }
  ],
  "apps": []
}`,
			wantModules: 2,
			wantApps:    0,
		},
		{
			name: "parses JSON with URLs in summary",
			jsonContent: `{
  "modules": [
    {
      "name": "api",
      "path": "internal/api",
      "summary": "Provides REST API at https://api.example.com/v1"
    }
  ],
  "apps": []
}`,
			wantModules: 1,
			wantApps:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			filePath := filepath.Join(tmpDir, "overview.json")
			if err := os.WriteFile(filePath, []byte(tt.jsonContent), 0644); err != nil {
				t.Fatalf("failed to write test file: %v", err)
			}

			overview, err := loadOverview(filePath)
			if err != nil {
				t.Fatalf("loadOverview() error = %v", err)
			}

			if len(overview.Modules) != tt.wantModules {
				t.Errorf("got %d modules, want %d", len(overview.Modules), tt.wantModules)
			}
			if len(overview.Apps) != tt.wantApps {
				t.Errorf("got %d apps, want %d", len(overview.Apps), tt.wantApps)
			}
		})
	}
}

func TestAllComponents(t *testing.T) {
	tests := []struct {
		name           string
		modules        []OverviewComponent
		apps           []OverviewComponent
		wantTotal      int
		wantFirstIsMod bool
	}{
		{
			name: "combines modules and apps",
			modules: []OverviewComponent{
				{Name: "mod1", Path: "internal/mod1", Summary: "Module 1"},
			},
			apps: []OverviewComponent{
				{Name: "app1", Path: "cmd/app1", Summary: "App 1"},
			},
			wantTotal:      2,
			wantFirstIsMod: true,
		},
		{
			name:           "empty modules and apps",
			modules:        []OverviewComponent{},
			apps:           []OverviewComponent{},
			wantTotal:      0,
			wantFirstIsMod: true,
		},
		{
			name: "modules only",
			modules: []OverviewComponent{
				{Name: "mod1", Path: "internal/mod1", Summary: "Module 1"},
				{Name: "mod2", Path: "internal/mod2", Summary: "Module 2"},
			},
			apps:           []OverviewComponent{},
			wantTotal:      2,
			wantFirstIsMod: true,
		},
		{
			name:    "apps only",
			modules: []OverviewComponent{},
			apps: []OverviewComponent{
				{Name: "app1", Path: "cmd/app1", Summary: "App 1"},
			},
			wantTotal:      1,
			wantFirstIsMod: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			overview := &Overview{
				Modules: tt.modules,
				Apps:    tt.apps,
			}
			all := overview.AllComponents()
			if len(all) != tt.wantTotal {
				t.Errorf("AllComponents() returned %d components, want %d", len(all), tt.wantTotal)
			}
			if tt.wantTotal > 0 {
				if tt.wantFirstIsMod && all[0].Name != "mod1" {
					t.Errorf("first component should be from modules, got %s", all[0].Name)
				}
				if !tt.wantFirstIsMod && all[0].Name != "app1" {
					t.Errorf("first component should be from apps, got %s", all[0].Name)
				}
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
