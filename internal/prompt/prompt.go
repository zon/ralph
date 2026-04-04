package prompt

import (
	"bytes"
	"fmt"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/context"
)

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

func BuildFixServicePrompt(ctx *context.Context, svc config.Service, svcErr error) (string, error) {
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

type PRSummaryPromptData struct {
	ProjectDesc   string
	ProjectStatus string
	BaseBranch    string
	CommitLog     string
	AbsPath       string
}

var prSummaryPromptTemplate = template.Must(template.New("prSummary").Parse(`Write a concise PR description (3-5 paragraphs max) for the changes made in this branch.

Project: {{.ProjectDesc}}
Status: {{.ProjectStatus}}

## Commit Log
{{.CommitLog}}

Review the git commits from {{.BaseBranch}}..HEAD to understand what was changed.
Use 'git log --format="%h: %B" {{.BaseBranch}}..HEAD' to see commit messages.
Use 'git diff {{.BaseBranch}}..HEAD' to see the full changes.

Summarize:
1. What was implemented/changed
2. Key technical decisions
3. Any notable considerations or future work

Be concise and focus on what matters for code review.

Write your summary to the file: {{.AbsPath}}
`))

func BuildPRSummaryPrompt(projectDesc, projectStatus, baseBranch, commitLog, outputFile string) (string, error) {
	absPath, err := filepath.Abs(outputFile)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path: %w", err)
	}

	var builder bytes.Buffer
	data := PRSummaryPromptData{
		ProjectDesc:   projectDesc,
		ProjectStatus: projectStatus,
		BaseBranch:    baseBranch,
		CommitLog:     commitLog,
		AbsPath:       absPath,
	}
	if err := prSummaryPromptTemplate.Execute(&builder, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}
	return builder.String(), nil
}

type ChangelogPromptData struct {
	OutputFile string
}

var changelogPromptTemplate = template.Must(template.New("changelog").Parse(`Write a concise changelog entry for the changes currently staged in git.

You are an AI agent that writes changelogs. Review the git diff (staged changes) and write a single changelog entry describing what changed.

Focus on:
• What was added, removed, or modified
• Why the changes were made (if apparent from the diff)
• Any notable implementation details

Write in the style of a conventional changelog entry, beginning with a verb in past tense (e.g., "Fixed", "Added", "Changed").

Write the changelog entry to the file: {{.OutputFile}}

Do not include any extra commentary, just the changelog entry.`))

func BuildChangelogPrompt(outputFile string) (string, error) {
	absPath, err := filepath.Abs(outputFile)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path: %w", err)
	}

	var b bytes.Buffer
	data := ChangelogPromptData{OutputFile: absPath}
	if err := changelogPromptTemplate.Execute(&b, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}
	return b.String(), nil
}

type ReviewPRBodyPromptData struct {
	ProjectName        string
	ProjectDescription string
	Requirements       []string
	AbsPath            string
}

var reviewPRBodyPromptTemplate = template.Must(template.New("reviewPRBody").Parse(`Write a concise PR description (2-4 paragraphs max) for this code review.

Review Name: {{.ProjectName}}
{{if .ProjectDescription}}
Description: {{.ProjectDescription}}
{{end}}

## Findings Summary
{{range .Requirements}}
{{.}}
{{end}}

Review the requirements above and write a summary that:
1. Lists the key findings from the review
2. Highlights any critical issues that need attention
3. Notes what's working well

Write your summary to the file: {{.AbsPath}}
`))

func BuildReviewPRBodyPrompt(projectName, projectDesc string, requirements []string, outputFile string) (string, error) {
	absPath, err := filepath.Abs(outputFile)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path: %w", err)
	}

	var builder bytes.Buffer
	data := ReviewPRBodyPromptData{
		ProjectName:        projectName,
		ProjectDescription: projectDesc,
		Requirements:       requirements,
		AbsPath:            absPath,
	}
	if err := reviewPRBodyPromptTemplate.Execute(&builder, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}
	return builder.String(), nil
}
