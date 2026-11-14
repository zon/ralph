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
	containerfilePath := filepath.Join(projectRoot, "containers", "ralph", "Containerfile")

	_, err := os.Stat(containerfilePath)
	require.NoError(t, err, "Containerfile should exist at %s", containerfilePath)
}

func TestDockerfileRuntimeCarriesRequiredComponents(t *testing.T) {
	projectRoot := filepath.Join("..", "..")
	containerfilePath := filepath.Join(projectRoot, "containers", "ralph", "Containerfile")

	content, err := os.ReadFile(containerfilePath)
	require.NoError(t, err, "Should be able to read Containerfile")

	runtime := runtimeStage(t, string(content))

	requiredComponents := []struct {
		name string
		line string
	}{
		{"Ralph binary", "COPY --from=builder /build/ralph /usr/local/bin/ralph"},
		{"argo CLI", "COPY --from=argo /out/argo /usr/local/bin/argo"},
		{"OpenCode", "COPY --from=opencode /out/opencode /usr/local/bin/opencode"},
		{"Git", "COPY --from=git /usr/bin/git /usr/bin/git-shell /usr/bin/scalar /usr/bin/"},
		{"GitHub CLI", "COPY --from=gh /out/gh /usr/local/bin/gh"},
		{"shell", `CMD ["/bin/sh"]`},
		{"CA certificates", "ca-certificates"},
	}

	for _, component := range requiredComponents {
		t.Run(component.name, func(t *testing.T) {
			assert.Contains(t, runtime, component.line,
				"Runtime stage should carry required component: %s", component.name)
		})
	}
}

func TestDockerfileUsesMultiStageBuilds(t *testing.T) {
	projectRoot := filepath.Join("..", "..")
	containerfilePath := filepath.Join(projectRoot, "containers", "ralph", "Containerfile")

	content, err := os.ReadFile(containerfilePath)
	require.NoError(t, err, "Should be able to read Containerfile")

	dockerfile := string(content)

	assert.True(t, strings.Contains(dockerfile, "AS builder") || strings.Contains(dockerfile, "AS build"),
		"Containerfile should use multi-stage builds")
	assert.Contains(t, dockerfile, "COPY --from=", "Containerfile should copy artifacts from build stage")
}

func TestDockerfileUsesUbuntuResoluteRuntimeBase(t *testing.T) {
	projectRoot := filepath.Join("..", "..")
	containerfilePath := filepath.Join(projectRoot, "containers", "ralph", "Containerfile")

	content, err := os.ReadFile(containerfilePath)
	require.NoError(t, err, "Should be able to read Containerfile")

	runtime := runtimeStage(t, string(content))

	pattern := regexp.MustCompile(`(?m)^FROM\s+\S*ubuntu:26\.04(?:\s|$)`)
	assert.Regexp(t, pattern, runtime,
		"Runtime stage should build on ubuntu:26.04")
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
	containerfilePath := filepath.Join(projectRoot, "containers", "ralph", "Containerfile")

	content, err := os.ReadFile(containerfilePath)
	require.NoError(t, err, "Should be able to read Containerfile")

	runtime := runtimeStage(t, string(content))

	upgradePattern := regexp.MustCompile(`apt-get\s+(?:dist-)?upgrade\s+-y`)
	assert.Regexp(t, upgradePattern, runtime,
		"Runtime stage should apply Ubuntu security updates with apt-get upgrade")
}

func TestDockerfileUpgradesArgo(t *testing.T) {
	projectRoot := filepath.Join("..", "..")
	containerfilePath := filepath.Join(projectRoot, "containers", "ralph", "Containerfile")

	content, err := os.ReadFile(containerfilePath)
	require.NoError(t, err, "Should be able to read Containerfile")

	version := containerfileVersion(t, string(content), "ARGO_VERSION")
	minimum := [3]int{4, 1, 4}
	assert.GreaterOrEqual(t, compareVersions(version, minimum), 0,
		"Containerfile should pin ARGO_VERSION at v%d.%d.%d or later, got v%d.%d.%d",
		minimum[0], minimum[1], minimum[2], version[0], version[1], version[2])
}

