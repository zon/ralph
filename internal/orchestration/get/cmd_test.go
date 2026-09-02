package get

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/project"
)

type mockProject struct {
	resolveErr    error
	complete      []string
	completeErr   error
	incomplete    []project.Item
	incompleteErr error
	validateErr   error

	resolveCalled bool
	resolvePath   string
	resolveQuery  string
	completeProj  *project.Project
	lastBase      string
}

func (m *mockProject) Resolve(path string, query string) (*project.Project, error) {
	m.resolveCalled = true
	m.resolvePath = path
	m.resolveQuery = query
	if m.resolveErr != nil {
		return nil, m.resolveErr
	}
	return project.WithItems(5), nil
}

func (m *mockProject) Complete(proj *project.Project, base string) ([]string, error) {
	m.completeProj = proj
	m.lastBase = base
	return m.complete, m.completeErr
}

func (m *mockProject) Incomplete(proj *project.Project, base string) ([]project.Item, error) {
	m.completeProj = proj
	m.lastBase = base
	return m.incomplete, m.incompleteErr
}

func (m *mockProject) ValidateFile(path string) error {
	return m.validateErr
}

func newMock() (*mockProject, *bytes.Buffer) {
	return &mockProject{}, &bytes.Buffer{}
}

func defaultConfig() *config.RalphConfig {
	return &config.RalphConfig{Items: ".requirements", DefaultBranch: "main"}
}

func TestScenarioCompleteWithoutProjectFile(t *testing.T) {
	m, buf := newMock()
	err := NewCmd(m, buf).Complete(defaultConfig(), Flags{})
	require.NoError(t, err)
	assert.False(t, m.resolveCalled, "no item array is resolved without a project file")
	assert.Nil(t, m.completeProj, "completion is read with no item array resolved")
}

func TestScenarioCompleteWithProjectFile(t *testing.T) {
	m, buf := newMock()
	err := NewCmd(m, buf).Complete(defaultConfig(), Flags{ProjectFile: "./projects/csv-export.yaml"})
	require.NoError(t, err)
	assert.True(t, m.resolveCalled)
	assert.Equal(t, "./projects/csv-export.yaml", m.resolvePath)
	assert.NotNil(t, m.completeProj, "the resolved item array bounds the reported indices")
}

func TestScenarioCompletedHashesPrinted(t *testing.T) {
	m, buf := newMock()
	m.complete = []string{"abc1234", "efg5678", "hij9012"}
	err := NewCmd(m, buf).Complete(defaultConfig(), Flags{})
	require.NoError(t, err)
	assert.Equal(t, "abc1234\nefg5678\nhij9012\n", buf.String())
}

func TestScenarioNothingCompletePrintsNothing(t *testing.T) {
	m, buf := newMock()
	err := NewCmd(m, buf).Complete(defaultConfig(), Flags{})
	require.NoError(t, err)
	assert.Equal(t, "", buf.String())
}

func TestScenarioCompleteWorksAfterProjectFileRemoved(t *testing.T) {
	m, buf := newMock()
	m.complete = []string{"abc1234", "efg5678", "hij9012"}
	err := NewCmd(m, buf).Complete(defaultConfig(), Flags{})
	require.NoError(t, err)
	assert.False(t, m.resolveCalled, "no project file is read when none is given")
	assert.Equal(t, "abc1234\nefg5678\nhij9012\n", buf.String())
}

func TestScenarioRemainingItemsPrinted(t *testing.T) {
	m, buf := newMock()
	m.incomplete = []project.Item{
		{Index: 1, Value: map[string]any{"slug": "csv-export"}},
		{Index: 3, Value: "plain-string-item"},
	}
	err := NewCmd(m, buf).Incomplete(defaultConfig(), Flags{ProjectFile: "./projects/csv-export.yaml"}, false)
	require.NoError(t, err)

	var got []any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &got))
	assert.Equal(t, []any{
		map[string]any{"slug": "csv-export"},
		"plain-string-item",
	}, got)
}

