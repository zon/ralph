package validate

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
	writeCanonicalFn func(path string, doc *projectfile.Document) error
	removeFn         func(path string) error
	canonicalPathFn  func(path string) string
	readCallCount    int
	writeCalls       []string
	removeCalls      []string
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

func (m *mockProjectFile) WriteCanonical(path string, doc *projectfile.Document) error {
	m.writeCalls = append(m.writeCalls, path)
	if m.writeCanonicalFn != nil {
		return m.writeCanonicalFn(path, doc)
	}
	return nil
}

func (m *mockProjectFile) Remove(path string) error {
	m.removeCalls = append(m.removeCalls, path)
	if m.removeFn != nil {
		return m.removeFn(path)
	}
	return nil
}

func (m *mockProjectFile) CanonicalPath(path string) string {
	if m.canonicalPathFn != nil {
		return m.canonicalPathFn(path)
	}
	return path
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
	project  ProjectFile
	agent    AgentClient
	model    string
	reporter Reporter
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
	if m.reporter == nil {
		m.reporter = &mockReporter{}
	}
	return &Validator{
		file:     m.project,
		agent:    m.agent,
		model:    m.model,
		reporter: m.reporter,
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

func withReporter(r Reporter) func(*mocks) {
	return func(m *mocks) {
		m.reporter = r
	}
}

type mockReporter struct {
	mu        sync.Mutex
	warns     []string
	warnfFunc func(format string, a ...any)
}

func (m *mockReporter) Warnf(format string, a ...any) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.warnfFunc != nil {
		m.warnfFunc(format, a...)
	}
	m.warns = append(m.warns, fmt.Sprintf(format, a...))
}

func (m *mockReporter) messages() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]string, len(m.warns))
	copy(out, m.warns)
	return out
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

// TestValidateInvokesAgentWithFilePathAndParseError covers the "Agent fixes a
// malformed file" scenario: the command invokes the agent with the file path
// and the parse error before retrying parsing against the updated file.
func TestValidateInvokesAgentWithFilePathAndParseError(t *testing.T) {
	ResetFixCalls()
	doc := &projectfile.Document{Raw: "parsed", Root: map[string]any{"slug": "one"}}
	svc := withMocks(
		withProject(thatParsesAfterFailures(1, doc)),
	)
	_, err := svc.Validate(anyPath, ".")
	require.NoError(t, err)

	calls := FixCalls()
	require.Len(t, calls, 1)
	assert.Equal(t, anyPath, calls[0].path)
	require.Equal(t, "parse failed", calls[0].parseErr.Error())
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

// TestValidateLimitExceededReportsLimitReached covers the "Limit exceeded"
// scenario: when the agent cannot repair the file, the command exits with a
// non-zero status, the error reports that the 10-attempt limit was reached, and
// the error message includes the final parse error.
func TestValidateLimitExceededReportsLimitReached(t *testing.T) {
	ResetFixCalls()
	svc := withMocks(
		withProject(thatAlwaysFailsToParse()),
	)
	_, err := svc.Validate(anyPath, ".")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "10-attempt limit")
	assert.Contains(t, err.Error(), "always fails")
	require.Len(t, FixCalls(), MaxAttempts-1)
}

// TestValidateReportsParseErrorBeforeInvokingAgent covers the item that each
// attempt reports the underlying parse error before invoking the agent: the
// reporter sees the parse error before the agent is called on the same attempt.
func TestValidateReportsParseErrorBeforeInvokingAgent(t *testing.T) {
	ResetFixCalls()
	doc := &projectfile.Document{Raw: "parsed", Root: map[string]any{"slug": "one"}}
	var events []string
	reporter := &mockReporter{
		warnfFunc: func(format string, a ...any) {
			events = append(events, fmt.Sprintf("report:%s", fmt.Sprintf(format, a...)))
		},
	}
	agent := &mockAgentClient{
		fixFunc: func(path string, parseErr error, model string) error {
			events = append(events, "fix:"+parseErr.Error())
			return nil
		},
	}
	svc := withMocks(
		withProject(thatParsesAfterFailures(1, doc)),
		withAgent(agent),
		withReporter(reporter),
	)
	_, err := svc.Validate(anyPath, ".")
	require.NoError(t, err)
	require.Len(t, events, 2)
	assert.Equal(t, "report:project file failed to parse: parse failed", events[0])
	assert.Equal(t, "fix:parse failed", events[1])
}

