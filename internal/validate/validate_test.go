package validate

import (
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/zon/ralph/internal/project"
)

type mockProjectClient struct {
	loadFunc     func(path string) (*project.Project, error)
	saveFunc     func(path string, proj *project.Project) error
	readFileFunc func(path string) ([]byte, error)
	readCallCount int
}

func (m *mockProjectClient) Load(path string) (*project.Project, error) {
	if m.loadFunc != nil {
		return m.loadFunc(path)
	}
	return nil, nil
}

func (m *mockProjectClient) Save(path string, proj *project.Project) error {
	project.SetLastSaved(proj)
	if m.saveFunc != nil {
		return m.saveFunc(path, proj)
	}
	return nil
}

func (m *mockProjectClient) ReadFile(path string) ([]byte, error) {
	if m.readFileFunc != nil {
		return m.readFileFunc(path)
	}
	m.readCallCount++
	return []byte(fmt.Sprintf("content-%d", m.readCallCount)), nil
}

var (
	agentFixMu    sync.Mutex
	agentFixCalls []struct {
		path    string
		loadErr error
		model   string
	}
)

func RecordFixCall(path string, loadErr error, model string) {
	agentFixMu.Lock()
	agentFixCalls = append(agentFixCalls, struct {
		path    string
		loadErr error
		model   string
	}{path, loadErr, model})
	agentFixMu.Unlock()
}

func FixCalls() []struct {
	path    string
	loadErr error
	model   string
} {
	agentFixMu.Lock()
	defer agentFixMu.Unlock()
	calls := make([]struct {
		path    string
		loadErr error
		model   string
	}, len(agentFixCalls))
	copy(calls, agentFixCalls)
	return calls
}

func ResetFixCalls() {
	agentFixMu.Lock()
	agentFixCalls = nil
	agentFixMu.Unlock()
}

type mockAgentClient struct {
	fixFunc func(path string, loadErr error, model string) error
}

func (m *mockAgentClient) FixProject(path string, loadErr error, model string) error {
	RecordFixCall(path, loadErr, model)
	if m.fixFunc != nil {
		return m.fixFunc(path, loadErr, model)
	}
	return nil
}

type mocks struct {
	project ProjectClient
	agent   AgentClient
	model   string
}

func withMocks(mockFns ...func(*mocks)) *Validator {
	m := &mocks{}
	for _, fn := range mockFns {
		fn(m)
	}
	if m.project == nil {
		m.project = &mockProjectClient{}
	}
	if m.agent == nil {
		m.agent = &mockAgentClient{}
	}
	return &Validator{
		project: m.project,
		agent:   m.agent,
		model:   m.model,
	}
}

func withModel(model string) func(*mocks) {
	return func(m *mocks) {
		m.model = model
	}
}

func withProject(pc ProjectClient) func(*mocks) {
	return func(m *mocks) {
		m.project = pc
	}
}

func withAgent(ac AgentClient) func(*mocks) {
	return func(m *mocks) {
		m.agent = ac
	}
}

func thatLoads(proj *project.Project) ProjectClient {
	return &mockProjectClient{
		loadFunc: func(path string) (*project.Project, error) {
			return proj, nil
		},
	}
}

func thatLoadsAfterFailures(n int, proj *project.Project) ProjectClient {
	attempts := 0
	return &mockProjectClient{
		loadFunc: func(path string) (*project.Project, error) {
			attempts++
			if attempts <= n {
				return nil, &mockLoadError{msg: "load failed"}
			}
			return proj, nil
		},
	}
}

func thatAlwaysFailsToLoad() ProjectClient {
	return &mockProjectClient{
		loadFunc: func(path string) (*project.Project, error) {
			return nil, &mockLoadError{msg: "always fails"}
		},
	}
}

func thatAlwaysFailsToLoadWithUnchangedFile() ProjectClient {
	return &mockProjectClient{
		loadFunc: func(path string) (*project.Project, error) {
			return nil, &mockLoadError{msg: "always fails"}
		},
		readFileFunc: func(path string) ([]byte, error) {
			return []byte("unchanged content"), nil
		},
	}
}

