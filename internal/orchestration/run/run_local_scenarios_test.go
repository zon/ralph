package run

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/zon/ralph/internal/project"
)

func TestProjectClientInterfaceSatisfiedByRealAndMockClients(t *testing.T) {
	var _ ProjectClient = (*project.Client)(nil)
	var _ ProjectClient = (*project.MockProject)(nil)
}

func TestProjectClientInterfaceDoesNotExposeLegacyMethods(t *testing.T) {
	typ := reflect.TypeOf((*ProjectClient)(nil)).Elem()
	for _, name := range []string{"Reload", "AllRequirementsPassing", "HasChanges", "NormalizeAndStage", "ExtraIterationsError"} {
		_, ok := typ.MethodByName(name)
		require.False(t, ok, "ProjectClient must not expose %s", name)
	}
}

// TestProjectClientInterfaceDoesNotExposeCompletedProjectScanning asserts the
// run orchestration's project client surface exposes no way to load every
// project file in a directory, scan a directory for fully-complete project
// files, or remove a set of project files and commit the deletion. Those
// helpers existed only for the removed merge path, so a run can neither call
// them nor receive them from a real or mock client.
func TestProjectClientInterfaceDoesNotExposeCompletedProjectScanning(t *testing.T) {
	typ := reflect.TypeOf((*ProjectClient)(nil)).Elem()
	for _, name := range []string{
		"LoadAll",              // loading every project file in a directory
		"FilterPassing",        // scanning a directory for fully-complete project files
		"FindCompleteProjects", // scanning a directory for fully-complete project files
		"DeleteAll",            // removing a set of project files
		"RemoveAndCommit",      // removing a set and committing the deletion
	} {
		_, ok := typ.MethodByName(name)
		require.False(t, ok, "ProjectClient must not expose %s", name)
	}
}

// TestProjectClientInterfaceKeepsSingleProjectFileRemoval asserts the run
// orchestration's project client surface still removes a single completed
// project file: deleting one project file is the run loop's own cleanup, and is
// not part of the removed merge path.
func TestProjectClientInterfaceKeepsSingleProjectFileRemoval(t *testing.T) {
	typ := reflect.TypeOf((*ProjectClient)(nil)).Elem()
	_, ok := typ.MethodByName("Remove")
	require.True(t, ok, "ProjectClient must keep the single-file Remove the run loop uses for cleanup")
}
