package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"text/template"

	_ "embed"
)

//go:embed overview-instructions.md
var overviewInstructions string

//go:embed component-review-instructions.md
var componentReviewInstructions string

type OverviewComponent struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	Summary string `json:"summary"`
}

type Overview struct {
	Modules []OverviewComponent `json:"modules"`
	Apps    []OverviewComponent `json:"apps"`
}

func (o *Overview) AllComponents() []OverviewComponent {
	result := make([]OverviewComponent, 0, len(o.Modules)+len(o.Apps))
	result = append(result, o.Modules...)
	result = append(result, o.Apps...)
	return result
}

func loadOverview(path string) (*Overview, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read overview file: %w", err)
	}

	var overview Overview
	if err := json.Unmarshal(data, &overview); err != nil {
		return nil, fmt.Errorf("failed to parse overview JSON: %w", err)
	}

	return &overview, nil
}

type overviewPromptData struct {
	OverviewPath string
}

var overviewPromptTemplate = template.Must(template.New("overview").Parse(overviewInstructions))

func buildOverviewPrompt(overviewPath string) string {
	var buf bytes.Buffer
	data := overviewPromptData{OverviewPath: overviewPath}
	overviewPromptTemplate.Execute(&buf, data)
	return buf.String()
}

type componentPromptData struct {
	ConfigContent    string
	RalphProjectDoc  string
	ComponentName    string
	ComponentPath    string
	ComponentSummary string
	SummaryPath      string
}

var componentPromptTemplate = template.Must(template.New("component").Parse(componentReviewInstructions))

func buildComponentPrompt(content, projectDoc string, component OverviewComponent, summaryPath string) string {
	var buf bytes.Buffer
	data := componentPromptData{
		ConfigContent:    content,
		RalphProjectDoc:  projectDoc,
		ComponentName:    component.Name,
		ComponentPath:    component.Path,
		ComponentSummary: component.Summary,
		SummaryPath:      summaryPath,
	}
	componentPromptTemplate.Execute(&buf, data)
	return buf.String()
}