func thatLoadsButFailsToSave(proj *project.Project) ProjectClient {
	return &mockProjectClient{
		loadFunc: func(path string) (*project.Project, error) {
			return proj, nil
		},
		saveFunc: func(path string, proj *project.Project) error {
			return &mockSaveError{msg: "save failed"}
		},
	}
}

type mockLoadError struct {
	msg string
}

func (e *mockLoadError) Error() string {
	return e.msg
}

type mockSaveError struct {
	msg string
}

func (e *mockSaveError) Error() string {
	return e.msg
}

func thatFailsToFix() AgentClient {
	return &mockAgentClient{
		fixFunc: func(path string, loadErr error, model string) error {
			return &mockFixError{msg: "agent fix failed"}
		},
	}
}

type mockFixError struct {
	msg string
}

func (e *mockFixError) Error() string {
	return e.msg
}

func TestValidateSucceedsOnFirstLoad(t *testing.T) {
	project.ResetLoadAttempts()
	ResetFixCalls()
	project.SetLastSaved(nil)
	proj := project.Any()
	svc := withMocks(
		withProject(thatLoads(proj)),
	)
	result, err := svc.Validate(project.AnyPath())
	require.NoError(t, err)
	require.Equal(t, proj, result)
	require.Equal(t, proj, project.LastSaved())
	require.Empty(t, FixCalls())
}

func TestValidateRepairsThenSucceeds(t *testing.T) {
	project.ResetLoadAttempts()
	ResetFixCalls()
	proj := project.Any()
	svc := withMocks(
		withProject(thatLoadsAfterFailures(1, proj)),
	)
	result, err := svc.Validate(project.AnyPath())
	require.NoError(t, err)
	require.Equal(t, proj, result)
	require.Len(t, FixCalls(), 1)
}

func TestValidateGivesUpAfterMaxAttempts(t *testing.T) {
	project.ResetLoadAttempts()
	ResetFixCalls()
	svc := withMocks(
		withProject(thatAlwaysFailsToLoad()),
	)
	_, err := svc.Validate(project.AnyPath())
	require.Error(t, err)
	require.Len(t, FixCalls(), MaxAttempts-1)
}

func TestValidatePropagatesAgentFailure(t *testing.T) {
	project.ResetLoadAttempts()
	ResetFixCalls()
	svc := withMocks(
		withProject(thatAlwaysFailsToLoad()),
		withAgent(thatFailsToFix()),
	)
	_, err := svc.Validate(project.AnyPath())
	require.Error(t, err)
}

func TestValidatePropagatesSaveFailure(t *testing.T) {
	project.ResetLoadAttempts()
	ResetFixCalls()
	svc := withMocks(
		withProject(thatLoadsButFailsToSave(project.Any())),
	)
	_, err := svc.Validate(project.AnyPath())
	require.Error(t, err)
}

func TestValidateUsesValidateSpecificModel(t *testing.T) {
	project.ResetLoadAttempts()
	ResetFixCalls()
	svc := withMocks(
		withModel("validate-model"),
		withProject(thatLoadsAfterFailures(1, project.Any())),
	)
	_, err := svc.Validate(project.AnyPath())
	require.NoError(t, err)
	require.Equal(t, "validate-model", FixCalls()[0].model)
}

func TestValidateFallsBackToMainModel(t *testing.T) {
	project.ResetLoadAttempts()
	ResetFixCalls()
	svc := withMocks(
		withModel("main-model"),
		withProject(thatLoadsAfterFailures(1, project.Any())),
	)
	_, err := svc.Validate(project.AnyPath())
	require.NoError(t, err)
	require.Equal(t, "main-model", FixCalls()[0].model)
}

func TestValidateFailsFastWhenAgentMakesNoChange(t *testing.T) {
	project.ResetLoadAttempts()
	ResetFixCalls()
	svc := withMocks(
		withProject(thatAlwaysFailsToLoadWithUnchangedFile()),
	)
	_, err := svc.Validate(project.AnyPath())
	require.ErrorIs(t, err, ErrNoChange)
	require.Len(t, FixCalls(), 1)
}