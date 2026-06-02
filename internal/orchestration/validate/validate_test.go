package validate

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/zon/ralph/internal/project"
)

var errMockFailure = errors.New("mock failure")

type mockValidator struct {
	validateFn func(path string) (*project.Project, error)
}

func (m *mockValidator) Validate(path string) (*project.Project, error) {
	if m.validateFn != nil {
		return m.validateFn(path)
	}
	return &project.Project{Slug: "test", Requirements: []project.Requirement{{Slug: "req1"}}}, nil
}

func TestValidateCmd_Run_Success(t *testing.T) {
	cmd := New(&mockValidator{})
	proj, err := cmd.Run("test.yaml")
	require.NoError(t, err)
	require.NotNil(t, proj)
	require.Equal(t, "test", proj.Slug)
	require.Len(t, proj.Requirements, 1)
}

func TestValidateCmd_Run_ValidatorFailure(t *testing.T) {
	cmd := New(&mockValidator{
		validateFn: func(path string) (*project.Project, error) {
			return nil, errMockFailure
		},
	})
	_, err := cmd.Run("test.yaml")
	require.Error(t, err)
	require.Contains(t, err.Error(), "mock failure")
}

func TestValidateCmd_Run_DelegatesPath(t *testing.T) {
	var gotPath string
	cmd := New(&mockValidator{
		validateFn: func(path string) (*project.Project, error) {
			gotPath = path
			return &project.Project{}, nil
		},
	})
	_, err := cmd.Run("my-project.yaml")
	require.NoError(t, err)
	require.Equal(t, "my-project.yaml", gotPath)
}
