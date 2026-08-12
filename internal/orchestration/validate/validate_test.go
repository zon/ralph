package validate

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zon/ralph/internal/projectfile"
)

func writeTempProject(t *testing.T, name, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	return path
}

type mockProjectFile struct {
	parseFunc        func(path string) (*projectfile.Document, error)
	resolveItemsFunc func(doc *projectfile.Document, query string) ([]any, error)
	readFileFunc     func(path string) ([]byte, error)
	readCallCount    int
}

func (m *mockProjectFile) Parse(path string) (*projectfile.Document, error) {
	if m.parseFunc != nil {
		return m.parseFunc(path)
	}
	return &projectfile.Document{Raw: "parsed", Root: map[string]any{"slug": "one"}}, nil
}

func (m *mockProjectFile) ResolveItems(doc *projectfile.Document, query string) ([]any, error) {
	if m.resolveItemsFunc != nil {
		return m.resolveItemsFunc(doc, query)
	}
	return []any{map[string]any{"slug": "one"}}, nil
}

func (m *mockProjectFile) ReadFile(path string) ([]byte, error) {
	if m.readFileFunc != nil {
		return m.readFileFunc(path)
	}
	m.readCallCount++
	return []byte(fmt.Sprintf("content-%d", m.readCallCount)), nil
}

type fixCall struct {
	path     string
	parseErr error
	model    string
}

var (
	fixCallMu  sync.Mutex
	fixCallLog []fixCall
)

func RecordFixCall(path string, parseErr error, model string) {
	fixCallMu.Lock()
	fixCallLog = append(fixCallLog, fixCall{path, parseErr, model})
	fixCallMu.Unlock()
}

func FixCalls() []fixCall {
	fixCallMu.Lock()
	defer fixCallMu.Unlock()
	calls := make([]fixCall, len(fixCallLog))
	copy(calls, fixCallLog)
	return calls
}

func ResetFixCalls() {
	fixCallMu.Lock()
	fixCallLog = nil
	fixCallMu.Unlock()
}

type mockAgentClient struct {
	fixFunc func(path string, parseErr error, model string) error
}

func (m *mockAgentClient) FixProject(path string, parseErr error, model string) error {
	RecordFixCall(path, parseErr, model)
	if m.fixFunc != nil {
		return m.fixFunc(path, parseErr, model)
	}
	return nil
}

type mocks struct {
	project ProjectFile
	agent   AgentClient
	model   string
}

func withMocks(opts ...func(*mocks)) *Validator {
	m := &mocks{}
	for _, fn := range opts {
		fn(m)
	}
	if m.project == nil {
		m.project = &mockProjectFile{}
	}
	if m.agent == nil {
		m.agent = &mockAgentClient{}
	}
	return &Validator{
		file:  m.project,
		agent: m.agent,
		model: m.model,
	}
}

func withProject(pf ProjectFile) func(*mocks) {
	return func(m *mocks) {
		m.project = pf
	}
}

func withAgent(ac AgentClient) func(*mocks) {
	return func(m *mocks) {
		m.agent = ac
	}
}

func withModel(model string) func(*mocks) {
	return func(m *mocks) {
		m.model = model
	}
}

func thatParses(doc *projectfile.Document) ProjectFile {
	return &mockProjectFile{
		parseFunc: func(path string) (*projectfile.Document, error) {
			return doc, nil
		},
	}
}

func thatParsesAfterFailures(n int, doc *projectfile.Document) ProjectFile {
	attempts := 0
	return &mockProjectFile{
		parseFunc: func(path string) (*projectfile.Document, error) {
			attempts++
			if attempts <= n {
				return nil, &mockParseError{msg: "parse failed"}
			}
			return doc, nil
		},
	}
}

func thatAlwaysFailsToParse() ProjectFile {
	return &mockProjectFile{
		parseFunc: func(path string) (*projectfile.Document, error) {
			return nil, &mockParseError{msg: "always fails"}
		},
	}
}

func thatAlwaysFailsToParseWithUnchangedFile() ProjectFile {
	return &mockProjectFile{
		parseFunc: func(path string) (*projectfile.Document, error) {
			return nil, &mockParseError{msg: "always fails"}
		},
		readFileFunc: func(path string) ([]byte, error) {
			return []byte("unchanged content"), nil
		},
	}
}

type mockParseError struct {
	msg string
}

func (e *mockParseError) Error() string {
	return e.msg
}

