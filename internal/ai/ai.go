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

//go:embed write-orchestration-instructions.md
var writeOrchestrationInstructions string

//go:embed write-project-instructions.md
var writeProjectInstructions string

//go:embed resolve-merge-conflicts-instructions.md
var resolveMergeConflictsInstructions string

//go:embed item-pick-instructions.md
var itemPickInstructions string

//go:embed item-develop-instructions.md
var itemDevelopInstructions string

//go:embed development-item-instructions.md
var itemDefaultInstructions string

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
	ProjectDesc string
	BaseBranch  string
	CommitLog   string
	AbsPath     string
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

type WriteProjectPromptData struct {
	InputPath        string
	InputType        string
	HasOrchestration bool
	OrchestrationPath string
}

type WriteOrchestrationPromptData struct {
	SpecPath string
}

type ResolveMergeConflictsPromptData struct {
	BaseBranch    string
	ProjectBranch string
}

// ItemPickPromptData carries the context for the picker agent: the full project
// file, the incomplete items each labelled with its index and key, and the
// recent commit log. The agent selects one item and reports its index.
type ItemPickPromptData struct {
	Notes          []string
	CommitLog      string
	ProjectContent string
	Items          string
}

// ItemDevelopPromptData carries the context for the development agent: the full
// project file, the selected item verbatim with its index and key, and the
// completion trailer the agent must use when the item is finished.
type ItemDevelopPromptData struct {
	Notes           []string
	CommitLog       string
	ProjectContent  string
	ItemIndex       int
	ItemKey         string
	ItemValue       string
	Trailer         string
	ProjectFilePath string
	Services        []config.Service
	Instructions    string
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

func BuildPRSummaryPrompt(projectDesc, baseBranch, commitLog, outputFile string) (string, error) {
	absPath, err := filepath.Abs(outputFile)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path: %w", err)
	}

	data := PRSummaryPromptData{
		ProjectDesc: projectDesc,
		BaseBranch:  baseBranch,
		CommitLog:   commitLog,
		AbsPath:     absPath,
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

func BuildWriteProjectPrompt(data WriteProjectPromptData) (string, error) {
	return executeTemplate(writeProjectInstructions, data)
}

func BuildWriteOrchestrationPrompt(data WriteOrchestrationPromptData) (string, error) {
	return executeTemplate(writeOrchestrationInstructions, data)
}

func BuildResolveMergeConflictsPrompt(baseBranch, projectBranch string) (string, error) {
	data := ResolveMergeConflictsPromptData{
		BaseBranch:    baseBranch,
		ProjectBranch: projectBranch,
	}
	return executeTemplate(resolveMergeConflictsInstructions, data)
}

// DefaultItemDevelopmentInstructions returns the embedded item-based default
// workflow steps for the development agent. The requirement-shaped config
// default describes a single requirement, so the item flow substitutes these
// item-shaped steps unless a custom instructions file overrides them.
func DefaultItemDevelopmentInstructions() string {
	return itemDefaultInstructions
}

// BuildItemPickPrompt renders the picker prompt from the project file content,
// the incomplete items rendered with their indices and keys, and the commit log.
func BuildItemPickPrompt(data ItemPickPromptData) (string, error) {
	tmplData := struct {
		Notes          []string
		CommitLog      string
		ProjectContent string
		Items          string
	}{
		Notes:          data.Notes,
		CommitLog:      data.CommitLog,
		ProjectContent: strings.TrimRight(data.ProjectContent, "\n"),
		Items:          strings.TrimRight(data.Items, "\n"),
	}
	return executeTemplate(itemPickInstructions, tmplData)
}

// BuildItemDevelopPrompt renders the development prompt carrying the full
// project file, the selected item verbatim with its index and key, and the
// completion trailer for the item. When no Instructions are supplied, the
// item-based default workflow steps are used.
func BuildItemDevelopPrompt(data ItemDevelopPromptData) (string, error) {
	if data.Instructions == "" {
		data.Instructions = DefaultItemDevelopmentInstructions()
	}
	tmplData := struct {
		Notes           []string
		CommitLog       string
		ProjectContent  string
		ItemIndex       int
		ItemKey         string
		ItemValue       string
		Trailer         string
		ProjectFilePath string
		Services        []config.Service
		Instructions    string
	}{
		Notes:           data.Notes,
		CommitLog:       data.CommitLog,
		ProjectContent:  strings.TrimRight(data.ProjectContent, "\n"),
		ItemIndex:       data.ItemIndex,
		ItemKey:         data.ItemKey,
		ItemValue:       strings.TrimRight(data.ItemValue, "\n"),
		Trailer:         data.Trailer,
		ProjectFilePath: data.ProjectFilePath,
		Services:        data.Services,
		Instructions:    data.Instructions,
	}
	return executeTemplate(itemDevelopInstructions, tmplData)
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

func RunAgent(ctx *execcontext.Context, oc opencode.OCClient, prompt string) error {
	if ctx.IsVerbose() {
		ctx.Output().Debug(prompt)
	}

	model := resolveModel(ctx)

	return oc.RunAgent(ctx.GoContext(), model, resolveVariant(ctx), prompt)
}

func RunAgentWithModel(ctx *execcontext.Context, oc opencode.OCClient, prompt string, model string) error {
	if ctx.IsVerbose() {
		ctx.Output().Debug(prompt)
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

func GeneratePRSummary(ctx *execcontext.Context, oc opencode.OCClient, projectDesc, baseBranch, commitLog string) (summary string, err error) {
	f, err := createTempFile("pr-summary.md")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary PR summary file: %w", err)
	}
	f.Close()
	tmpFile := f.Name()
	defer os.Remove(tmpFile)

	prPrompt, err := BuildPRSummaryPrompt(projectDesc, baseBranch, commitLog, tmpFile)
	if err != nil {
		return "", fmt.Errorf("failed to build PR summary prompt: %w", err)
	}

	if ctx.IsVerbose() {
		ctx.Output().Debug(prPrompt)
	}

	model := resolveModel(ctx)
	summary, err = runOpenCodeAndReadResult(ctx, oc, model, prPrompt, tmpFile)
	if err != nil {
		return "", err
	}

	return summary, nil
}

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
		ctx.Output().Debug(changelogPrompt)
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
		ctx.Output().Debug(reviewPrompt)
	}

	model := resolveModel(ctx)
	summary, err = runOpenCodeAndReadResult(ctx, oc, model, reviewPrompt, tmpFile)
	if err != nil {
		return "", err
	}

	return summary, nil
}