// TestValidateFixesParseableWithinLimit covers the "File becomes parseable
// within the limit" scenario: a file the agent repairs after several attempts
// exits the loop as soon as parsing succeeds and proceeds to the query checks.
func TestValidateFixesParseableWithinLimit(t *testing.T) {
	ResetFixCalls()
	doc := &projectfile.Document{Raw: "parsed", Root: map[string]any{"slug": "one"}}
	svc := withMocks(
		withProject(thatParsesAfterFailures(3, doc)),
	)
	result, err := svc.Validate(anyPath, ".")
	require.NoError(t, err)
	require.Equal(t, 1, result.ItemCount)
	require.Len(t, FixCalls(), 3)
	require.Less(t, len(FixCalls()), MaxAttempts)
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
// rejected. Because validation now rewrites the file in canonical YAML, the
// unrecognized field must survive the rewrite with its original value rather
// than the file staying byte-identical.
func TestValidateAcceptsUnrecognizedFields(t *testing.T) {
	ResetFixCalls()
	content := "slug: csv-export\nunrelated: [1, 2, 3]\nrequirements:\n  - slug: one\n"
	path := writeTempProject(t, "project.yaml", content)
	svc := withMocks(withProject(&projectFile{}), withAgent(&mockAgentClient{}))

	result, err := svc.Validate(path, ".requirements")
	require.NoError(t, err)
	require.Equal(t, 1, result.ItemCount)
	require.Empty(t, FixCalls())

	rewritten, err := projectfile.Parse(path)
	require.NoError(t, err)
	root, ok := rewritten.Root.(map[string]any)
	require.True(t, ok)
	require.Equal(t, "csv-export", root["slug"])
	require.Equal(t, []any{1, 2, 3}, root["unrelated"])
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

// TestValidateRewritesFileInCanonicalFormat covers the "File rewritten in
// canonical format" scenario: when validation finishes, the file is rewritten
// as canonical YAML and the on-disk content parses to the same document as the
// input.
func TestValidateRewritesFileInCanonicalFormat(t *testing.T) {
	ResetFixCalls()
	content := "slug: csv-export\nrequirements:\n  - slug: one\n  - slug: two\n"
	path := writeTempProject(t, "project.yaml", content)
	svc := withMocks(withProject(&projectFile{}), withAgent(&mockAgentClient{}))

	before, err := projectfile.Parse(path)
	require.NoError(t, err)

	result, err := svc.Validate(path, ".requirements")
	require.NoError(t, err)
	require.Equal(t, 2, result.ItemCount)
	require.Empty(t, FixCalls())

	after, err := projectfile.Parse(path)
	require.NoError(t, err)
	assert.Equal(t, before.Root, after.Root, "on-disk content parses to the same document as the input")
}

// TestValidateUnrecognizedFieldsSurviveRewrite covers the "Unrecognized fields
// survive the rewrite" scenario: fields ralph does not read are present in the
// rewritten output with their original values.
func TestValidateUnrecognizedFieldsSurviveRewrite(t *testing.T) {
	ResetFixCalls()
	content := "slug: csv-export\nunrelated: [1, 2, 3]\nforeign:\n  note: keep me\nrequirements:\n  - slug: one\n"
	path := writeTempProject(t, "project.yaml", content)
	svc := withMocks(withProject(&projectFile{}), withAgent(&mockAgentClient{}))

	_, err := svc.Validate(path, ".requirements")
	require.NoError(t, err)
	require.Empty(t, FixCalls())

	rewritten, err := projectfile.Parse(path)
	require.NoError(t, err)
	root, ok := rewritten.Root.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, []any{1, 2, 3}, root["unrelated"])
	foreign, ok := root["foreign"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "keep me", foreign["note"])
}

// TestValidateJSONRenamedToYAML covers the "JSON file renamed to YAML" scenario:
// a .json input is written to a new file with the same name but a .yaml
// extension, and the original .json file is removed.
func TestValidateJSONRenamedToYAML(t *testing.T) {
	ResetFixCalls()
	jsonContent := `{"slug":"csv-export","requirements":[{"slug":"one"}]}`
	path := writeTempProject(t, "project.json", jsonContent)
	svc := withMocks(withProject(&projectFile{}), withAgent(&mockAgentClient{}))

	result, err := svc.Validate(path, ".requirements")
	require.NoError(t, err)
	require.Equal(t, 1, result.ItemCount)
	require.Empty(t, FixCalls())

	yamlPath := strings.TrimSuffix(path, filepath.Ext(path)) + ".yaml"
	require.FileExists(t, yamlPath)
	_, err = os.Stat(path)
	assert.True(t, os.IsNotExist(err), "the original .json file is removed")

	yamlDoc, err := projectfile.Parse(yamlPath)
	require.NoError(t, err)
	root, ok := yamlDoc.Root.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "csv-export", root["slug"])
	require.Len(t, root["requirements"], 1)
}

