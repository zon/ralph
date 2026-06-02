package pass

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/zon/ralph/internal/project"
)

var errMockFailure = errors.New("mock failure")

type mockProjectLoader struct {
	loadProjectFn func(path string) (*project.Project, error)
}

func (m *mockProjectLoader) LoadProject(path string) (*project.Project, error) {
	if m.loadProjectFn != nil {
		return m.loadProjectFn(path)
	}
	return &project.Project{
		Slug: "test-project",
		Requirements: []project.Requirement{
			{Slug: "req1", Passing: false},
		},
	}, nil
}

type mockProjectSaver struct {
	saveProjectFn func(path string, p *project.Project) error
}

func (m *mockProjectSaver) SaveProject(path string, p *project.Project) error {
	if m.saveProjectFn != nil {
		return m.saveProjectFn(path, p)
	}
	return nil
}

type deps struct {
	loader ProjectLoader
	saver  ProjectSaver
}

type Opt func(*deps)

func withLoader(l ProjectLoader) Opt {
	return func(d *deps) {
		d.loader = l
	}
}

func withSaver(s ProjectSaver) Opt {
	return func(d *deps) {
		d.saver = s
	}
}

func newPassCmd(opts ...Opt) *PassCmd {
	d := &deps{
		loader: &mockProjectLoader{},
		saver:  &mockProjectSaver{},
	}
	for _, opt := range opts {
		opt(d)
	}
	return New(d.loader, d.saver)
}

func TestRun_Success(t *testing.T) {
	var saved bool
	cmd := newPassCmd(
		withSaver(&mockProjectSaver{
			saveProjectFn: func(path string, p *project.Project) error {
				saved = true
				require.Equal(t, "req1", p.Requirements[0].Slug)
				require.True(t, p.Requirements[0].Passing)
				require.Equal(t, "test.yaml", path)
				return nil
			},
		}),
	)
	err := cmd.Run("test.yaml", "req1", true)
	require.NoError(t, err)
	require.True(t, saved)
}

func TestRun_LoadProjectFails(t *testing.T) {
	cmd := newPassCmd(
		withLoader(&mockProjectLoader{
			loadProjectFn: func(path string) (*project.Project, error) {
				return nil, errMockFailure
			},
		}),
	)
	err := cmd.Run("test.yaml", "req1", true)
	require.Error(t, err)
	require.Contains(t, err.Error(), "mock failure")
}

func TestRun_RequirementNotFound(t *testing.T) {
	cmd := newPassCmd()
	err := cmd.Run("test.yaml", "nonexistent", true)
	require.Error(t, err)
	require.Contains(t, err.Error(), "requirement not found")
}

func TestRun_SaveProjectFails(t *testing.T) {
	cmd := newPassCmd(
		withSaver(&mockProjectSaver{
			saveProjectFn: func(path string, p *project.Project) error {
				return errMockFailure
			},
		}),
	)
	err := cmd.Run("test.yaml", "req1", true)
	require.Error(t, err)
	require.Contains(t, err.Error(), "mock failure")
}

func TestRun_SetsPassingFalse(t *testing.T) {
	var saved bool
	cmd := newPassCmd(
		withSaver(&mockProjectSaver{
			saveProjectFn: func(path string, p *project.Project) error {
				saved = true
				require.False(t, p.Requirements[0].Passing)
				return nil
			},
		}),
	)
	err := cmd.Run("test.yaml", "req1", false)
	require.NoError(t, err)
	require.True(t, saved)
}

func TestRun_DelegatesPath(t *testing.T) {
	var gotLoaderPath, gotSaverPath string
	cmd := newPassCmd(
		withLoader(&mockProjectLoader{
			loadProjectFn: func(path string) (*project.Project, error) {
				gotLoaderPath = path
				return &project.Project{
					Slug: "test",
					Requirements: []project.Requirement{
						{Slug: "my-req"},
					},
				}, nil
			},
		}),
		withSaver(&mockProjectSaver{
			saveProjectFn: func(path string, p *project.Project) error {
				gotSaverPath = path
				return nil
			},
		}),
	)
	err := cmd.Run("my-project.yaml", "my-req", true)
	require.NoError(t, err)
	require.Equal(t, "my-project.yaml", gotLoaderPath)
	require.Equal(t, "my-project.yaml", gotSaverPath)
}
