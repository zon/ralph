package project

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zon/ralph/internal/config"
)

func TestWithItems_HoldsExactlyNItems(t *testing.T) {
	t.Run("empty array for zero", func(t *testing.T) {
		proj := WithItems(0)
		assert.NotNil(t, proj)
		assert.Len(t, proj.Items, 0)
	})
	t.Run("exactly n items with 0-based indices", func(t *testing.T) {
		proj := WithItems(5)
		require.Len(t, proj.Items, 5)
		for i, it := range proj.Items {
			assert.Equal(t, i, it.Index)
		}
	})
}

func TestAny_ValidDefaultState(t *testing.T) {
	proj := Any()
	assert.NotNil(t, proj)
	assert.Equal(t, "test-project", proj.Slug)
	assert.Equal(t, "Test Project", proj.Title)
	assert.NotEmpty(t, proj.Items)
}

func TestThatReportsAllComplete(t *testing.T) {
	client := ThatReportsAllComplete()
	incomplete, err := client.Incomplete(WithItems(3), "main")
	require.NoError(t, err)
	assert.Empty(t, incomplete)
}

func TestThatReportsIncompleteUntil(t *testing.T) {
	client := ThatReportsIncompleteUntil(2)
	proj := WithItems(3)
	for call := 1; call <= 2; call++ {
		incomplete, err := client.Incomplete(proj, "main")
		require.NoError(t, err)
		assert.NotEmpty(t, incomplete, "call %d should report items incomplete", call)
	}
	incomplete, err := client.Incomplete(proj, "main")
	require.NoError(t, err)
	assert.Empty(t, incomplete, "calls past n report every item complete")
}

func TestThatAlwaysReportsIncomplete(t *testing.T) {
	client := ThatAlwaysReportsIncomplete()
	for call := 1; call <= 3; call++ {
		incomplete, err := client.Incomplete(WithItems(3), "main")
		require.NoError(t, err)
		assert.Len(t, incomplete, 3, "call %d should always report items incomplete", call)
	}
}

func TestThatReportsComplete(t *testing.T) {
	client := ThatReportsComplete(0, 2)
	proj := WithItems(4)
	complete, err := client.Complete(proj, "main")
	require.NoError(t, err)
	assert.Equal(t, []int{0, 2}, complete)
	incomplete, err := client.Incomplete(proj, "main")
	require.NoError(t, err)
	assert.Equal(t, []int{1, 3}, []int{incomplete[0].Index, incomplete[1].Index})
}

func TestThatReportsCompleteThenAllComplete(t *testing.T) {
	client := ThatReportsComplete(0).ThenAllComplete()
	proj := WithItems(3)
	first, err := client.Incomplete(proj, "main")
	require.NoError(t, err)
	require.Len(t, first, 2)
	assert.Equal(t, []int{1, 2}, []int{first[0].Index, first[1].Index})
	second, err := client.Incomplete(proj, "main")
	require.NoError(t, err)
	assert.Empty(t, second, "later calls report every item complete")
}

func TestThatFailsResolution(t *testing.T) {
	client := ThatFailsResolution()
	proj, err := client.Resolve("/tmp/project.yaml", ".")
	require.Nil(t, proj)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "item query yielded no items")
}

func TestMockProjectRecordsCalls(t *testing.T) {
	t.Run("last item query and resolve count", func(t *testing.T) {
		client := ThatReportsAllComplete()
		_, err := client.Resolve("/tmp/project.yaml", ".requirements")
		require.NoError(t, err)
		_, err = client.Resolve("/tmp/project.yaml", ".tasks")
		require.NoError(t, err)
		assert.Equal(t, ".tasks", client.LastQuery())
		assert.Equal(t, 2, client.ResolveCount())
	})
	t.Run("last base branch and incomplete call count", func(t *testing.T) {
		client := ThatReportsAllComplete()
		proj := WithItems(3)
		_, err := client.Incomplete(proj, "develop")
		require.NoError(t, err)
		_, err = client.Incomplete(proj, "main")
		require.NoError(t, err)
		assert.Equal(t, "main", client.LastBase())
		assert.Equal(t, 2, client.IncompleteCallCount())
	})
	t.Run("written records a write to the project file", func(t *testing.T) {
		client := ThatReportsAllComplete()
		assert.False(t, client.Written())
		client.Write(WithItems(2))
		assert.True(t, client.Written())
	})
	t.Run("removed records project file removal", func(t *testing.T) {
		client := ThatReportsAllComplete()
		assert.False(t, client.Removed())
		require.NoError(t, client.Remove(WithItems(2)))
		assert.True(t, client.Removed())
	})
	t.Run("base branch recorded by Complete", func(t *testing.T) {
		client := ThatReportsComplete(1)
		_, err := client.Complete(WithItems(3), "base-branch")
		require.NoError(t, err)
		assert.Equal(t, "base-branch", client.LastBase())
	})
}