func TestScenarioIndicesEmittedInsteadOfItems(t *testing.T) {
	m, buf := newMock()
	m.incomplete = []project.Item{
		{Index: 1, Value: "item-1"},
		{Index: 4, Value: "item-4"},
	}
	err := NewCmd(m, buf).Incomplete(defaultConfig(), Flags{ProjectFile: "./projects/csv-export.yaml"}, true)
	require.NoError(t, err)
	assert.Equal(t, "[1,4]", strings.TrimSpace(buf.String()))
}

func TestScenarioEveryItemComplete(t *testing.T) {
	m, buf := newMock()
	err := NewCmd(m, buf).Incomplete(defaultConfig(), Flags{ProjectFile: "./projects/csv-export.yaml"}, false)
	require.NoError(t, err)
	assert.Equal(t, "[]", strings.TrimSpace(buf.String()))
}

func TestScenarioIncompleteWithoutProjectFile(t *testing.T) {
	m, buf := newMock()
	err := NewCmd(m, buf).Incomplete(defaultConfig(), Flags{}, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "project file path is required")
	assert.False(t, m.resolveCalled)
}

func TestCompleteFileNotFound(t *testing.T) {
	m, buf := newMock()
	m.validateErr = errors.New("project file not found: /no/such/project.yaml")
	err := NewCmd(m, buf).Complete(defaultConfig(), Flags{ProjectFile: "/no/such/project.yaml"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "project file not found")
	assert.False(t, m.resolveCalled, "resolution is skipped when the file does not exist")
}

func TestIncompleteFileNotFound(t *testing.T) {
	m, buf := newMock()
	m.validateErr = errors.New("project file not found: /no/such/project.yaml")
	err := NewCmd(m, buf).Incomplete(defaultConfig(), Flags{ProjectFile: "/no/such/project.yaml"}, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "project file not found")
	assert.False(t, m.resolveCalled)
}

func TestItemsFlagTakesPrecedence(t *testing.T) {
	m, buf := newMock()
	cfg := &config.RalphConfig{Items: ".requirements", DefaultBranch: "main"}
	err := NewCmd(m, buf).Complete(cfg, Flags{ProjectFile: "p.yaml", Items: ".spec.tasks"})
	require.NoError(t, err)
	assert.Equal(t, ".spec.tasks", m.resolveQuery)
}

func TestItemsConfigUsedWhenNoFlag(t *testing.T) {
	m, buf := newMock()
	cfg := &config.RalphConfig{Items: ".requirements", DefaultBranch: "main"}
	err := NewCmd(m, buf).Complete(cfg, Flags{ProjectFile: "p.yaml"})
	require.NoError(t, err)
	assert.Equal(t, ".requirements", m.resolveQuery)
}

func TestItemsDefaultsToDot(t *testing.T) {
	m, buf := newMock()
	cfg := &config.RalphConfig{DefaultBranch: "main"}
	err := NewCmd(m, buf).Complete(cfg, Flags{ProjectFile: "p.yaml"})
	require.NoError(t, err)
	assert.Equal(t, ".", m.resolveQuery)
}

func TestBaseFlagTakesPrecedence(t *testing.T) {
	m, buf := newMock()
	cfg := &config.RalphConfig{DefaultBranch: "main"}
	err := NewCmd(m, buf).Complete(cfg, Flags{ProjectFile: "p.yaml", Base: "develop"})
	require.NoError(t, err)
	assert.Equal(t, "develop", m.lastBase)
}

func TestBaseConfigUsedWhenNoFlag(t *testing.T) {
	m, buf := newMock()
	cfg := &config.RalphConfig{DefaultBranch: "develop"}
	err := NewCmd(m, buf).Incomplete(cfg, Flags{ProjectFile: "p.yaml"}, false)
	require.NoError(t, err)
	assert.Equal(t, "develop", m.lastBase)
}

func TestCompleteSurfacesCommitLogError(t *testing.T) {
	m, buf := newMock()
	m.completeErr = errors.New("git log boom")
	err := NewCmd(m, buf).Complete(defaultConfig(), Flags{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "git log boom")
}

func TestIncompleteSurfacesResolutionError(t *testing.T) {
	m, buf := newMock()
	m.resolveErr = errors.New("item query yielded no items: .")
	err := NewCmd(m, buf).Incomplete(defaultConfig(), Flags{ProjectFile: "p.yaml"}, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "item query yielded no items")
}
