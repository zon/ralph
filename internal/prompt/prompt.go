package prompt

import (
	"bytes"
	"fmt"
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
