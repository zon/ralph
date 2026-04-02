package ai

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/context"
	"github.com/zon/ralph/internal/fileutil"
	"github.com/zon/ralph/internal/logger"
)

const mockAIEnv = "RALPH_MOCK_AI"

func resolveModel(ctx *context.Context) string {
	if ctx.Model() != "" {
		return ctx.Model()
	}
	ralphConfig, err := config.LoadConfig()
	if err != nil {
		return "deepseek/deepseek-chat"
	}
	return ralphConfig.Model
}

// RunAgent executes an AI agent with the given prompt using OpenCode CLI
// OpenCode manages its own configuration for API keys and models
func RunAgent(ctx *context.Context, prompt string) error {
	if os.Getenv(mockAIEnv) == "true" {
		return runMockAgent(ctx, prompt)
	}

	if ctx.IsVerbose() {
		logger.Verbose(prompt)
	}

	model := resolveModel(ctx)

	cmd := exec.Command("opencode", "run", "--model", model, prompt)
	cmd.Env = append(os.Environ(), "FORCE_COLOR=1")

	ring := &ringWriter{n: 10}
	cmd.Stdout = io.MultiWriter(os.Stdout, ring)
	cmd.Stderr = io.MultiWriter(os.Stderr, ring)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("opencode execution failed: %w\n\nLast 10 lines of output:\n%s", err, ring.tail())
	}

	return nil
}

// ringWriter captures the last n lines written to it.
type ringWriter struct {
	n     int
	lines []string
	buf   string // partial line not yet terminated
}

func (r *ringWriter) Write(p []byte) (int, error) {
	s := r.buf + string(p)
	parts := strings.Split(s, "\n")
	// last element is either "" (if s ended with \n) or a partial line
	r.buf = parts[len(parts)-1]
	for _, line := range parts[:len(parts)-1] {
		r.lines = append(r.lines, line)
		if len(r.lines) > r.n {
			r.lines = r.lines[1:]
		}
	}
	return len(p), nil
}

func (r *ringWriter) tail() string {
	lines := r.lines
	if r.buf != "" {
		lines = append(lines, r.buf)
	}
	return strings.Join(lines, "\n")
}

// RunOpenCodeAndReadResult runs opencode with the given prompt and reads the result from the output file.
func RunOpenCodeAndReadResult(ctx *context.Context, model, prompt, outputFile string) (string, error) {
	cmd := exec.Command("opencode", "run", "--model", model, prompt)
	cmd.Env = append(os.Environ(), "FORCE_COLOR=1")
	if ctx.IsVerbose() {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("opencode execution failed: %w", err)
	}

	// Read the summary from the file the agent wrote
	summaryBytes, err := fileutil.ReadFile(outputFile)
	if err != nil {
		return "", fmt.Errorf("failed to read summary file: %w", err)
	}

	summary := strings.TrimSpace(string(summaryBytes))
	if summary == "" {
		return "", fmt.Errorf("summary file is empty")
	}

	return summary, nil
}

// runMockAgent simulates AI execution for testing purposes.
// It parses the prompt to determine what file to write and creates mock output files.
func runMockAgent(ctx *context.Context, prompt string) error {
	if os.Getenv("RALPH_MOCK_AI_FAIL") == "true" {
		return fmt.Errorf("opencode execution failed: mock AI failure\n\nline 9 output\nline 10 output\nline 11 output\nline 12 output")
	}

	promptLower := strings.ToLower(prompt)

	if strings.Contains(promptLower, "picked-requirement") {
		absProjectFile := ctx.ProjectFile()
		if absProjectFile == "" {
			return fmt.Errorf("mock AI requires project file to be set")
		}

		pickedReqPath := filepath.Join(filepath.Dir(absProjectFile), "picked-requirement.yaml")
		mockReqContent := `- description: Mock requirement
  passing: false
`
		if err := os.WriteFile(pickedReqPath, []byte(mockReqContent), 0644); err != nil {
			return fmt.Errorf("mock AI failed to write picked-requirement.yaml: %w", err)
		}
		logger.Verbosef("Mock AI wrote picked-requirement.yaml")
	}

	if strings.Contains(promptLower, "report.md") {
		if err := os.WriteFile("report.md", []byte("Mock: test commit\n"), 0644); err != nil {
			return fmt.Errorf("mock AI failed to write report.md: %w", err)
		}
		logger.Verbosef("Mock AI wrote report.md")
	}

	if strings.Contains(promptLower, "overview") {
		// Find the JSON file path in the prompt
		var jsonPath string
		words := strings.Fields(prompt)
		for _, word := range words {
			if strings.HasSuffix(word, ".json") {
				jsonPath = word
				break
			}
		}
		if jsonPath == "" {
			// fallback to a default path
			jsonPath = "overview.json"
		}
		overview := struct {
			Components []struct {
				Name    string `json:"name"`
				Path    string `json:"path"`
				Summary string `json:"summary"`
			} `json:"components"`
		}{
			Components: []struct {
				Name    string `json:"name"`
				Path    string `json:"path"`
				Summary string `json:"summary"`
			}{
				{Name: "mock-component", Path: "internal/mock", Summary: "Mock component for testing"},
			},
		}
		data, err := json.Marshal(overview)
		if err != nil {
			return fmt.Errorf("mock AI failed to marshal overview: %w", err)
		}
		if err := os.WriteFile(jsonPath, data, 0644); err != nil {
			return fmt.Errorf("mock AI failed to write overview JSON: %w", err)
		}
		logger.Verbosef("Mock AI wrote overview JSON to %s", jsonPath)
	}

	// Modify project file to simulate a finding (for testing loop exit)
	absProjectFile := ctx.ProjectFile()
	if absProjectFile != "" {
		f, err := os.OpenFile(absProjectFile, os.O_APPEND|os.O_WRONLY, 0644)
		if err == nil {
			defer f.Close()
			if _, err := f.WriteString("\n# mock modification"); err != nil {
				logger.Verbosef("Mock AI failed to append to project file: %v", err)
			} else {
				logger.Verbosef("Mock AI appended to project file: %s", absProjectFile)
			}
		}
	}

	return nil
}
