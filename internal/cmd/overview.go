package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"text/template"

	"github.com/zon/ralph/internal/ai"
	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/git"
	"github.com/zon/ralph/internal/logger"
	"gopkg.in/yaml.v3"

	_ "embed"
)

//go:embed overview-instructions.md
var overviewInstructions string

//go:embed component-review-instructions.md
var componentReviewInstructions string

type OverviewComponent struct {
	Name    string `json:"name" yaml:"name"`
	Path    string `json:"path" yaml:"path"`
	Summary string `json:"summary" yaml:"summary"`
}

type Overview struct {
	Modules []OverviewComponent `json:"modules" yaml:"modules"`
	Apps    []OverviewComponent `json:"apps" yaml:"apps"`
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

type OverviewCmd struct {
	Model   string `help:"Override the AI model from config" name:"model" optional:""`
	Verbose bool   `help:"Enable verbose logging" default:"false"`
}

func (o *OverviewCmd) Run() error {
	if o.Verbose {
		logger.SetVerbose(true)
	}

	if err := os.MkdirAll("projects", 0755); err != nil {
		return fmt.Errorf("failed to create projects directory: %w", err)
	}

	overviewPath, err := git.TmpPath("overview.json")
	if err != nil {
		return fmt.Errorf("failed to resolve overview path: %w", err)
	}

	ralphConfig, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	model := o.resolveModel(ralphConfig)

	ctx := createExecutionContext()
	ctx.SetVerbose(o.Verbose)
	ctx.SetModel(model)
	ctx.SetLocal(true)

	prompt := buildOverviewPrompt(overviewPath)

	if o.Verbose {
		logger.Verbose(prompt)
	}

	logger.Verbose("Running overview step: generating code overview...")
	if err := ai.RunAgent(ctx, prompt); err != nil {
		os.Remove(overviewPath)
		return fmt.Errorf("overview step failed: %w", err)
	}

	overview, err := loadOverview(overviewPath)
	if err != nil {
		os.Remove(overviewPath)
		return fmt.Errorf("failed to load overview: %w", err)
	}

	os.Remove(overviewPath)

	output, err := yaml.Marshal(overview)
	if err != nil {
		return fmt.Errorf("failed to marshal overview to YAML: %w", err)
	}

	fmt.Print(string(output))

	return nil
}

func (o *OverviewCmd) resolveModel(ralphConfig *config.RalphConfig) string {
	if o.Model != "" {
		return o.Model
	}
	if ralphConfig.Review.Model != "" {
		return ralphConfig.Review.Model
	}
	return ralphConfig.Model
}
