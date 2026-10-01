package docker

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

type workflowStep struct {
	name      string
	uses      string
	run       string
	condition string
	with      map[string]interface{}
}

func TestScanWorkflow(t *testing.T) {
	workflowPath := filepath.Join("..", "..", ".github", "workflows", "scan.yaml")
	wf := readWorkflow(t, workflowPath)

	t.Run("named Scan", func(t *testing.T) {
		assert.Equal(t, "Scan", wf["name"])
	})

	t.Run("runs on pull requests", func(t *testing.T) {
		on, ok := wf["on"].(map[string]interface{})
		require.True(t, ok, "workflow should have an 'on' trigger")
		assert.Contains(t, on, "pull_request", "workflow should trigger on pull_request")
	})

	steps := workflowSteps(t, wf)
	trivy := trivySteps(steps)

	t.Run("builds the Containerfile", func(t *testing.T) {
		found := false
		for _, step := range steps {
			if strings.Contains(step.run, "docker build") && strings.Contains(step.run, "Containerfile") {
				found = true
			}
			if step.withString("file") == "Containerfile" {
				found = true
			}
		}
		assert.True(t, found, "workflow should build the Containerfile")
	})

	t.Run("scans the image at MEDIUM and above", func(t *testing.T) {
		require.NotEmpty(t, trivy, "workflow should run Trivy")

		found := false
		for _, step := range trivy {
			if hasSeverities(step.withString("severity"), "MEDIUM", "HIGH", "CRITICAL") {
				found = true
			}
		}
		assert.True(t, found, "a Trivy scan should cover MEDIUM, HIGH, and CRITICAL")
	})

	t.Run("writes a JSON report", func(t *testing.T) {
		found := false
		for _, step := range trivy {
			if step.withString("format") == "json" && strings.HasSuffix(step.withString("output"), ".json") {
				found = true
			}
		}
		assert.True(t, found, "Trivy should write a JSON report")
	})

	t.Run("writes a table report", func(t *testing.T) {
		found := false
		for _, step := range trivy {
			if step.withString("format") == "table" && strings.HasSuffix(step.withString("output"), ".txt") {
				found = true
			}
		}
		assert.True(t, found, "Trivy should write a table report")
	})

	t.Run("uploads the reports as an artifact", func(t *testing.T) {
		upload, found := findStep(steps, "actions/upload-artifact@")
		require.True(t, found, "workflow should upload the reports as an artifact")

		path := upload.withString("path")
		assert.Contains(t, path, ".json", "artifact should include the JSON report")
		assert.Contains(t, path, ".txt", "artifact should include the table report")
		assert.Equal(t, "always()", upload.condition,
			"artifact upload should still run when a CRITICAL finding fails the check")
	})

	t.Run("fails on CRITICAL findings", func(t *testing.T) {
		found := false
		for _, step := range trivy {
			if hasSeverities(step.withString("severity"), "CRITICAL") && step.withString("exit-code") == "1" {
				found = true
			}
		}
		assert.True(t, found, "a Trivy scan should fail the check on CRITICAL findings")
	})
}

func readWorkflow(t *testing.T, path string) map[string]interface{} {
	t.Helper()

	data, err := os.ReadFile(path)
	require.NoError(t, err, "workflow file should exist at %s", path)

	var wf map[string]interface{}
	require.NoError(t, yaml.Unmarshal(data, &wf), "workflow should be valid YAML")
	return wf
}

func workflowSteps(t *testing.T, wf map[string]interface{}) []workflowStep {
	t.Helper()

	jobs, ok := wf["jobs"].(map[string]interface{})
	require.True(t, ok, "workflow should have jobs")

	var steps []workflowStep
	for _, jobValue := range jobs {
		job, ok := jobValue.(map[string]interface{})
		if !ok {
			continue
		}
		rawSteps, ok := job["steps"].([]interface{})
		if !ok {
			continue
		}
		for _, raw := range rawSteps {
			m, ok := raw.(map[string]interface{})
			if !ok {
				continue
			}
			step := workflowStep{}
			step.name, _ = m["name"].(string)
			step.uses, _ = m["uses"].(string)
			step.run, _ = m["run"].(string)
			step.condition, _ = m["if"].(string)
			step.with, _ = m["with"].(map[string]interface{})
			steps = append(steps, step)
		}
	}
	return steps
}

func trivySteps(steps []workflowStep) []workflowStep {
	var trivy []workflowStep
	for _, step := range steps {
		if strings.HasPrefix(step.uses, "aquasecurity/trivy-action@") {
			trivy = append(trivy, step)
		}
	}
	return trivy
}

func findStep(steps []workflowStep, usesPrefix string) (workflowStep, bool) {
	for _, step := range steps {
		if strings.HasPrefix(step.uses, usesPrefix) {
			return step, true
		}
	}
	return workflowStep{}, false
}

func (s workflowStep) withString(key string) string {
	if s.with == nil {
		return ""
	}
	value, _ := s.with[key].(string)
	return value
}

func hasSeverities(severity string, wanted ...string) bool {
	if severity == "" {
		return false
	}

	severities := strings.Split(severity, ",")
	for _, want := range wanted {
		found := false
		for _, have := range severities {
			if strings.EqualFold(strings.TrimSpace(have), want) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}