func TestMockProjectIncompleteError(t *testing.T) {
	t.Run("nils when nothing is incomplete", func(t *testing.T) {
		client := ThatReportsAllComplete()
		assert.NoError(t, client.IncompleteError(WithItems(3), "main"))
	})
	t.Run("names the still incomplete items", func(t *testing.T) {
		client := ThatAlwaysReportsIncomplete()
		err := client.IncompleteError(WithItems(3), "main")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "item 0")
		assert.Contains(t, err.Error(), "item 1")
		assert.Contains(t, err.Error(), "item 2")
		assert.Contains(t, err.Error(), "incomplete")
	})
}

func TestMockProjectExtraIterations(t *testing.T) {
	proj := WithItems(10)
	client := ThatAlwaysReportsIncomplete()
	assert.Equal(t, 2, client.ExtraIterations(proj, &config.RalphConfig{}))
	v := 5
	assert.Equal(t, 5, client.ExtraIterations(proj, &config.RalphConfig{ExtraIterations: &v}))
}

func TestMockProjectOrchestrationFlow(t *testing.T) {
	t.Run("has spec and orchestration modifiers", func(t *testing.T) {
		assert.False(t, ThatReportsAllComplete().WithNoSpec().HasSpec(nil))
		assert.True(t, ThatReportsAllComplete().WithSpecButNoOrchestration().HasSpec(nil))
		assert.False(t, ThatReportsAllComplete().WithSpecButNoOrchestration().HasOrchestration(nil))
		assert.True(t, ThatReportsAllComplete().WithOrchestration().HasSpec(nil))
		assert.True(t, ThatReportsAllComplete().WithOrchestration().HasOrchestration(nil))
	})
	t.Run("remove orchestration records and can fail", func(t *testing.T) {
		client := ThatReportsAllComplete().WithOrchestration()
		require.NoError(t, client.RemoveOrchestration(nil))
		assert.True(t, client.OrchestrationRemoved())
		failing := ThatReportsAllComplete().WithOrchestration().ThatFailsRemoval()
		require.Error(t, failing.RemoveOrchestration(nil))
		assert.True(t, failing.OrchestrationRemoved())
	})
	t.Run("project removal can fail", func(t *testing.T) {
		client := ThatReportsAllComplete().ThatFailsProjectRemoval()
		require.Error(t, client.Remove(nil))
		assert.True(t, client.Removed())
	})
}

func TestInputFactories(t *testing.T) {
	t.Run("wraps a project as an input file", func(t *testing.T) {
		p := WithItems(2)
		f := ForProjectInput(p)
		assert.True(t, f.IsProject())
		assert.False(t, f.IsSpec())
		assert.False(t, f.IsOrchestration())
		assert.Equal(t, p, f.Project())
	})
	t.Run("wraps an orchestration document as an input file", func(t *testing.T) {
		f := ForOrchestrationInput("/tmp/orchestration.md")
		assert.True(t, f.IsOrchestration())
		assert.False(t, f.IsProject())
		assert.False(t, f.IsSpec())
	})
	t.Run("wraps a spec document as an input file", func(t *testing.T) {
		f := ForSpecInput("/tmp/spec.md")
		assert.True(t, f.IsSpec())
		assert.False(t, f.IsProject())
		assert.False(t, f.IsOrchestration())
	})
}
