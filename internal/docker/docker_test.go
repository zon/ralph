package docker

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDockerfileExists(t *testing.T) {
	projectRoot := filepath.Join("..", "..")
	containerfilePath := filepath.Join(projectRoot, "Containerfile")

	_, err := os.Stat(containerfilePath)
	require.NoError(t, err, "Containerfile should exist at %s", containerfilePath)
}

func TestDockerfileContainsRequiredComponents(t *testing.T) {
	projectRoot := filepath.Join("..", "..")
	containerfilePath := filepath.Join(projectRoot, "Containerfile")

	content, err := os.ReadFile(containerfilePath)
	require.NoError(t, err, "Should be able to read Containerfile")

	dockerfile := string(content)

	requiredComponents := []struct {
		name        string
		searchTerms []string
	}{
		{
			name:        "Ralph binary",
			searchTerms: []string{"ralph"},
		},
		{
			name:        "argo CLI",
			searchTerms: []string{"argo"},
		},
		{
			name:        "OpenCode",
			searchTerms: []string{"opencode"},
		},
		{
			name:        "Git",
			searchTerms: []string{"git"},
		},
		{
			name:        "GitHub CLI",
			searchTerms: []string{"cli.github.com", "gh"},
		},
		{
			name:        "shell",
			searchTerms: []string{"/bin/sh"},
		},
		{
			name:        "CA certificates",
			searchTerms: []string{"ca-certificates"},
		},
	}

	for _, component := range requiredComponents {
		t.Run(component.name, func(t *testing.T) {
			found := false
			for _, term := range component.searchTerms {
				if strings.Contains(strings.ToLower(dockerfile), strings.ToLower(term)) {
					found = true
					break
				}
			}
			assert.True(t, found, "Containerfile should contain required component: %s", component.name)
		})
	}
}

func TestDockerfileUsesMultiStageBuilds(t *testing.T) {
	projectRoot := filepath.Join("..", "..")
	containerfilePath := filepath.Join(projectRoot, "Containerfile")

	content, err := os.ReadFile(containerfilePath)
	require.NoError(t, err, "Should be able to read Containerfile")

	dockerfile := string(content)

	assert.True(t, strings.Contains(dockerfile, "AS builder") || strings.Contains(dockerfile, "AS build"),
		"Containerfile should use multi-stage builds")
	assert.Contains(t, dockerfile, "COPY --from=", "Containerfile should copy artifacts from build stage")
}

func TestDockerfileUsesUbuntuNobleRuntimeBase(t *testing.T) {
	projectRoot := filepath.Join("..", "..")
	containerfilePath := filepath.Join(projectRoot, "Containerfile")

	content, err := os.ReadFile(containerfilePath)
	require.NoError(t, err, "Should be able to read Containerfile")

	runtime := runtimeStage(t, string(content))

	pattern := regexp.MustCompile(`(?m)^FROM\s+\S*ubuntu:24\.04(?:\s|$)`)
	assert.Regexp(t, pattern, runtime,
		"Runtime stage should build on ubuntu:24.04")
	assert.NotContains(t, strings.ToLower(runtime), "playwright",
		"Runtime stage should not build on the Playwright image")
}

func runtimeStage(t *testing.T, dockerfile string) string {
	t.Helper()

	fromPattern := regexp.MustCompile(`(?m)^FROM\s+`)
	matches := fromPattern.FindAllStringIndex(dockerfile, -1)
	require.NotEmpty(t, matches, "Containerfile should declare a stage")

	return dockerfile[matches[len(matches)-1][0]:]
}

func containerfileStage(t *testing.T, dockerfile, name string) string {
	t.Helper()

	header := regexp.MustCompile(`(?mi)^FROM\s+.*\s+AS\s+` + regexp.QuoteMeta(name) + `\s*$`)
	location := header.FindStringIndex(dockerfile)
	require.NotNil(t, location, "Containerfile should declare stage %q", name)

	end := len(dockerfile)
	rest := dockerfile[location[1]:]
	if next := regexp.MustCompile(`(?m)^FROM\s+`).FindStringIndex(rest); next != nil {
		end = location[1] + next[0]
	}
	return dockerfile[location[0]:end]
}

func TestDockerfileAppliesUbuntuSecurityUpdates(t *testing.T) {
	projectRoot := filepath.Join("..", "..")
	containerfilePath := filepath.Join(projectRoot, "Containerfile")

	content, err := os.ReadFile(containerfilePath)
	require.NoError(t, err, "Should be able to read Containerfile")

	runtime := runtimeStage(t, string(content))

	upgradePattern := regexp.MustCompile(`apt-get\s+(?:dist-)?upgrade\s+-y`)
	assert.Regexp(t, upgradePattern, runtime,
		"Runtime stage should apply Ubuntu security updates with apt-get upgrade")
}

