package repostructure

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// repoRoot walks up from the test working directory to find the repository root.
func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	require.NoError(t, err)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found")
		}
		dir = parent
	}
}

func readRepoFile(t *testing.T, rel string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(repoRoot(t), rel))
	require.NoError(t, err, "expected %s to exist", rel)
	return data
}

// files walks the repository tree, skipping the git directory, the projects
// directory (item definitions reference the paths being removed while the run
// is in progress), and local machine settings.
func files(t *testing.T) []string {
	t.Helper()
	root := repoRoot(t)
	var out []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if rel == ".git" || strings.HasPrefix(rel, ".git"+string(filepath.Separator)) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if rel == "projects" || strings.HasPrefix(rel, "projects"+string(filepath.Separator)) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if rel == ".claude/settings.local.json" || rel == "report.md" || rel == "blocked.md" {
			return nil
		}
		if rel == "internal/repostructure" || strings.HasPrefix(rel, "internal/repostructure"+string(filepath.Separator)) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Ext(rel) == "" {
			return nil
		}
		out = append(out, rel)
		return nil
	})
	require.NoError(t, err)
	return out
}

// isTestFile reports whether the path is a Go test file, whose assertions
// legitimately name the removed paths while asserting their absence.
func isTestFile(rel string) bool {
	return strings.HasSuffix(rel, "_test.go")
}

func TestInstalledStandardsReachableFromAgentInstructions(t *testing.T) {
	// GIVEN a fresh clone of the repository
	// WHEN an agent follows AGENTS.md to find the coding standard
	// THEN it is directed to docs/zpecs/architecture.md and that file exists
	agents := string(readRepoFile(t, "AGENTS.md"))
	assert.Contains(t, agents, "[docs/zpecs/architecture.md](docs/zpecs/architecture.md)", "AGENTS.md must point the coding standard at the installed document")
	assert.Contains(t, agents, "[docs/testing.md](docs/testing.md)", "AGENTS.md must point the testing standard at the installed document")
	assert.Contains(t, agents, "[Architecture Format](docs/zpecs/architecture-outline.md)", "AGENTS.md must reach the architecture format the coding standard depends on")
	_, err := os.Stat(filepath.Join(repoRoot(t), "docs/testing.md"))
	require.NoError(t, err, "docs/testing.md must exist")

	for _, name := range []string{"README", "specs", "architecture-outline", "architecture", "project", "glossary", "requirements", "prompts", "orchestration", "dependencies", "prose"} {
		_, err := os.Stat(filepath.Join(repoRoot(t), "docs/zpecs", name+".md"))
		require.NoError(t, err, "docs/zpecs/%s.md must be installed", name)
	}
}

func TestNoDanglingReferencesToRemovedDocuments(t *testing.T) {
	// GIVEN the documents have been deleted
	// WHEN the repository is searched for links to their old paths
	// THEN no match is found outside of git history
	banned := []string{
		"docs/formats",
		"docs/outline",
		"docs/code.md",
		"docs/writing-requirements.md",
		"docs/prompts.md",
	}
	for _, rel := range files(t) {
		if isTestFile(rel) {
			continue
		}
		data, err := os.ReadFile(filepath.Join(repoRoot(t), rel))
		require.NoError(t, err)
		for _, b := range banned {
			assert.NotContains(t, string(data), b, "%s must not link to removed path %s", rel, b)
		}
	}
}

func TestProjectMechanicsSurviveTheSplit(t *testing.T) {
	// GIVEN docs/projects.md
	// WHEN a reader looks for how an item index is assigned
	// THEN the resolution rules and the dropping of empty outputs are still documented
	content := string(readRepoFile(t, "docs/projects.md"))
	assert.Contains(t, content, "## Item Query")
	assert.Contains(t, content, "Dropping happens before indices are assigned")
	assert.Contains(t, content, "item query yielded no items")
}

func TestProjectOpinionDoesNotSurviveTheSplit(t *testing.T) {
	// GIVEN docs/projects.md
	// WHEN a reader looks for the requirements list or a code entry
	// THEN the document defers to docs/zpecs/project.md instead of describing them
	content := string(readRepoFile(t, "docs/projects.md"))
	assert.Contains(t, content, "[Project Format](zpecs/project.md)")
	assert.NotContains(t, content, "## Requirements")
	assert.NotContains(t, content, "## Items")
	assert.NotContains(t, content, "## Code and Tests")
}

