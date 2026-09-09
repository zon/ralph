package ai

import (
	"bytes"
	_ "embed"
	"errors"
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

//go:embed project-fix-instructions.md
var projectFixInstructions string

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

//go:embed loop-instructions.md
var loopInstructions string

//go:embed loop-slug-instructions.md
var loopSlugInstructions string

type FixServicePromptData struct {
	Notes       []string
	ServiceName string
	ServiceCmd  string
	ServicePort int
	Error       string
}

type PRSummaryPromptData struct {
	ProjectDesc string
	BaseBranch  string
	CommitLog   string
	Usage       string
	AbsPath     string
}

type ChangelogPromptData struct {
	OutputFile string
}

// LoopPromptData carries the steps of the loop.
type LoopPromptData struct {
	Steps []string
}

// LoopSlugPromptData carries the steps of the loop and the output file path
// where the AI must write the proposed branch slug.
type LoopSlugPromptData struct {
	Steps      []string
	OutputFile string
}

type WriteProjectPromptData struct {
	InputPath         string
	InputType         string
	HasOrchestration  bool
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

func BuildPRSummaryPrompt(projectDesc, baseBranch, commitLog, usage, outputFile string) (string, error) {
	absPath, err := filepath.Abs(outputFile)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path: %w", err)
	}

	data := PRSummaryPromptData{
		ProjectDesc: projectDesc,
		BaseBranch:  baseBranch,
		CommitLog:   commitLog,
		Usage:       usage,
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

// BuildLoopPrompt renders the loop prompt embedding the given steps in order.
func BuildLoopPrompt(steps []string) (string, error) {
	return executeTemplate(loopInstructions, LoopPromptData{Steps: steps})
}

// BuildLoopSlugPrompt renders the loop slug prompt embedding the given steps in
// order and the absolute path of the file the AI must write the slug to.
func BuildLoopSlugPrompt(steps []string, outputPath string) (string, error) {
	absPath, err := filepath.Abs(outputPath)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path: %w", err)
	}

	data := LoopSlugPromptData{
		Steps:      steps,
		OutputFile: absPath,
	}
	return executeTemplate(loopSlugInstructions, data)
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

// DefaultItemDevelopmentInstructions returns the embedded default workflow
// steps for the development agent. They are used whenever the repository has
// no instructions file of its own.
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
		return ""
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

func resolveAgent(ctx *execcontext.Context) string {
	if a := ctx.Agent(); a != "" {
		return a
	}
	ralphConfig, err := config.LoadConfig()
	if err != nil {
		return ""
	}
	return ralphConfig.Agent
}

func RunAgent(ctx *execcontext.Context, oc opencode.OCClient, prompt string) error {
	if ctx.IsVerbose() {
		ctx.Output().Debug(prompt)
	}

	model := resolveModel(ctx)

	return oc.RunAgent(ctx.GoContext(), model, resolveVariant(ctx), resolveAgent(ctx), prompt)
}

// RunAgentPrimary runs the prompt with opencode's primary agent and is for
// prompts that produce supporting artifacts without touching repository code.
// It resolves the model and variant exactly like RunAgent but never passes the
// configured agent.
func RunAgentPrimary(ctx *execcontext.Context, oc opencode.OCClient, prompt string) error {
	if ctx.IsVerbose() {
		ctx.Output().Debug(prompt)
	}

	model := resolveModel(ctx)

	return oc.RunAgent(ctx.GoContext(), model, resolveVariant(ctx), "", prompt)
}

// RunAgentWithModel runs the prompt with an explicit model. It never passes
// the configured agent: the prompt runs with opencode's primary agent.
func RunAgentWithModel(ctx *execcontext.Context, oc opencode.OCClient, prompt string, model string) error {
	if ctx.IsVerbose() {
		ctx.Output().Debug(prompt)
	}

	return oc.RunAgent(ctx.GoContext(), model, resolveVariant(ctx), "", prompt)
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

// maxDeliverableAttempts is how many times a prompt that must leave a usable
// deliverable is run before the command reports an error naming the limit.
const maxDeliverableAttempts = 3

// runOpenCodeAndReadValidated runs opencode with the given prompt and returns
// the trimmed content of the output file the agent must write, provided the
// validate function accepts it. When opencode finishes without a usable
// deliverable — an output file that is missing or empty, or content that the
// validate function rejects — the same prompt is re-run until
// maxDeliverableAttempts attempts have been made. When every attempt is
// unusable the returned error names the attempt limit. An opencode execution
// failure is returned immediately and is never retried. A nil validate
// function accepts any content that reads from the output file.
func runOpenCodeAndReadValidated(ctx *execcontext.Context, oc opencode.OCClient, model, prompt, outputFile string, validate func(string) error) (string, error) {
	var stdoutWriter, stderrWriter io.Writer
	if ctx.IsVerbose() {
		stdoutWriter = os.Stdout
		stderrWriter = os.Stderr
	}

	var lastErr error
	for attempt := 1; attempt <= maxDeliverableAttempts; attempt++ {
		if err := oc.RunCommand(ctx.GoContext(), model, resolveVariant(ctx), "", prompt, stdoutWriter, stderrWriter); err != nil {
			return "", fmt.Errorf("opencode execution failed: %w", err)
		}

		content, err := readDeliverable(outputFile)
		if err == nil && validate != nil {
			err = validate(content)
		}
		if err == nil {
			return content, nil
		}
		lastErr = err
	}
	return "", fmt.Errorf("no usable deliverable after the %d-attempt limit: %w", maxDeliverableAttempts, lastErr)
}

// runOpenCodeAndReadDeliverable runs opencode with the given prompt and returns
// the trimmed content of the output file the agent must write. When opencode
// finishes without a usable deliverable — an output file that is missing or
// empty — the same prompt is re-run until maxDeliverableAttempts attempts have
// been made. When every attempt is unusable the returned error names the
// attempt limit. An opencode execution failure is returned immediately and is
// never retried.
func runOpenCodeAndReadDeliverable(ctx *execcontext.Context, oc opencode.OCClient, model, prompt, outputFile string) (string, error) {
	return runOpenCodeAndReadValidated(ctx, oc, model, prompt, outputFile, nil)
}

// readDeliverable reads and trims the content of the agent's output file. An
// output file that is missing or empty leaves no usable deliverable.
func readDeliverable(outputFile string) (string, error) {
	data, err := os.ReadFile(outputFile)
	if err != nil {
		return "", fmt.Errorf("failed to read output file: %w", err)
	}
	content := strings.TrimSpace(string(data))
	if content == "" {
		return "", errors.New("output file is empty")
	}
	return content, nil
}

func GeneratePRSummary(ctx *execcontext.Context, oc opencode.OCClient, projectDesc, baseBranch, commitLog, usage string) (summary string, err error) {
	f, err := createTempFile("pr-summary.md")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary PR summary file: %w", err)
	}
	f.Close()
	tmpFile := f.Name()
	defer os.Remove(tmpFile)

	prPrompt, err := BuildPRSummaryPrompt(projectDesc, baseBranch, commitLog, usage, tmpFile)
	if err != nil {
		return "", fmt.Errorf("failed to build PR summary prompt: %w", err)
	}

	if ctx.IsVerbose() {
		ctx.Output().Debug(prPrompt)
	}

	model := resolveModel(ctx)
	summary, err = runOpenCodeAndReadDeliverable(ctx, oc, model, prPrompt, tmpFile)
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
	_, err = runOpenCodeAndReadDeliverable(ctx, oc, model, changelogPrompt, tmpFile)
	if err != nil {
		return err
	}

	if err = os.Rename(tmpFile, "report.md"); err != nil {
		return fmt.Errorf("failed to rename changelog to report.md: %w", err)
	}

	return nil
}

// usableSlug reports whether the AI proposed a usable slug: non-empty after
// trimming, a single token of lowercase letters, digits, and hyphens only, not
// starting or ending with a hyphen, and without consecutive hyphens.
func usableSlug(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" || s[0] == '-' || s[len(s)-1] == '-' || strings.Contains(s, "--") {
		return false
	}
	for _, r := range s {
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '-' {
			return false
		}
	}
	return true
}

// ProposeLoopSlug asks the AI to read the loop steps and propose a short slug
// for the git branch that will run them. A proposed slug is usable only when
// the output file holds non-empty content that passes slug validation; when a
// run leaves no usable slug, the same prompt is re-run up to three attempts
// before an error naming the attempt limit is returned.
func ProposeLoopSlug(ctx *execcontext.Context, oc opencode.OCClient, steps []string) (slug string, err error) {
	f, err := createTempFile("loop-slug.md")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary loop slug file: %w", err)
	}
	f.Close()
	tmpFile := f.Name()
	defer os.Remove(tmpFile)

	slugPrompt, err := BuildLoopSlugPrompt(steps, tmpFile)
	if err != nil {
		return "", fmt.Errorf("failed to build loop slug prompt: %w", err)
	}

	if ctx.IsVerbose() {
		ctx.Output().Debug(slugPrompt)
	}

	model := resolveModel(ctx)
	content, err := runOpenCodeAndReadValidated(ctx, oc, model, slugPrompt, tmpFile, func(content string) error {
		if !usableSlug(content) {
			return errors.New("no usable slug proposed by the AI")
		}
		return nil
	})
	if err != nil {
		return "", err
	}

	return content, nil
}
