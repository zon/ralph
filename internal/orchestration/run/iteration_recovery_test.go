package run

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/project"
	"github.com/zon/ralph/internal/services"
)

// TestIterateNonFatalPickFailureCarriesErrorIntoNextPrompts asserts a
// non-fatal picker failure does not stop the run: the loop continues and the
// next iteration's picker and development prompts both carry the failure.
func TestIterateNonFatalPickFailureCarriesErrorIntoNextPrompts(t *testing.T) {
	picks := 0
	ai := &mockAI{
		runPickerFunc: func(_ *project.Project, incomplete []project.Item, _ error) (project.Item, error) {
			picks++
			if picks == 1 {
				return project.Item{}, errNonFatal
			}
			return incomplete[0], nil
		},
	}
	runner := withMocks(
		withProject(project.ThatReportsIncompleteUntil(2).WithResolvedItems(3)),
		withAI(ai),
	)

	err := runner.RunLocal(project.ForProjectInput(project.WithItems(3)), config.WithExtraIterations(0))
	require.NoError(t, err)
	require.Equal(t, []error{nil, errNonFatal}, ai.pickerPreviousErrs)
	require.Equal(t, []error{errNonFatal}, ai.developPreviousErrs)
}

// TestIterateNonFatalDevelopFailureCarriesErrorIntoNextPrompts asserts a
// non-fatal developer failure is recorded and carried into the next
// iteration's picker and development prompts.
func TestIterateNonFatalDevelopFailureCarriesErrorIntoNextPrompts(t *testing.T) {
	develops := 0
	ai := &mockAI{
		runDeveloperFunc: func(_ *project.Project, _ project.Item, _ error) error {
			develops++
			if develops == 1 {
				return errNonFatal
			}
			return nil
		},
	}
	runner := withMocks(
		withProject(project.ThatReportsIncompleteUntil(2).WithResolvedItems(3)),
		withAI(ai),
	)

	err := runner.RunLocal(project.ForProjectInput(project.WithItems(3)), config.WithExtraIterations(0))
	require.NoError(t, err)
	require.Equal(t, []error{nil, errNonFatal}, ai.pickerPreviousErrs)
	require.Equal(t, []error{nil, errNonFatal}, ai.developPreviousErrs)
}

// TestIterateSuccessfulIterationClearsPreviousError asserts a successful
// iteration clears the recorded failure, so a later prompt carries none.
func TestIterateSuccessfulIterationClearsPreviousError(t *testing.T) {
	picks := 0
	ai := &mockAI{
		runPickerFunc: func(_ *project.Project, incomplete []project.Item, _ error) (project.Item, error) {
			picks++
			if picks == 1 {
				return project.Item{}, errNonFatal
			}
			return incomplete[0], nil
		},
	}
	runner := withMocks(
		withProject(project.ThatReportsIncompleteUntil(3).WithResolvedItems(3)),
		withAI(ai),
	)

	err := runner.RunLocal(project.ForProjectInput(project.WithItems(3)), config.WithExtraIterations(0))
	require.NoError(t, err)
	require.Equal(t, []error{nil, errNonFatal, nil}, ai.pickerPreviousErrs)
}

// TestIterateCarriesOnlyMostRecentFailure asserts that when consecutive
// iterations fail, the next prompt carries only the immediately preceding
// failure.
func TestIterateCarriesOnlyMostRecentFailure(t *testing.T) {
	picks := 0
	ai := &mockAI{
		runPickerFunc: func(_ *project.Project, incomplete []project.Item, _ error) (project.Item, error) {
			picks++
			switch picks {
			case 1:
				return project.Item{}, errNonFatal
			case 2:
				return project.Item{}, errNonFatalOther
			default:
				return incomplete[0], nil
			}
		},
	}
	runner := withMocks(
		withProject(project.ThatReportsIncompleteUntil(3).WithResolvedItems(3)),
		withAI(ai),
	)

	err := runner.RunLocal(project.ForProjectInput(project.WithItems(3)), config.WithExtraIterations(1))
	require.NoError(t, err)
	require.Equal(t, []error{nil, errNonFatal, errNonFatalOther}, ai.pickerPreviousErrs)
}

// TestIterateServiceFixFailureCarriesErrorIntoNextPicker asserts a non-fatal
// service-startup fix failure is recorded and carried into the next
// iteration's picker prompt instead of stopping the run.
func TestIterateServiceFixFailureCarriesErrorIntoNextPicker(t *testing.T) {
	svc := &mockServices{}
	svc.startFunc = func() (*services.Manager, error) {
		if svc.startCount == 1 {
			return nil, errNonFatal
		}
		return &services.Manager{}, nil
	}
	ai := &mockAI{
		fixServiceFunc: func(*config.RalphConfig, error) error { return errNonFatal },
	}
	runner := withMocks(
		withProject(project.ThatReportsIncompleteUntil(2).WithResolvedItems(3)),
		withServices(svc),
		withAI(ai),
	)

	err := runner.RunLocal(project.ForProjectInput(project.WithItems(3)), config.WithExtraIterations(0))
	require.NoError(t, err)
	require.True(t, aiServiceFixCalled(runner))
	require.Equal(t, []error{errNonFatal}, ai.pickerPreviousErrs)
}

// TestIterateNonFatalFailureConsumesAnIteration asserts a failed iteration
// still counts against the loop's cap: a prompt that always fails non-fatally
// runs the cap's worth of AI passes and then ends.
func TestIterateNonFatalFailureConsumesAnIteration(t *testing.T) {
	runner := withMocks(
		withProject(project.ThatAlwaysReportsIncomplete().WithResolvedItems(3)),
		withAI(aiThatAlwaysFails()),
	)

	err := runner.RunLocal(project.ForProjectInput(project.WithItems(3)), config.WithExtraIterations(0))
	require.Error(t, err)
	require.Equal(t, 3, aiPickCalls(runner))
}

// TestIterateDoesNotCommitFailedIteration asserts a non-fatal failure is not
// committed: the loop carries on but the failed iteration leaves no commit.
func TestIterateDoesNotCommitFailedIteration(t *testing.T) {
	picks := 0
	ai := &mockAI{
		runPickerFunc: func(_ *project.Project, incomplete []project.Item, _ error) (project.Item, error) {
			picks++
			if picks == 1 {
				return project.Item{}, errNonFatal
			}
			return incomplete[0], nil
		},
	}
	git := gitWithChangesAndReport()
	runner := withMocks(
		withProject(project.ThatReportsIncompleteUntil(2).WithResolvedItems(3)),
		withGit(git),
		withAI(ai),
	)

	err := runner.RunLocal(project.ForProjectInput(project.WithItems(3)), config.WithExtraIterations(0))
	require.NoError(t, err)
	require.Equal(t, 1, git.commitFromReportCalls, "only the successful iteration is committed")
}
