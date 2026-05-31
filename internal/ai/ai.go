package ai

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/zon/ralph/internal/config"
	execcontext "github.com/zon/ralph/internal/context"
	"github.com/zon/ralph/internal/eino"
	"github.com/zon/ralph/internal/git"
	"github.com/zon/ralph/internal/logger"
)

const mockAIEnv = "RALPH_MOCK_AI"

//go:embed pr-summary-instructions.md
var prSummaryInstructions string

//go:embed changelog-instructions.md
var changelogInstructions string

//go:embed review-pr-body-instructions.md
var reviewPRBodyInstructions string

//go:embed architecture-instructions.md
var architectureInstructions string

//go:embed architecture-fix-instructions.md
var architectureFixInstructions string

//go:embed project-fix-instructions.md
var projectFixInstructions string

//go:embed review-instructions.md
var reviewInstructions string

type FixServicePromptData struct {
	Notes       []string
	ServiceName string
	ServiceCmd  string
	ServicePort int
	Error       string
}

type DevelopPromptData struct {
	Notes               []string
	CommitLog           string
	ProjectContent      string
	SelectedRequirement string
	ProjectFilePath     string
	Services            []config.Service
	Instructions        string
}

type PickPromptData struct {
	Notes          []string
	CommitLog      string
	ProjectContent string
	PickedReqPath  string
}

type PRSummaryPromptData struct {
	ProjectDesc   string
	ProjectStatus string
	BaseBranch    string
	CommitLog     string
	AbsPath       string
}

type ChangelogPromptData struct {
	OutputFile string
}

type ReviewPRBodyPromptData struct {
	ProjectName        string
	ProjectDescription string
	Requirements       []string
	AbsPath            string
}

type ArchitecturePromptData struct {
	OutputFile string
}

type ArchitectureFixPromptData struct {
	OutputFile string
	Errors     []string
}

type ReviewItemPromptData struct {
	ItemContent string
}

type LoopItemPromptData struct {
	FunctionName string
	FunctionPath string
}

func executeTemplate(templateContent string, data interface{}) (string, error) {
	tmpl, err := template.New("prompt").Parse(templateContent)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}
	return buf.String(), nil
}

func BuildFixServicePrompt(ctx *execcontext.Context, svc config.Service, svcErr error) (string, error) {
	cmd := svc.Command
	if len(svc.Args) > 0 {
		cmd = fmt.Sprintf("%s %s", svc.Command, strings.Join(svc.Args, " "))
	}

	data := FixServicePromptData{
		Notes:       ctx.Notes(),
		ServiceName: svc.Name,
		ServiceCmd:  cmd,
		ServicePort: svc.Port,
		Error:       svcErr.Error(),
	}

	return executeTemplate(config.DefaultFixServiceInstructions(), data)
}

func BuildDevelopPrompt(data DevelopPromptData) (string, error) {
	tmplData := struct {
		Notes               []string
		CommitLog           string
		ProjectContent      string
		SelectedRequirement string
		ProjectFilePath     string
		Services            []config.Service
	}{
		Notes:               data.Notes,
		CommitLog:           data.CommitLog,
		ProjectContent:      strings.TrimRight(data.ProjectContent, "\n"),
		SelectedRequirement: data.SelectedRequirement,
		ProjectFilePath:     data.ProjectFilePath,
		Services:            data.Services,
	}

	return executeTemplate(data.Instructions, tmplData)
}

func BuildPickPrompt(data PickPromptData) (string, error) {
	tmplData := struct {
		Notes          []string
		CommitLog      string
		ProjectContent string
		PickedReqPath  string
	}{
		Notes:          data.Notes,
		CommitLog:      data.CommitLog,
		ProjectContent: strings.TrimRight(data.ProjectContent, "\n"),
		PickedReqPath:  data.PickedReqPath,
	}

	return executeTemplate(config.DefaultPickInstructions(), tmplData)
}

func BuildPRSummaryPrompt(projectDesc, projectStatus, baseBranch, commitLog, outputFile string) (string, error) {
	absPath, err := filepath.Abs(outputFile)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path: %w", err)
	}

	data := PRSummaryPromptData{
		ProjectDesc:   projectDesc,
		ProjectStatus: projectStatus,
		BaseBranch:    baseBranch,
		CommitLog:     commitLog,
		AbsPath:       absPath,
	}
	return executeTemplate(prSummaryInstructions, data)
}

func BuildChangelogPrompt(outputFile string) (string, error) {
	absPath, err := filepath.Abs(outputFile)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path: %w", err)
	}

	data := ChangelogPromptData{OutputFile: absPath}
	return executeTemplate(changelogInstructions, data)
}

func BuildReviewPRBodyPrompt(projectName, projectDesc string, requirements []string, outputFile string) (string, error) {
	absPath, err := filepath.Abs(outputFile)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path: %w", err)
	}

	data := ReviewPRBodyPromptData{
		ProjectName:        projectName,
		ProjectDescription: projectDesc,
		Requirements:       requirements,
		AbsPath:            absPath,
	}
	return executeTemplate(reviewPRBodyInstructions, data)
}

