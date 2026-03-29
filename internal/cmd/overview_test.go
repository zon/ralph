package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBuildOverviewPrompt(t *testing.T) {
	tests := []struct {
		name         string
		overviewPath string
		wantContains string
	}{
		{
			name:         "prompt contains overview path",
			overviewPath: "projects/review-2024-01-01-overview.yaml",
			wantContains: "projects/review-2024-01-01-overview.yaml",
		},
		{
			name:         "prompt contains YAML format instruction",
			overviewPath: "/tmp/overview.yaml",
			wantContains: "YAML format",
		},
		{
			name:         "prompt contains components field",
			overviewPath: "overview.yaml",
			wantContains: "components",
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
		projectPath  string
		projectDoc   string
		reviewName   string
		component    OverviewComponent
		summaryPath  string
		wantContains []string
	}{
		{
			name:        "prompt contains component name",
			content:     "Check for security issues",
			projectPath: "projects/review.yaml",
			projectDoc:  "Project documentation",
			reviewName:  "review-2024-01-01",
			component: OverviewComponent{
				Name:    "auth",
				Path:    "internal/auth",
				Summary: "Handles authentication and authorization",
			},
			summaryPath:  "tmp/summary-auth-0.txt",
			wantContains: []string{"auth", "internal/auth", "Handles authentication and authorization", "projects/review.yaml", "review-2024-01-01", "Check for security issues", "tmp/summary-auth-0.txt"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prompt := buildComponentPrompt(tt.content, tt.projectPath, tt.projectDoc, tt.reviewName, tt.component, tt.summaryPath)
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
		name           string
		yamlContent    string
		wantComponents int
	}{
		{
			name: "parses valid YAML with components",
			yamlContent: `components:
  - name: auth
    path: internal/auth
    summary: Handles authentication
  - name: api
    path: internal/api
    summary: REST API handlers
`,
			wantComponents: 2,
		},
		{
			name: "parses YAML with single component",
			yamlContent: `components:
  - name: core
    path: internal/core
    summary: Core business logic
`,
			wantComponents: 1,
		},
		{
			name:           "parses empty components list",
			yamlContent:    "components: []",
			wantComponents: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			filePath := filepath.Join(tmpDir, "overview.yaml")
			if err := os.WriteFile(filePath, []byte(tt.yamlContent), 0644); err != nil {
				t.Fatalf("failed to write test file: %v", err)
			}

			overview, err := loadOverview(filePath)
			if err != nil {
				t.Fatalf("loadOverview() error = %v", err)
			}

			if len(overview.Components) != tt.wantComponents {
				t.Errorf("got %d components, want %d", len(overview.Components), tt.wantComponents)
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
			filePath: "/nonexistent/path/overview.yaml",
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