// TestDockerfileBuildsArgoWithPatchedDependencies covers the argo CLI, whose
// latest release still ships go-jose and x/net with known HIGH vulnerabilities.
// The Containerfile builds it from its release tag in its own stage, raises
// those dependencies, and keeps the Go module cache out of the runtime image.
func TestDockerfileBuildsArgoWithPatchedDependencies(t *testing.T) {
	projectRoot := filepath.Join("..", "..")
	containerfilePath := filepath.Join(projectRoot, "containers", "ralph", "Containerfile")

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
	assert.Contains(t, argo, "go get github.com/go-jose/go-jose/v4@${ARGO_JOSE_VERSION} golang.org/x/net@${ARGO_NET_VERSION}")
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

	netVersion := containerfileVersion(t, argo, "ARGO_NET_VERSION")
	minimumNet := [3]int{0, 62, 0}
	assert.GreaterOrEqual(t, compareVersions(netVersion, minimumNet), 0,
		"Containerfile should build argo with ARGO_NET_VERSION v%d.%d.%d or later, got v%d.%d.%d",
		minimumNet[0], minimumNet[1], minimumNet[2], netVersion[0], netVersion[1], netVersion[2])

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

// TestDockerfileBuildsGhWithPatchedXNet covers the GitHub CLI, whose latest
// release still ships x/net with a known HIGH vulnerability. The Containerfile
// builds it from its release tag in its own stage, raises x/net, and keeps the
// Go module cache out of the runtime image.
func TestDockerfileBuildsGhWithPatchedXNet(t *testing.T) {
	projectRoot := filepath.Join("..", "..")
	containerfilePath := filepath.Join(projectRoot, "containers", "ralph", "Containerfile")

	content, err := os.ReadFile(containerfilePath)
	require.NoError(t, err, "Should be able to read Containerfile")

	dockerfile := string(content)
	gh := containerfileStage(t, dockerfile, "gh")
	runtime := runtimeStage(t, dockerfile)

	basePattern := regexp.MustCompile(`(?m)^FROM\s+\S*golang:\d+\.\d+\.\d+-\S+\s+AS\s+gh\s*$`)
	assert.Regexp(t, basePattern, gh,
		"gh stage should build on a golang image pinned to a patch version")

	assert.Contains(t, gh, "git clone", "gh stage should build from its release tag")
	assert.Contains(t, gh, "--branch v${GH_VERSION}")
	assert.Contains(t, gh, "https://github.com/cli/cli.git")
	assert.Contains(t, gh, "go get golang.org/x/net@${GH_NET_VERSION}")
	assert.Contains(t, gh, "-X github.com/cli/cli/v2/internal/build.Version=${GH_VERSION}")
	assert.Contains(t, gh, "-o /out/gh ./cmd/gh")

	assert.Contains(t, runtime, "COPY --from=gh /out/gh /usr/local/bin/gh",
		"Runtime stage should copy the GitHub CLI from its build stage")
	assert.NotContains(t, runtime, "apt-get install -y --no-install-recommends gh",
		"Runtime stage should not install the packaged GitHub CLI while it ships a vulnerable x/net")

	version := containerfileVersion(t, gh, "GH_VERSION")
	minimum := [3]int{2, 104, 0}
	assert.GreaterOrEqual(t, compareVersions(version, minimum), 0,
		"Containerfile should build with GH_VERSION v%d.%d.%d or later, got v%d.%d.%d",
		minimum[0], minimum[1], minimum[2], version[0], version[1], version[2])

	netVersion := containerfileVersion(t, gh, "GH_NET_VERSION")
	minimumNet := [3]int{0, 62, 0}
	assert.GreaterOrEqual(t, compareVersions(netVersion, minimumNet), 0,
		"Containerfile should build gh with GH_NET_VERSION v%d.%d.%d or later, got v%d.%d.%d",
		minimumNet[0], minimumNet[1], minimumNet[2], netVersion[0], netVersion[1], netVersion[2])
}

func TestDockerfileInstallsPinnedOpenCodeInOwnStage(t *testing.T) {
	projectRoot := filepath.Join("..", "..")
	containerfilePath := filepath.Join(projectRoot, "containers", "ralph", "Containerfile")

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

func TestDockerfileCopiesGitFromOwnStage(t *testing.T) {
	projectRoot := filepath.Join("..", "..")
	containerfilePath := filepath.Join(projectRoot, "containers", "ralph", "Containerfile")

	content, err := os.ReadFile(containerfilePath)
	require.NoError(t, err, "Should be able to read Containerfile")

	dockerfile := string(content)
	git := containerfileStage(t, dockerfile, "git")
	runtime := runtimeStage(t, dockerfile)

	assert.Contains(t, git, "apt-get install -y --no-install-recommends git",
		"git stage should install the git package")

	assert.Contains(t, runtime, "COPY --from=git /usr/bin/git /usr/bin/git-shell /usr/bin/scalar /usr/bin/",
		"Runtime stage should copy only the compiled git commands")
	assert.Contains(t, runtime, "COPY --from=git /usr/lib/git-core /usr/lib/git-core")
	assert.Contains(t, runtime, "COPY --from=git /usr/share/git-core /usr/share/git-core")
	assert.NotContains(t, strings.ToLower(runtime), "git-man",
		"Runtime stage should not carry the git documentation package")

	for _, library := range []string{"libcurl3t64-gnutls", "libexpat1", "libpcre2-8-0"} {
		assert.Contains(t, runtime, library,
			"Runtime stage should install the shared library git links against: %s", library)
	}
}

var forbiddenRuntimeTools = []struct {
	name    string
	pattern *regexp.Regexp
}{
	{"Go", regexp.MustCompile(`(?i)\bgo\b|\bgofmt\b|\bgopath\b|\bgoroot\b|/usr/local/go|go\.dev|\bGO_VERSION\b|\bGOFLAGS\b`)},
	{"Bun", regexp.MustCompile(`(?i)\bbun(?:x)?\b`)},
	{"Node", regexp.MustCompile(`(?i)\bnode(?:\.?js)?\b|node_modules`)},
	{"npm", regexp.MustCompile(`(?i)\bnpm\b|\bnpx\b|npmjs`)},
	{"Rust", regexp.MustCompile(`(?i)\brustc?\b|\bcargo\b`)},
	{"Python", regexp.MustCompile(`(?i)\bpython[0-9.]*\b|\bpip[0-9.]*\b`)},
	{"Ruby", regexp.MustCompile(`(?i)\bruby\b|\bbundler\b`)},
	{"Java", regexp.MustCompile(`(?i)\bjava\b|\bjavac\b|\bjre\b|\bjdk\b|\bmaven\b|\bgradle\b`)},
	{"Perl", regexp.MustCompile(`(?i)\bperl\b`)},
	{"air", regexp.MustCompile(`(?i)\bair\b`)},
	{"templ", regexp.MustCompile(`(?i)\btempl\b`)},
	{"helm", regexp.MustCompile(`(?i)\bhelm\b`)},
	{"just", regexp.MustCompile(`(?i)\bjust\b`)},
	{"Playwright", regexp.MustCompile(`(?i)\bplaywright\b`)},
	{"Chromium", regexp.MustCompile(`(?i)\bchromium\b`)},
	{"Chrome", regexp.MustCompile(`(?i)\bchrome\b|\bgoogle-chrome\b`)},
	{"Firefox", regexp.MustCompile(`(?i)\bfirefox\b`)},
	{"jq", regexp.MustCompile(`(?i)\bjq\b`)},
	{"unzip", regexp.MustCompile(`(?i)\bunzip\b`)},
	{"net-tools", regexp.MustCompile(`(?i)net-tools`)},
}

func TestDockerfileRuntimeCarriesNoToolchainOrBrowser(t *testing.T) {
	projectRoot := filepath.Join("..", "..")
	containerfilePath := filepath.Join(projectRoot, "containers", "ralph", "Containerfile")

	content, err := os.ReadFile(containerfilePath)
	require.NoError(t, err, "Should be able to read Containerfile")

	runtime := runtimeStage(t, string(content))

	for _, tool := range forbiddenRuntimeTools {
		t.Run(tool.name, func(t *testing.T) {
			assert.False(t, tool.pattern.MatchString(runtime),
				"Runtime stage should not carry %s", tool.name)
		})
	}
}

func TestRuntimeToolChecksRejectReturningTools(t *testing.T) {
	cases := []struct {
		name string
		line string
	}{
		{"Go toolchain", `RUN curl https://go.dev/dl/go1.27.4.linux-amd64.tar.gz | tar -C /usr/local -xzf -`},
		{"Go binary", `COPY --from=builder /usr/local/go/bin/go /usr/local/bin/go`},
		{"Bun", `COPY --from=opencode /usr/local/bun/bin/bun /usr/local/bin/bun`},
		{"Node.js", `RUN apt-get install -y --no-install-recommends nodejs`},
		{"npm", `RUN npm install -g @opencode/cli`},
		{"Rust", `RUN apt-get install -y --no-install-recommends cargo`},
		{"Python", `RUN apt-get install -y --no-install-recommends python3`},
		{"Ruby", `RUN apt-get install -y --no-install-recommends ruby`},
		{"Java", `RUN apt-get install -y --no-install-recommends default-jre`},
		{"Perl", `RUN apt-get install -y --no-install-recommends perl`},
		{"air", `COPY --from=builder /go/bin/air /usr/local/bin/air`},
		{"templ", `COPY --from=builder /go/bin/templ /usr/local/bin/templ`},
		{"helm", `COPY --from=builder /go/bin/helm /usr/local/bin/helm`},
		{"just", `RUN apt-get install -y --no-install-recommends just`},
		{"Playwright", `RUN npx playwright install`},
		{"Chromium", `RUN apt-get install -y --no-install-recommends chromium`},
		{"Chrome", `RUN apt-get install -y --no-install-recommends google-chrome-stable`},
		{"Firefox", `RUN apt-get install -y --no-install-recommends firefox`},
		{"jq", `RUN apt-get install -y --no-install-recommends jq`},
		{"unzip", `RUN apt-get install -y --no-install-recommends unzip`},
		{"net-tools", `RUN apt-get install -y --no-install-recommends net-tools`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			runtime := "FROM ubuntu:26.04\n" + tc.line
			assert.False(t, runtimeCarriesNoForbiddenTool(runtime),
				"Runtime tool check should reject a runtime that adds %s", tc.name)
		})
	}
}

func runtimeCarriesNoForbiddenTool(runtime string) bool {
	for _, tool := range forbiddenRuntimeTools {
		if tool.pattern.MatchString(runtime) {
			return false
		}
	}
	return true
}

func TestDockerfileRuntimeReceivesOnlyOpenCodeBinary(t *testing.T) {
	projectRoot := filepath.Join("..", "..")
	containerfilePath := filepath.Join(projectRoot, "containers", "ralph", "Containerfile")

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
	WebSearch    any                      `json:"websearch"`
	Permissions  []openCodePermissionRule `json:"permissions"`
	Experimental map[string]any           `json:"experimental"`
}

func TestContainerOpenCodeConfigServesSandbox(t *testing.T) {
	projectRoot := filepath.Join("..", "..")
	configPath := filepath.Join(projectRoot, "containers", "ralph", "opencode.json")

	content, err := os.ReadFile(configPath)
	require.NoError(t, err, "Should be able to read containers/ralph/opencode.json")

	var cfg openCodeContainerConfig
	require.NoError(t, json.Unmarshal(content, &cfg), "containers/ralph/opencode.json should be valid JSON")

	assert.Equal(t, "https://opencode.ai/config.json", cfg.Schema,
		"config should declare the published schema")

	assert.Equal(t, "disable", cfg.Update,
		"config should disable updates so the pinned OpenCode version is the one runs use")

	webSearch, ok := cfg.WebSearch.(map[string]any)
	require.True(t, ok, "config should set websearch.provider so an unattended run never gets the provider prompt")
	assert.Equal(t, "random", webSearch["provider"],
		"config should select a web search provider automatically")

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

	_, hasSubagentDepth := cfg.Experimental["subagent_depth"]
	assert.True(t, hasSubagentDepth, "config should keep the configured subagent depth")
}

func TestContainerfileConfiguresOpenCode(t *testing.T) {
	projectRoot := filepath.Join("..", "..")
	containerfilePath := filepath.Join(projectRoot, "containers", "ralph", "Containerfile")

	content, err := os.ReadFile(containerfilePath)
	require.NoError(t, err, "Should be able to read Containerfile")

	assert.Contains(t, string(content), "COPY containers/ralph/opencode.json /root/.config/opencode/opencode.json",
		"Containerfile should install the container OpenCode configuration as the global config")
}

func TestContainerfileKeepsWorkspaceAndDefaultShell(t *testing.T) {
	projectRoot := filepath.Join("..", "..")
	containerfilePath := filepath.Join(projectRoot, "containers", "ralph", "Containerfile")

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
		{"golang.org/x/net", [3]int{0, 62, 0}},
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