func thatFailsToFix() AgentClient {
	return &mockAgentClient{
		fixFunc: func(path string, parseErr error, model string) error {
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

// TestFixLoopSucceedsOnFirstParse covers the "Well-formed project" scenario and
// the item that validation performs its three checks without invoking an agent
// when the file already parses.
func TestFixLoopSucceedsOnFirstParse(t *testing.T) {
	ResetFixCalls()
	doc := &projectfile.Document{Raw: "parsed", Root: map[string]any{"slug": "one"}}
	svc := withMocks(
		withProject(thatParses(doc)),
	)
	result, err := svc.Validate(anyPath, ".")
	require.NoError(t, err)
	require.Equal(t, 1, result.ItemCount)
	require.Equal(t, anyPath, result.Path)
	require.Empty(t, FixCalls())
}

func TestValidateRepairsThenSucceeds(t *testing.T) {
	ResetFixCalls()
	doc := &projectfile.Document{Raw: "parsed", Root: map[string]any{"slug": "one"}}
	svc := withMocks(
		withProject(thatParsesAfterFailures(1, doc)),
	)
	result, err := svc.Validate(anyPath, ".")
	require.NoError(t, err)
	require.Equal(t, 1, result.ItemCount)
	require.Len(t, FixCalls(), 1)
}

func TestValidateGivesUpAfterMaxAttempts(t *testing.T) {
	ResetFixCalls()
	svc := withMocks(
		withProject(thatAlwaysFailsToParse()),
	)
	_, err := svc.Validate(anyPath, ".")
	require.Error(t, err)
	require.Len(t, FixCalls(), MaxAttempts-1)
}

func TestValidateFailsFastWhenAgentMakesNoChange(t *testing.T) {
	ResetFixCalls()
	svc := withMocks(
		withProject(thatAlwaysFailsToParseWithUnchangedFile()),
	)
	_, err := svc.Validate(anyPath, ".")
	require.ErrorIs(t, err, ErrNoChange)
	require.Len(t, FixCalls(), 1)
}

func TestValidatePropagatesAgentFailure(t *testing.T) {
	ResetFixCalls()
	svc := withMocks(
		withProject(thatAlwaysFailsToParse()),
		withAgent(thatFailsToFix()),
	)
	_, err := svc.Validate(anyPath, ".")
	require.Error(t, err)
}

func TestValidateUsesValidateSpecificModel(t *testing.T) {
	ResetFixCalls()
	doc := &projectfile.Document{Raw: "parsed", Root: map[string]any{"slug": "one"}}
	svc := withMocks(
		withModel("validate-model"),
		withProject(thatParsesAfterFailures(1, doc)),
	)
	_, err := svc.Validate(anyPath, ".")
	require.NoError(t, err)
	require.Equal(t, "validate-model", FixCalls()[0].model)
}

func TestValidateFallsBackToMainModel(t *testing.T) {
	ResetFixCalls()
	doc := &projectfile.Document{Raw: "parsed", Root: map[string]any{"slug": "one"}}
	svc := withMocks(
		withModel("main-model"),
		withProject(thatParsesAfterFailures(1, doc)),
	)
	_, err := svc.Validate(anyPath, ".")
	require.NoError(t, err)
	require.Equal(t, "main-model", FixCalls()[0].model)
}

// TestValidateAcceptsUnrecognizedFields covers the "Unrecognized fields
// accepted" scenario and the item that no field is required and no field is
// rejected.
func TestValidateAcceptsUnrecognizedFields(t *testing.T) {
	ResetFixCalls()
	content := "slug: csv-export\nunrelated: [1, 2, 3]\nrequirements:\n  - slug: one\n"
	path := writeTempProject(t, "project.yaml", content)
	svc := withMocks(withProject(&projectFile{}), withAgent(&mockAgentClient{}))

	result, err := svc.Validate(path, ".requirements")
	require.NoError(t, err)
	require.Equal(t, 1, result.ItemCount)
	require.Empty(t, FixCalls())

	after, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, content, string(after))
}

// TestValidateAcceptsTopLevelArrayOfStrings covers the "Items with no
// conventional shape accepted" scenario.
func TestValidateAcceptsTopLevelArrayOfStrings(t *testing.T) {
	ResetFixCalls()
	path := writeTempProject(t, "project.yaml", "- Add a CSV serializer\n- Add an export endpoint\n")
	svc := withMocks(withProject(&projectFile{}), withAgent(&mockAgentClient{}))

	result, err := svc.Validate(path, ".")
	require.NoError(t, err)
	require.Equal(t, 2, result.ItemCount)
	require.Empty(t, FixCalls())
}

// TestValidateQueryEvaluationFailureInvokesNoAgent covers the "Query evaluation
// failure" scenario and the item that a query which cannot be evaluated exits
// reporting the query error without invoking an agent.
func TestValidateQueryEvaluationFailureInvokesNoAgent(t *testing.T) {
	ResetFixCalls()
	path := writeTempProject(t, "project.yaml", "foo: 1\n")
	svc := withMocks(withProject(&projectFile{}), withAgent(&mockAgentClient{}))

	_, err := svc.Validate(path, ".foo.bar")
	require.Error(t, err)
	assert.Contains(t, err.Error(), ".foo.bar")
	assert.NotContains(t, err.Error(), "yielded no items")
	require.Empty(t, FixCalls())
}

// TestValidateNoItemsInvokesNoAgent covers the "Query yields no items" scenario
// and the item that a query producing no output exits with an error naming the
// query without invoking an agent.
func TestValidateNoItemsInvokesNoAgent(t *testing.T) {
	ResetFixCalls()
	path := writeTempProject(t, "project.yaml", "requirements: []\n")
	svc := withMocks(withProject(&projectFile{}), withAgent(&mockAgentClient{}))

	_, err := svc.Validate(path, ".requirements")
	require.Error(t, err)
	assert.Equal(t, "item query yielded no items: .requirements", err.Error())
	require.Empty(t, FixCalls())
}

// TestValidateQueryRunsAgainstRepairedFile covers the item that after the fix
// loop exits, the query checks run against the repaired file.
func TestValidateQueryRunsAgainstRepairedFile(t *testing.T) {
	ResetFixCalls()
	attempts := 0
	svc := withMocks(
		withProject(&mockProjectFile{
			parseFunc: func(path string) (*projectfile.Document, error) {
				attempts++
				if attempts == 1 {
					return nil, &mockParseError{msg: "first parse fails"}
				}
				return &projectfile.Document{Raw: "repaired", Root: map[string]any{"slug": "one"}}, nil
			},
			resolveItemsFunc: func(doc *projectfile.Document, query string) ([]any, error) {
				return nil, nil
			},
		}),
	)
	_, err := svc.Validate(anyPath, ".")
	require.Error(t, err)
	assert.Equal(t, "item query yielded no items: .", err.Error())
	require.Len(t, FixCalls(), 1)
}

const anyPath = "/workspace/repo/projects/test-project.yaml"