func TestDockerfileUpgradesArgo(t *testing.T) {
	projectRoot := filepath.Join("..", "..")
	containerfilePath := filepath.Join(projectRoot, "Containerfile")

	content, err := os.ReadFile(containerfilePath)
	require.NoError(t, err, "Should be able to read Containerfile")

	version := containerfileVersion(t, string(content), "ARGO_VERSION")
	minimum := [3]int{4, 1, 4}
	assert.GreaterOrEqual(t, compareVersions(version, minimum), 0,
		"Containerfile should pin ARGO_VERSION at v%d.%d.%d or later, got v%d.%d.%d",
		minimum[0], minimum[1], minimum[2], version[0], version[1], version[2])
}

// TestDockerfileBuildsArgoWithPatchedGoJose covers the argo CLI, whose latest
// release still ships go-jose with a known HIGH vulnerability. The Containerfile
// builds it from its release tag in its own stage, raises go-jose, and keeps
// the Go module cache out of the runtime image.
func TestDockerfileBuildsArgoWithPatchedGoJose(t *testing.T) {
	projectRoot := filepath.Join("..", "..")
	containerfilePath := filepath.Join(projectRoot, "Containerfile")

	content, err := os.ReadFile(containerfilePath)
	require.NoError(t, err, "Should be able to read Containerfile")

	dockerfile := string(content)
	argo := containerfileStage(t, dockerfile, "argo")
	runtime := runtimeStage(t, dockerfile)

	basePattern := regexp.MustCompile(`(?m)^FROM\s+\S*golang:\d+\.\d+\.\d+-\S+\s+AS\s+argo\s*$`)
	assert.Regexp(t, basePattern, argo,
		"argo stage should build on a golang image pinned to a patch version")

	assert.Contains(t, argo, "git clone", "argo stage should build from its release tag")
	assert.Contains(t, argo, "--branch ${ARGO_VERSION}")
	assert.Contains(t, argo, "https://github.com/argoproj/argo-workflows.git")
	assert.Contains(t, argo, "go get github.com/go-jose/go-jose/v4@${ARGO_JOSE_VERSION}")
	assert.Contains(t, argo, "-o /out/argo ./cmd/argo")

	assert.Contains(t, runtime, "COPY --from=argo /out/argo /usr/local/bin/argo",
		"Runtime stage should copy the argo CLI from its build stage")
	assert.NotContains(t, runtime, "go install",
		"runtime stage should not build Go tools, which would leave the module cache in the image")

	version := containerfileVersion(t, argo, "ARGO_JOSE_VERSION")
	minimum := [3]int{4, 1, 6}
	assert.GreaterOrEqual(t, compareVersions(version, minimum), 0,
		"Containerfile should build with ARGO_JOSE_VERSION v%d.%d.%d or later, got v%d.%d.%d",
		minimum[0], minimum[1], minimum[2], version[0], version[1], version[2])

	t.Run("no air, templ, or helm", func(t *testing.T) {
		for _, term := range []string{
			"AIR_VERSION",
			"TEMPL_VERSION",
			"HELM_VERSION",
			"github.com/air-verse/air",
			"github.com/a-h/templ",
			"helm.sh/helm",
			"AS go-tools",
		} {
			assert.NotContains(t, dockerfile, term,
				"Containerfile should no longer build or name %s", term)
		}
	})
}

func TestDockerfileInstallsPinnedOpenCodeInOwnStage(t *testing.T) {
	projectRoot := filepath.Join("..", "..")
	containerfilePath := filepath.Join(projectRoot, "Containerfile")

	content, err := os.ReadFile(containerfilePath)
	require.NoError(t, err, "Should be able to read Containerfile")

	dockerfile := string(content)
	opencode := containerfileStage(t, dockerfile, "opencode")

	basePattern := regexp.MustCompile(`(?m)^FROM\s+(?:--platform=\S+\s+)?\S*bun:\d+\.\d+\.\d+`)
	assert.Regexp(t, basePattern, opencode,
		"OpenCode should build in its own stage on a pinned Bun image")

	// Bun never runs a dependency's lifecycle scripts, so the install must
	// trust the package or its postinstall leaves the bin/opencode.exe stub
	// that fails to exec.
	assert.Contains(t, opencode, "bun install -g --trust @opencode/cli@${OPENCODE_VERSION}",
		"OpenCode stage should install the trusted, pinned OpenCode CLI")

	version := containerfileVersion(t, opencode, "OPENCODE_VERSION")
	minimum := [3]int{2, 0, 29}
	assert.GreaterOrEqual(t, compareVersions(version, minimum), 0,
		"Containerfile should pin OPENCODE_VERSION at v%d.%d.%d or later, got v%d.%d.%d",
		minimum[0], minimum[1], minimum[2], version[0], version[1], version[2])

	assert.Contains(t, opencode, "/out/opencode",
		"OpenCode stage should keep only the native binary as its output")
}

