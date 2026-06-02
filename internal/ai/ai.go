package ai

import (
	"bytes"
	_ "embed"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/zon/ralph/internal/config"
	execcontext "github.com/zon/ralph/internal/context"
	"github.com/zon/ralph/internal/git"
	"github.com/zon/ralph/internal/logger"
	"github.com/zon/ralph/internal/opencode"
)

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
}

func BuildProjectFixPrompt(projectFile string, loadErr error) (string, error) {
	absPath, err := filepath.Abs(projectFile)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path: %w", err)
	}

	data := ProjectFixPromptData{ProjectFile: absPath, LoadError: loadErr.Error()}
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

// RunAgent executes an AI agent with the given prompt using OpenCode CLI
// OpenCode manages its own configuration for API keys and models
func RunAgent(ctx *execcontext.Context, oc opencode.OCClient, prompt string) error {
	if ctx.IsVerbose() {
		logger.Verbose(prompt)
	}

	model := resolveModel(ctx)

	return oc.RunAgent(ctx.GoContext(), model, resolveVariant(ctx), prompt)
}

// RunAgentWithModel executes an AI agent with an explicitly provided model,
// bypassing the context-based model resolution used by RunAgent.
func RunAgentWithModel(ctx *execcontext.Context, oc opencode.OCClient, prompt string, model string) error {
	if ctx.IsVerbose() {
		logger.Verbose(prompt)
	}

	return oc.RunAgent(ctx.GoContext(), model, resolveVariant(ctx), prompt)
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

// runOpenCodeAndReadResult runs opencode with the given prompt and reads the result from the output file
func runOpenCodeAndReadResult(ctx *execcontext.Context, oc opencode.OCClient, model, prompt, outputFile string) (string, error) {
	var stdoutWriter, stderrWriter io.Writer
	if ctx.IsVerbose() {
		stdoutWriter = os.Stdout
		stderrWriter = os.Stderr
	}

	if err := oc.RunCommand(ctx.GoContext(), model, resolveVariant(ctx), prompt, stdoutWriter, stderrWriter); err != nil {
		return "", fmt.Errorf("opencode execution failed: %w", err)
	}

	summaryBytes, err := os.ReadFile(outputFile)
	if err != nil {
		return "", fmt.Errorf("failed to read summary file: %w", err)
	}

	summary := strings.TrimSpace(string(summaryBytes))
	if summary == "" {
		return "", fmt.Errorf("summary file is empty")
	}

	return summary, nil
}

// GeneratePRSummary generates a pull request summary using AI
// It includes project description, status, commits, and diff
// This matches ralph.sh's approach: agent writes to a file, we read it back
func GeneratePRSummary(ctx *execcontext.Context, oc opencode.OCClient, projectDesc, projectStatus, baseBranch, commitLog string) (summary string, err error) {
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

	model := resolveModel(ctx)
	summary, err = runOpenCodeAndReadResult(ctx, oc, model, prPrompt, tmpFile)
	if err != nil {
		return "", err
	}

	return summary, nil
}

// GenerateChangelog prompts opencode to inspect the current git diff and write a
// descriptive changelog to report.md.
func GenerateChangelog(ctx *execcontext.Context, oc opencode.OCClient) (err error) {
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

	model := resolveModel(ctx)
	_, err = runOpenCodeAndReadResult(ctx, oc, model, changelogPrompt, tmpFile)
	if err != nil {
		return err
	}

	if err = os.Rename(tmpFile, "report.md"); err != nil {
		return fmt.Errorf("failed to rename changelog to report.md: %w", err)
	}

	return nil
}

// GenerateReviewPRBody generates a PR body for review findings using AI
// It reads the review project file and writes a concise summary of recommended changes
func GenerateReviewPRBody(ctx *execcontext.Context, oc opencode.OCClient, projectName, projectDesc string, requirementSummaries []string) (summary string, err error) {
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

	model := resolveModel(ctx)
	summary, err = runOpenCodeAndReadResult(ctx, oc, model, reviewPrompt, tmpFile)
	if err != nil {
		return "", err
	}

	return summary, nil
}

