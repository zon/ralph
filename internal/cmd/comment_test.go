package cmd

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/services"
)

func cleanupLogs(t *testing.T, svcs []config.Service) {
	t.Helper()
	for _, svc := range svcs {
		os.Remove(services.LogFileName(svc.Name))
	}
}

func TestStartServicesIfNeeded_NoServices(t *testing.T) {
	cleanup := startServicesIfNeeded(true, []config.Service{{Name: "test", Command: "sleep", Args: []string{"1"}}}, nil)
	assert.Nil(t, cleanup, "should return nil when noServices is true")
}

func TestStartServicesIfNeeded_EmptyServiceList(t *testing.T) {
	cleanup := startServicesIfNeeded(false, []config.Service{}, nil)
	assert.Nil(t, cleanup, "should return nil when service list is empty")
}

func TestStartServicesIfNeeded_ServiceStartSuccessNoRegistrar(t *testing.T) {
	svcs := []config.Service{
		{Name: "start-test-svc1", Command: "sleep", Args: []string{"30"}},
	}
	t.Cleanup(func() { cleanupLogs(t, svcs) })

	cleanup := startServicesIfNeeded(false, svcs, nil)
	require.NotNil(t, cleanup, "should return cleanup function when services start successfully")

	assert.NotPanics(t, func() { cleanup() }, "cleanup should stop services without panicking")
}

func TestStartServicesIfNeeded_ServiceStartSuccessWithRegistrar(t *testing.T) {
	svcs := []config.Service{
		{Name: "start-test-svc2", Command: "sleep", Args: []string{"30"}},
	}
	t.Cleanup(func() { cleanupLogs(t, svcs) })

	var registeredCleanup func()
	registrar := func(c func()) {
		registeredCleanup = c
	}

	cleanup := startServicesIfNeeded(false, svcs, registrar)
	require.NotNil(t, cleanup, "should return cleanup function")
	require.NotNil(t, registeredCleanup, "cleanup should be registered")

	registeredCleanup()
}

func TestStartServicesIfNeeded_ServiceStartFailure(t *testing.T) {
	svcs := []config.Service{
		{Name: "nonexistent-service-xyz", Command: "nonexistent-command-xyz", Args: []string{}},
	}

	cleanup := startServicesIfNeeded(false, svcs, nil)
	assert.Nil(t, cleanup, "should return nil when services fail to start")
}

func TestStartServicesIfNeeded_RegistersAndReturnsCleanup(t *testing.T) {
	svcs := []config.Service{
		{Name: "start-test-svc3", Command: "sleep", Args: []string{"30"}},
	}
	t.Cleanup(func() { cleanupLogs(t, svcs) })

	var registered func()
	registrar := func(c func()) {
		registered = c
	}

	cleanup := startServicesIfNeeded(false, svcs, registrar)
	require.NotNil(t, cleanup)
	require.NotNil(t, registered, "cleanup should be registered with registrar")

	assert.NotPanics(t, func() { registered() }, "registered cleanup should stop services")
}

func TestStartServicesIfNeeded_ServiceRunningAndStops(t *testing.T) {
	svcs := []config.Service{
		{Name: "start-test-svc4", Command: "sleep", Args: []string{"30"}},
	}
	t.Cleanup(func() { cleanupLogs(t, svcs) })

	cleanup := startServicesIfNeeded(false, svcs, nil)
	require.NotNil(t, cleanup)

	time.Sleep(100 * time.Millisecond)

	cleanup()
	time.Sleep(100 * time.Millisecond)
}

func TestRenderInstructions(t *testing.T) {
	tests := []struct {
		name           string
		tmplText       string
		repo           string
		branch         string
		body           string
		pr             string
		expectedOutput string
	}{
		{
			name:           "replaces CommentBody, PRNumber, PRBranch, RepoOwner, RepoName with provided values",
			tmplText:       "Comment: {{.CommentBody}}\nPR: {{.PRNumber}}\nBranch: {{.PRBranch}}\nOwner: {{.RepoOwner}}\nRepo: {{.RepoName}}",
			repo:           "zon/ralph",
			branch:         "feature/test",
			body:           "Please review this",
			pr:             "123",
			expectedOutput: "Comment: Please review this\nPR: 123\nBranch: feature/test\nOwner: zon\nRepo: ralph",
		},
		{
			name:           "splits repo string on / to populate RepoOwner and RepoName correctly",
			tmplText:       "Owner: {{.RepoOwner}}, Repo: {{.RepoName}}",
			repo:           "myorg/my-repo",
			branch:         "main",
			body:           "",
			pr:             "1",
			expectedOutput: "Owner: myorg, Repo: my-repo",
		},
		{
			name:           "returns raw template text when template contains invalid Go template syntax",
			tmplText:       "Invalid: {{.InvalidField",
			repo:           "owner/repo",
			branch:         "main",
			body:           "test",
			pr:             "1",
			expectedOutput: "Invalid: {{.InvalidField",
		},
		{
			name:           "handles repo string without a / by leaving RepoOwner and RepoName as empty strings",
			tmplText:       "Owner: '{{.RepoOwner}}', Repo: '{{.RepoName}}'",
			repo:           "invalid-repo",
			branch:         "main",
			body:           "test",
			pr:             "1",
			expectedOutput: "Owner: '', Repo: ''",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := renderInstructions(tt.tmplText, tt.repo, tt.branch, tt.body, tt.pr)
			assert.Equal(t, tt.expectedOutput, result, "renderInstructions should return expected output")
		})
	}
}

func TestProjectFileFromBranch(t *testing.T) {
	tests := []struct {
		name     string
		branch   string
		expected string
	}{
		{
			name:     "converts ralph/my-feature to projects/my-feature.yaml",
			branch:   "ralph/my-feature",
			expected: "projects/my-feature.yaml",
		},
		{
			name:     "converts branch without ralph/ prefix by replacing slashes with dashes",
			branch:   "feature/thing",
			expected: "projects/feature-thing.yaml",
		},
		{
			name:     "handles branch name with no slashes",
			branch:   "my-feature",
			expected: "projects/my-feature.yaml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := projectFileFromBranch(tt.branch)
			assert.Equal(t, tt.expected, result, "projectFileFromBranch should return expected output")
		})
	}
}