func TestDockerfileRuntimeReceivesOnlyOpenCodeBinary(t *testing.T) {
	projectRoot := filepath.Join("..", "..")
	containerfilePath := filepath.Join(projectRoot, "Containerfile")

	content, err := os.ReadFile(containerfilePath)
	require.NoError(t, err, "Should be able to read Containerfile")

	runtime := runtimeStage(t, string(content))

	assert.Contains(t, runtime, "COPY --from=opencode /out/opencode /usr/local/bin/opencode",
		"Runtime stage should copy only the native OpenCode binary from its build stage")

	assert.NotRegexp(t, regexp.MustCompile(`(?i)\bbun\b`), runtime,
		"Runtime stage should not carry a Bun runtime")

	lower := strings.ToLower(runtime)
	for _, forbidden := range []string{
		"bun install",
		"@opencode/cli",
		"node_modules",
		"cli-linux",
		"musl",
		"baseline",
	} {
		assert.NotContains(t, lower, forbidden,
			"Runtime stage should not carry the OpenCode build stage's Bun cache or platform packages: %s", forbidden)
	}
}

type openCodePermissionRule struct {
	Action   string `json:"action"`
	Resource string `json:"resource"`
	Effect   string `json:"effect"`
}

type openCodeContainerConfig struct {
	Schema       string                   `json:"$schema"`
	Update       string                   `json:"update"`
	Permissions  []openCodePermissionRule `json:"permissions"`
	Experimental map[string]any           `json:"experimental"`
}

func TestContainerOpenCodeConfigServesSandbox(t *testing.T) {
	projectRoot := filepath.Join("..", "..")
	configPath := filepath.Join(projectRoot, "container", "opencode.json")

	content, err := os.ReadFile(configPath)
	require.NoError(t, err, "Should be able to read container/opencode.json")

	var cfg openCodeContainerConfig
	require.NoError(t, json.Unmarshal(content, &cfg), "container/opencode.json should be valid JSON")

	assert.Equal(t, "https://opencode.ai/config.json", cfg.Schema,
		"config should declare the published schema")

	assert.Equal(t, "disable", cfg.Update,
		"config should disable updates so the pinned OpenCode version is the one runs use")

	allowsEverything := false
	for _, rule := range cfg.Permissions {
		assert.NotEqual(t, "ask", rule.Effect,
			"an unattended container has no client to answer %q on %q", rule.Effect, rule.Resource)
		if rule.Action == "*" && rule.Resource == "*" && rule.Effect == "allow" {
			allowsEverything = true
		}
	}
	assert.True(t, allowsEverything,
		"config should allow every tool action because the container is the sandbox")

	_, ok := cfg.Experimental["subagent_depth"]
	assert.True(t, ok, "config should keep the configured subagent depth")
}

func TestContainerfileConfiguresOpenCode(t *testing.T) {
	projectRoot := filepath.Join("..", "..")
	containerfilePath := filepath.Join(projectRoot, "Containerfile")

	content, err := os.ReadFile(containerfilePath)
	require.NoError(t, err, "Should be able to read Containerfile")

	assert.Contains(t, string(content), "COPY container/opencode.json /root/.config/opencode/opencode.json",
		"Containerfile should install the container OpenCode configuration as the global config")
}

func TestContainerfileKeepsWorkspaceAndDefaultShell(t *testing.T) {
	projectRoot := filepath.Join("..", "..")
	containerfilePath := filepath.Join(projectRoot, "Containerfile")

	content, err := os.ReadFile(containerfilePath)
	require.NoError(t, err, "Should be able to read Containerfile")

	runtime := runtimeStage(t, string(content))

	assert.Contains(t, runtime, "WORKDIR /workspace",
		"Runtime stage should keep the workspace directory")
	assert.Contains(t, runtime, `CMD ["/bin/sh"]`,
		"Runtime stage should keep the default shell command")
}

func TestRalphBinaryHasPatchedGoDependencies(t *testing.T) {
	projectRoot := filepath.Join("..", "..")
	goModPath := filepath.Join(projectRoot, "go.mod")

	content, err := os.ReadFile(goModPath)
	require.NoError(t, err, "Should be able to read go.mod")

	goMod := string(content)

	cases := []struct {
		module  string
		minimum [3]int
	}{
		{"golang.org/x/net", [3]int{0, 60, 0}},
		{"golang.org/x/text", [3]int{0, 39, 0}},
	}

	for _, tc := range cases {
		t.Run(tc.module, func(t *testing.T) {
			version := goModVersion(t, goMod, tc.module)
			assert.GreaterOrEqual(t, compareVersions(version, tc.minimum), 0,
				"go.mod should require %s v%d.%d.%d or later, got v%d.%d.%d",
				tc.module, tc.minimum[0], tc.minimum[1], tc.minimum[2],
				version[0], version[1], version[2])
		})
	}
}