func BuildArchitecturePrompt(outputFile string) (string, error) {
	absPath, err := filepath.Abs(outputFile)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path: %w", err)
	}

	data := ArchitecturePromptData{OutputFile: absPath}
	return executeTemplate(architectureInstructions, data)
}

func BuildReviewItemPrompt(content string) (string, error) {
	data := ReviewItemPromptData{ItemContent: content}
	return executeTemplate(reviewInstructions, data)
}

func BuildLoopItemPrompt(content, functionName, functionPath string) (string, error) {
	loopData := LoopItemPromptData{
		FunctionName: functionName,
		FunctionPath: functionPath,
	}
	rendered, err := executeTemplate(content, loopData)
	if err != nil {
		return "", err
	}
	return executeTemplate(reviewInstructions, ReviewItemPromptData{ItemContent: rendered})
}

func BuildArchitectureFixPrompt(outputFile string, errors []string) (string, error) {
	absPath, err := filepath.Abs(outputFile)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path: %w", err)
	}

	data := ArchitectureFixPromptData{OutputFile: absPath, Errors: errors}
	return executeTemplate(architectureFixInstructions, data)
}

type ProjectFixPromptData struct {
	ProjectFile string
	LoadError   string
	Content     string
}

func BuildProjectFixPrompt(projectFile string, content []byte, loadErr error) (string, error) {
	absPath, err := filepath.Abs(projectFile)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path: %w", err)
	}

	data := ProjectFixPromptData{ProjectFile: absPath, LoadError: loadErr.Error(), Content: string(content)}
	return executeTemplate(projectFixInstructions, data)
}

func resolveModel(ctx *execcontext.Context) string {
	if ctx.Model() != "" {
		return ctx.Model()
	}
	ralphConfig, err := config.LoadConfig()
	if err != nil {
		return "deepseek/deepseek-chat"
	}
	return ralphConfig.Model
}

func resolveVariant(ctx *execcontext.Context) string {
	if v := ctx.Variant(); v != "" {
		return v
	}
	ralphConfig, err := config.LoadConfig()
	if err != nil {
		return ""
	}
	return ralphConfig.Variant
}

type trackerKey struct{}

func ContextWithTracker(ctx context.Context, tracker *eino.TokenTracker) context.Context {
	return context.WithValue(ctx, trackerKey{}, tracker)
}

func trackerFromContext(ctx context.Context) *eino.TokenTracker {
	t, _ := ctx.Value(trackerKey{}).(*eino.TokenTracker)
	return t
}

func RunAgent(ctx *execcontext.Context, prompt string) error {
	if os.Getenv(mockAIEnv) == "true" {
		return runMockAgent(ctx, prompt)
	}

	if ctx.IsVerbose() {
		logger.Verbose(prompt)
	}

	tracker := trackerFromContext(ctx.GoContext())
	return eino.RunAgent(ctx.GoContext(), resolveModel(ctx), resolveVariant(ctx), prompt, tracker)
}

func RunAgentWithModel(ctx *execcontext.Context, prompt string, model string) error {
	if os.Getenv(mockAIEnv) == "true" {
		return runMockAgent(ctx, prompt)
	}

	if ctx.IsVerbose() {
		logger.Verbose(prompt)
	}

	tracker := trackerFromContext(ctx.GoContext())
	return eino.RunAgent(ctx.GoContext(), model, resolveVariant(ctx), prompt, tracker)
}

// createTempFile creates a temp file under the repo's tmp/ directory so that
// workflow agents, which lack access to /tmp, can read and write it.
func createTempFile(name string) (*os.File, error) {
	path, err := git.TmpPath(name)
	if err != nil {
		return nil, err
	}
	return os.Create(path)
}

func GeneratePRSummary(ctx *execcontext.Context, projectDesc, projectStatus, baseBranch, commitLog string) (summary string, err error) {
	f, err := createTempFile("pr-summary.md")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary PR summary file: %w", err)
	}
	f.Close()
	tmpFile := f.Name()
	defer os.Remove(tmpFile)

	prPrompt, err := BuildPRSummaryPrompt(projectDesc, projectStatus, baseBranch, commitLog, tmpFile)
	if err != nil {
		return "", fmt.Errorf("failed to build PR summary prompt: %w", err)
	}

	if ctx.IsVerbose() {
		logger.Verbose(prPrompt)
	}

	result, err := eino.Complete(ctx.GoContext(), resolveModel(ctx), prPrompt)
	if err != nil {
		return "", fmt.Errorf("model completion failed: %w", err)
	}

	if err := os.WriteFile(tmpFile, []byte(result), 0644); err != nil {
		return "", fmt.Errorf("failed to write summary file: %w", err)
	}

	return strings.TrimSpace(result), nil
}

