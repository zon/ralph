package cmd

import (
	"bytes"
	"fmt"
	"os"
	"text/template"

	"gopkg.in/yaml.v3"
)

type OverviewComponent struct {
	Name    string `yaml:"name"`
	Path    string `yaml:"path"`
	Summary string `yaml:"summary"`
}

type Overview struct {
	Components []OverviewComponent `yaml:"components"`
}

func loadOverview(path string) (*Overview, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read overview file: %w", err)
	}

	var overview Overview
	if err := yaml.Unmarshal(data, &overview); err != nil {
		return nil, fmt.Errorf("failed to parse overview YAML: %w", err)
	}

	return &overview, nil
}

type overviewPromptData struct {
	OverviewPath string
}

var overviewPromptTemplate = template.Must(template.New("overview").Parse(`Explore the codebase and identify the major code components (packages, modules, or logical groupings).
For each component, provide its name, path relative to the repository root, and a one-sentence description of what it does.
Write the overview to {{.OverviewPath}} in YAML format with a top-level "components" list.
Each component entry should have "name", "path", and "summary" fields.
`))

func buildOverviewPrompt(overviewPath string) string {
	var buf bytes.Buffer
	data := overviewPromptData{OverviewPath: overviewPath}
	overviewPromptTemplate.Execute(&buf, data)
	return buf.String()
}

type componentPromptData struct {
	ConfigContent    string
	Project          string
	RalphProjectDoc  string
	ReviewName       string
	ComponentName    string
	ComponentPath    string
	ComponentSummary string
	SummaryPath      string
}

var componentPromptTemplate = template.Must(template.New("component").Parse(`You are a software architect reviewing source code. Does the code meet these standards?

## Review Content
{{.ConfigContent}}

## Component Context
Focus your review on the component named "{{.ComponentName}}" located at {{.ComponentPath}}.
This component: {{.ComponentSummary}}

## Instructions
Create or edit the ralph project at {{.Project}} with any issues found.
Set the project name field to "{{.ReviewName}}".

After completing your review, write a brief one-sentence summary of your recommendations to {{.SummaryPath}}.

{{.RalphProjectDoc}}
`))

func buildComponentPrompt(content, projectPath, projectDoc, reviewName string, component OverviewComponent, summaryPath string) string {
	var buf bytes.Buffer
	data := componentPromptData{
		ConfigContent:    content,
		Project:          projectPath,
		RalphProjectDoc:  projectDoc,
		ReviewName:       reviewName,
		ComponentName:    component.Name,
		ComponentPath:    component.Path,
		ComponentSummary: component.Summary,
		SummaryPath:      summaryPath,
	}
	componentPromptTemplate.Execute(&buf, data)
	return buf.String()
}