func goModVersion(t *testing.T, goMod, module string) [3]int {
	t.Helper()

	pattern := regexp.MustCompile(`(?m)^\s*` + regexp.QuoteMeta(module) + `\s+v(\d+)\.(\d+)\.(\d+)`)
	match := pattern.FindStringSubmatch(goMod)
	require.NotNil(t, match, "go.mod should require %s", module)

	var version [3]int
	for i := range version {
		part, err := strconv.Atoi(match[i+1])
		require.NoError(t, err, "%s should have numeric version parts", module)
		version[i] = part
	}
	return version
}

func containerfileVersion(t *testing.T, dockerfile, envVar string) [3]int {
	t.Helper()

	pattern := regexp.MustCompile(`ENV\s+` + envVar + `=v?(\d+)\.(\d+)\.(\d+)`)
	match := pattern.FindStringSubmatch(dockerfile)
	require.NotNil(t, match, "Containerfile should pin %s", envVar)

	var version [3]int
	for i := range version {
		part, err := strconv.Atoi(match[i+1])
		require.NoError(t, err, "%s should have numeric version parts", envVar)
		version[i] = part
	}
	return version
}

func compareVersions(a, b [3]int) int {
	for i := range a {
		if a[i] != b[i] {
			if a[i] > b[i] {
				return 1
			}
			return -1
		}
	}
	return 0
}

func TestPushScriptExists(t *testing.T) {
	projectRoot := filepath.Join("..", "..")
	scriptPath := filepath.Join(projectRoot, "scripts", "push-image.sh")

	_, err := os.Stat(scriptPath)
	require.NoError(t, err, "Push script should exist at %s", scriptPath)
}

func TestPushScriptIsExecutable(t *testing.T) {
	projectRoot := filepath.Join("..", "..")
	scriptPath := filepath.Join(projectRoot, "scripts", "push-image.sh")

	info, err := os.Stat(scriptPath)
	require.NoError(t, err, "Should be able to stat push script")

	mode := info.Mode()
	assert.NotZero(t, mode&0111, "Push script should be executable")
}

func TestPushScriptContentsValid(t *testing.T) {
	projectRoot := filepath.Join("..", "..")
	scriptPath := filepath.Join(projectRoot, "scripts", "push-image.sh")

	content, err := os.ReadFile(scriptPath)
	require.NoError(t, err, "Should be able to read push script")

	script := string(content)

	requiredElements := []struct {
		name    string
		pattern string
	}{
		{"Shebang", "#!/bin/bash"},
		{"Set error handling", "set -e"},
		{"Repository variable", "REPOSITORY"},
		{"Tag variable", "TAG"},
		{"Podman build command", "podman build"},
		{"Podman push command", "podman push"},
		{"Containerfile reference", "Containerfile"},
	}

	for _, element := range requiredElements {
		t.Run(element.name, func(t *testing.T) {
			assert.Contains(t, script, element.pattern, "Push script should contain: %s", element.name)
		})
	}
}

func TestPushScriptUsesEnvironmentVariables(t *testing.T) {
	projectRoot := filepath.Join("..", "..")
	scriptPath := filepath.Join(projectRoot, "scripts", "push-image.sh")

	content, err := os.ReadFile(scriptPath)
	require.NoError(t, err, "Should be able to read push script")

	script := string(content)

	envVarPatterns := []string{
		"RALPH_IMAGE_REPOSITORY",
		"RALPH_IMAGE_TAG",
	}

	for _, pattern := range envVarPatterns {
		t.Run(pattern, func(t *testing.T) {
			assert.Contains(t, script, pattern, "Push script should support %s environment variable", pattern)
		})
	}

	assert.Contains(t, script, "ghcr.io/zon/ralph", "Push script should have default repository")
	assert.Contains(t, script, "latest", "Push script should have default tag")
}

func TestDefaultImageMatchesWorkflowDefault(t *testing.T) {
	projectRoot := filepath.Join("..", "..")
	scriptPath := filepath.Join(projectRoot, "scripts", "push-image.sh")

	content, err := os.ReadFile(scriptPath)
	require.NoError(t, err, "Should be able to read push script")

	script := string(content)

	expectedRepo := "ghcr.io/zon/ralph"
	assert.Contains(t, script, expectedRepo, "Script default repository should be %s", expectedRepo)
}
