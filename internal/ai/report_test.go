package ai

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestReportIsNothingToDo covers the exact-match semantics of
// Report.IsNothingToDo. Surrounding whitespace is ignored, but the content
// must otherwise equal the NOTHING_TO_DO marker exactly.
func TestReportIsNothingToDo(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    bool
	}{
		{name: "exact nothing to do", content: "NOTHING_TO_DO", want: true},
		{name: "whitespace around nothing to do", content: "  NOTHING_TO_DO  \n", want: true},
		{name: "work was done", content: "did the work", want: false},
		{name: "marker must match exactly", content: "nothing_to_do", want: false},
		{name: "marker followed by more content", content: "NOTHING_TO_DO\n\nmore", want: false},
		{name: "marker with trailing note", content: "NOTHING_TO_DO - done", want: false},
		{name: "empty report", content: "", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			report := Report{Content: tt.content}
			assert.Equal(t, tt.want, report.IsNothingToDo())
		})
	}
}

// TestReadReport covers reading the agent's report.md from the working
// directory. The report content is returned exactly, and the missing-file case
// yields an error naming report.md.
func TestReadReport(t *testing.T) {
	t.Run("returns report.md content exactly", func(t *testing.T) {
		t.Chdir(t.TempDir())
		content := "did the work\n"
		require.NoError(t, os.WriteFile("report.md", []byte(content), 0644))

		report, err := ReadReport()
		require.NoError(t, err)
		assert.Equal(t, content, report.Content)
		assert.False(t, report.IsNothingToDo(), "the written content is not the nothing-to-do marker")
	})

	t.Run("missing report.md returns error naming the file", func(t *testing.T) {
		t.Chdir(t.TempDir())

		_, err := ReadReport()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "report.md")
	})
}