func GenerateChangelog(ctx *execcontext.Context) (err error) {
	f, err := createTempFile("changelog.md")
	if err != nil {
		return fmt.Errorf("failed to create temporary changelog file: %w", err)
	}
	f.Close()
	tmpFile := f.Name()
	defer os.Remove(tmpFile)

	changelogPrompt, err := BuildChangelogPrompt(tmpFile)
	if err != nil {
		return fmt.Errorf("failed to build changelog prompt: %w", err)
	}

	if ctx.IsVerbose() {
		logger.Verbose(changelogPrompt)
	}

	result, err := eino.Complete(ctx.GoContext(), resolveModel(ctx), changelogPrompt)
	if err != nil {
		return fmt.Errorf("model completion failed: %w", err)
	}

	if err := os.WriteFile(tmpFile, []byte(result), 0644); err != nil {
		return fmt.Errorf("failed to write changelog file: %w", err)
	}

	if err = os.Rename(tmpFile, "report.md"); err != nil {
		return fmt.Errorf("failed to rename changelog to report.md: %w", err)
	}

	return nil
}

func GenerateReviewPRBody(ctx *execcontext.Context, projectName, projectDesc string, requirementSummaries []string) (summary string, err error) {
	f, err := createTempFile("review-pr-body.md")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary review PR body file: %w", err)
	}
	f.Close()
	tmpFile := f.Name()
	defer os.Remove(tmpFile)

	reviewPrompt, err := BuildReviewPRBodyPrompt(projectName, projectDesc, requirementSummaries, tmpFile)
	if err != nil {
		return "", fmt.Errorf("failed to build review PR body prompt: %w", err)
	}

	if ctx.IsVerbose() {
		logger.Verbose(reviewPrompt)
	}

	result, err := eino.Complete(ctx.GoContext(), resolveModel(ctx), reviewPrompt)
	if err != nil {
		return "", fmt.Errorf("model completion failed: %w", err)
	}

	if err := os.WriteFile(tmpFile, []byte(result), 0644); err != nil {
		return "", fmt.Errorf("failed to write review PR body file: %w", err)
	}

	return strings.TrimSpace(result), nil
}


// runMockAgent simulates AI execution for testing purposes.
// It parses the prompt to determine what file to write and creates mock output files.
func runMockAgent(ctx *execcontext.Context, prompt string) error {
	if os.Getenv("RALPH_MOCK_AI_FAIL") == "true" {
		logger.Verbosef("Mock AI failing as requested")
		return fmt.Errorf("opencode execution failed: mock AI failure\n\nline 9 output\nline 10 output\nline 11 output\nline 12 output")
	}

	promptLower := strings.ToLower(prompt)

	if strings.Contains(promptLower, "picked-requirement") {
		absProjectFile := ctx.ProjectFile()
		if absProjectFile == "" {
			return fmt.Errorf("mock AI requires project file to be set")
		}

		pickedReqPath := filepath.Join(filepath.Dir(absProjectFile), "picked-requirement.yaml")
		mockReqContent := `- slug: mock-requirement
  description: Mock requirement
  items:
    - Mock item
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
		var jsonPath string
		words := strings.Fields(prompt)
		for _, word := range words {
			if strings.HasSuffix(word, ".json") {
				jsonPath = word
				break
			}
		}
		if jsonPath == "" {
			jsonPath = "overview.json"
		}
		overview := struct {
			Modules []struct {
				Name    string `json:"name"`
				Path    string `json:"path"`
				Summary string `json:"summary"`
			} `json:"modules"`
			Apps []struct {
				Name    string `json:"name"`
				Path    string `json:"path"`
				Summary string `json:"summary"`
			} `json:"apps"`
		}{
			Modules: []struct {
				Name    string `json:"name"`
				Path    string `json:"path"`
				Summary string `json:"summary"`
			}{
				{Name: "mock-module", Path: "internal/mock", Summary: "Mock module for testing"},
			},
			Apps: []struct {
				Name    string `json:"name"`
				Path    string `json:"path"`
				Summary string `json:"summary"`
			}{
				{Name: "mock-app", Path: "cmd/mock", Summary: "Mock app for testing"},
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

	if strings.Contains(promptLower, "projects/") {
		if err := os.MkdirAll("projects", 0755); err != nil {
			return fmt.Errorf("mock AI failed to create projects directory: %w", err)
		}
		mockProjectContent := `slug: mock-review
title: Mock project for testing
requirements:
  - slug: mock-requirement
    description: Mock requirement
    items:
      - Mock item
    passing: true
`
		projectPath := filepath.Join("projects", "mock-review.yaml")
		if err := os.WriteFile(projectPath, []byte(mockProjectContent), 0644); err != nil {
			return fmt.Errorf("mock AI failed to write project file: %w", err)
		}
		logger.Verbosef("Mock AI wrote project file to %s", projectPath)
	} else {
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
	}

	return nil
}

var fatalAIPatterns = []string{
	// Generic billing/quota patterns
	"billing",
	"account",
	"payment required",
	"Insufficient Balance",
	// Anthropic
	"credit balance is too low",
	"overloaded",
	"rate_limit_error",
	// Google
	"RESOURCE_EXHAUSTED",
	"quota exceeded",
	// DeepSeek
	"insufficient balance",
	"rate limit",
}

// IsFatalError checks if the error matches known fatal AI provider error
// patterns such as billing, quota, or rate-limit issues.
func IsFatalError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	for _, pattern := range fatalAIPatterns {
		if strings.Contains(errStr, pattern) {
			return true
		}
	}
	return false
}