func TestGlossaryKeepsOnlyRalphRuntimeVocabulary(t *testing.T) {
	content := string(readRepoFile(t, "docs/glossary.md"))
	for _, kept := range []string{"## Project", "## Item", "## Item Query", "## Item Key", "## Completion Trailer"} {
		assert.Contains(t, content, kept)
	}
	for _, removed := range []string{"## Component", "## Deep Module", "## Feature", "## Implementation Module", "## Pure Module", "## Orchestration Module"} {
		assert.NotContains(t, content, removed)
	}
}

func TestUserFacingDocsDescribeBareTrailer(t *testing.T) {
	// GIVEN the user-facing documentation
	// WHEN a reader looks for the completion trailer
	// THEN it is described as a bare <branch>-<index> line and never as
	// "Ralph item ... completed"
	for _, rel := range []string{
		"README.md",
		"docs/iterations.md",
		"docs/glossary.md",
		"docs/projects.md",
		"docs/config.md",
		"docs/cli.md",
	} {
		content := string(readRepoFile(t, rel))
		assert.Contains(t, content, "<branch>-<index>", "%s must describe the trailer as <branch>-<index>", rel)
		assert.NotContains(t, content, "Ralph item", "%s must not describe the old Ralph item ... completed form", rel)
	}
	iterations := string(readRepoFile(t, "docs/iterations.md"))
	assert.NotContains(t, iterations, "key mismatch", "docs/iterations.md must drop the key-mismatch warning")
	assert.NotContains(t, iterations, "carries a key", "docs/iterations.md must drop the key-mismatch warning")
}

func TestNoSkillsShipInTheRepository(t *testing.T) {
	_, err := os.Stat(filepath.Join(repoRoot(t), ".claude", "skills"))
	assert.True(t, os.IsNotExist(err), ".claude/skills must be removed")
	agents := string(readRepoFile(t, "AGENTS.md"))
	assert.NotContains(t, agents, "Ralph Skills")
	assert.Contains(t, agents, "installed from the specs repository")
}

func TestNoReferencesToRemovedModules(t *testing.T) {
	// GIVEN the modules have been deleted
	// WHEN the repository is searched for imports of internal/skills or internal/orchestration/setup
	// THEN no match is found and the full test suite builds
	for _, rel := range files(t) {
		if filepath.Ext(rel) != ".go" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(repoRoot(t), rel))
		require.NoError(t, err)
		assert.NotContains(t, string(data), `"github.com/zon/ralph/internal/skills"`, "%s must not import the removed skills module", rel)
		assert.NotContains(t, string(data), `"github.com/zon/ralph/internal/orchestration/setup"`, "%s must not import the removed setup module", rel)
		assert.NotContains(t, string(data), `"github.com/zon/ralph/internal/architecture"`, "%s must not import the removed architecture module", rel)
	}
	_, err := os.Stat(filepath.Join(repoRoot(t), "internal", "skills"))
	assert.True(t, os.IsNotExist(err), "internal/skills must be deleted")
	_, err = os.Stat(filepath.Join(repoRoot(t), "internal", "orchestration", "setup"))
	assert.True(t, os.IsNotExist(err), "internal/orchestration/setup must be deleted")
	_, err = os.Stat(filepath.Join(repoRoot(t), "internal", "architecture"))
	assert.True(t, os.IsNotExist(err), "internal/architecture must be deleted")
}

func TestNoGoFileReadsOrWritesSpecsArchitecture(t *testing.T) {
	for _, rel := range files(t) {
		if filepath.Ext(rel) != ".go" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(repoRoot(t), rel))
		require.NoError(t, err)
		assert.NotContains(t, string(data), "specs/architecture.yaml", "%s must not read or write specs/architecture.yaml", rel)
	}
}

func TestEveryReadmeLinkResolves(t *testing.T) {
	// GIVEN the README
	// WHEN each relative link in it is checked
	// THEN every target file exists
	content := string(readRepoFile(t, "README.md"))
	root := repoRoot(t)
	for _, link := range markdownLinks(content) {
		if strings.HasPrefix(link, "http") || strings.HasPrefix(link, "#") {
			continue
		}
		path := strings.TrimPrefix(strings.SplitN(link, "#", 2)[0], "./")
		if path == "" {
			continue
		}
		_, err := os.Stat(filepath.Join(root, path))
		assert.NoError(t, err, "README link %q must resolve to an existing file", link)
	}
}

// markdownLinks extracts the link targets of markdown links in content.
func markdownLinks(content string) []string {
	var links []string
	rest := content
	for {
		open := strings.Index(rest, "](")
		if open < 0 {
			break
		}
		start := open + 2
		end := strings.Index(rest[start:], ")")
		if end < 0 {
			break
		}
		links = append(links, rest[start:start+end])
		rest = rest[start+end:]
	}
	return links
}