// TestValidateCanonicalYAMLUnchanged covers the "Already-canonical YAML file is
// unchanged" scenario: a file already in canonical YAML form is rewritten
// byte-identically.
func TestValidateCanonicalYAMLUnchanged(t *testing.T) {
	ResetFixCalls()
	content := "slug: csv-export\nrequirements:\n    - slug: one\n      items:\n        - a\n"
	path := writeTempProject(t, "project.yaml", content)
	svc := withMocks(withProject(&projectFile{}), withAgent(&mockAgentClient{}))

	_, err := svc.Validate(path, ".requirements")
	require.NoError(t, err)
	require.Empty(t, FixCalls())

	after, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, content, string(after))
}

// TestValidateComposesFileReaderWriteOperation covers the item that validation
// composes the file reader's parse, query, and write operations rather than
// performing any of them itself: a successful validation invokes the reader's
// canonical write.
func TestValidateComposesFileReaderWriteOperation(t *testing.T) {
	ResetFixCalls()
	pf := &mockProjectFile{}
	svc := withMocks(withProject(pf))

	_, err := svc.Validate(anyPath, ".")
	require.NoError(t, err)
	require.Len(t, pf.writeCalls, 1)
	assert.Equal(t, anyPath, pf.writeCalls[0])
	require.Empty(t, pf.removeCalls)
}

// TestValidateRepairedFileRewrittenInCanonicalFormat covers the "File rewritten
// in canonical format" scenario for a file that validates only after agent
// fixes: the repaired file is rewritten in canonical YAML.
func TestValidateRepairedFileRewrittenInCanonicalFormat(t *testing.T) {
	ResetFixCalls()
	broken := "slug: [unclosed\nrequirements:\n"
	fixed := "slug: csv-export\nrequirements:\n  - slug: one\n"
	path := writeTempProject(t, "project.yaml", broken)
	svc := withMocks(withProject(&projectFile{}), withAgent(&mockAgentClient{
		fixFunc: func(p string, parseErr error, model string) error {
			return os.WriteFile(p, []byte(fixed), 0o644)
		},
	}))

	before, err := projectfile.Parse(writeTempProject(t, "before.yaml", fixed))
	require.NoError(t, err)

	result, err := svc.Validate(path, ".requirements")
	require.NoError(t, err)
	require.Equal(t, 1, result.ItemCount)
	require.Len(t, FixCalls(), 1)

	after, err := projectfile.Parse(path)
	require.NoError(t, err)
	assert.Equal(t, before.Root, after.Root, "the repaired file is rewritten in canonical YAML")
}

// TestValidateComposesFileReaderRemoval covers the item that a .json input is
// written to a sibling .yaml file and the original removed: validation composes
// the file reader's canonical write and removal operations rather than doing
// the file work itself.
func TestValidateComposesFileReaderRemoval(t *testing.T) {
	ResetFixCalls()
	jsonPath := "/workspace/repo/projects/project.json"
	yamlPath := "/workspace/repo/projects/project.yaml"
	pf := &mockProjectFile{canonicalPathFn: func(path string) string {
		if path == jsonPath {
			return yamlPath
		}
		return path
	}}
	svc := withMocks(withProject(pf))

	result, err := svc.Validate(jsonPath, ".")
	require.NoError(t, err)
	require.Equal(t, yamlPath, result.Path)
	require.Equal(t, []string{jsonPath}, pf.writeCalls)
	require.Equal(t, []string{jsonPath}, pf.removeCalls)
}

const anyPath = "/workspace/repo/projects/test-project.yaml"
