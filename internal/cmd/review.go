package cmd

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/zon/ralph/internal/ai"
	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/logger"
)

const (
	defaultProjectFile = "projects/review.yaml"
	ralphProjectDocURL = "https://raw.githubusercontent.com/zon/ralph/refs/heads/main/docs/projects.md"
)

type ReviewCmd struct {
	ProjectFile string `help:"Path to output project YAML file" name:"project" short:"p" default:"projects/review.yaml"`
	Model       string `help:"Override the AI model from config" name:"model" optional:""`
	Local       bool   `help:"Run on this machine instead of submitting to Argo Workflows" default:"false"`
	Verbose     bool   `help:"Enable verbose logging" default:"false"`
}

func (r *ReviewCmd) Run() error {
	if r.Verbose {
		logger.SetVerbose(true)
	}

	absProjectFile, err := filepath.Abs(r.ProjectFile)
	if err != nil {
		return fmt.Errorf("failed to resolve project file path: %w", err)
	}

	projectDir := filepath.Dir(absProjectFile)
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		return fmt.Errorf("failed to create project directory: %w", err)
	}

	ralphConfig, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if len(ralphConfig.Review.Items) == 0 {
		return fmt.Errorf("no review items found in config")
	}

	projectDoc, err := fetchRalphProjectDoc()
	if err != nil {
		logger.Verbosef("Failed to fetch Ralph project doc: %v", err)
		projectDoc = ""
	}

	model := r.resolveModel(ralphConfig)

	ctx := createExecutionContext()
	ctx.SetProjectFile(absProjectFile)
	ctx.SetVerbose(r.Verbose)
	ctx.SetModel(model)
	ctx.SetLocal(r.Local)

	for i, item := range ralphConfig.Review.Items {
		content, err := r.loadItemContent(item)
		if err != nil {
			return fmt.Errorf("failed to load review item %d: %w", i, err)
		}

		prompt := r.buildPrompt(content, absProjectFile, projectDoc)

		if r.Verbose {
			logger.Verbose(prompt)
		}

		logger.Verbosef("Running review item %d/%d...", i+1, len(ralphConfig.Review.Items))
		if err := ai.RunAgent(ctx, prompt); err != nil {
			return fmt.Errorf("review item %d failed: %w", i, err)
		}
	}

	return nil
}

func (r *ReviewCmd) resolveModel(ralphConfig *config.RalphConfig) string {
	if r.Model != "" {
		return r.Model
	}
	if ralphConfig.Review.Model != "" {
		return ralphConfig.Review.Model
	}
	return ralphConfig.Model
}

func (r *ReviewCmd) loadItemContent(item config.ReviewItem) (string, error) {
	switch {
	case item.Text != "":
		return item.Text, nil
	case item.File != "":
		absPath, err := filepath.Abs(item.File)
		if err != nil {
			return "", fmt.Errorf("failed to resolve file path: %w", err)
		}
		data, err := os.ReadFile(absPath)
		if err != nil {
			return "", fmt.Errorf("failed to read file: %w", err)
		}
		return string(data), nil
	case item.URL != "":
		resp, err := http.Get(item.URL)
		if err != nil {
			return "", fmt.Errorf("failed to fetch URL: %w", err)
		}
		defer resp.Body.Close()
		data, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", fmt.Errorf("failed to read response: %w", err)
		}
		return string(data), nil
	default:
		return "", fmt.Errorf("review item has no content")
	}
}

type reviewPromptData struct {
	ConfigContent   string
	Project         string
	RalphProjectDoc string
}

var reviewPromptTemplate = template.Must(template.New("review").Parse(`You are a software architect reviewing source code. Does the code meet these standards?

## Review Content
{{.ConfigContent}}

## Instructions
Create or edit the ralph project at {{.Project}} with any issues found.

{{.RalphProjectDoc}}
`))

func (r *ReviewCmd) buildPrompt(content, projectPath, projectDoc string) string {
	var buf bytes.Buffer
	data := reviewPromptData{
		ConfigContent:   content,
		Project:         projectPath,
		RalphProjectDoc: projectDoc,
	}
	reviewPromptTemplate.Execute(&buf, data)
	return buf.String()
}

func fetchRalphProjectDoc() (string, error) {
	resp, err := http.Get(ralphProjectDocURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}
