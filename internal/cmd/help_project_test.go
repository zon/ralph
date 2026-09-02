package cmd

import (
	"bytes"
	"testing"

	"github.com/alecthomas/kong"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zon/ralph/internal/projectfile"
)

func TestHelpProjectSubcommandParsed(t *testing.T) {
	cmd := &Cmd{}
	parser, err := kong.New(cmd,
		kong.Name("ralph"),
		kong.Exit(func(int) {}),
	)
	require.NoError(t, err)

	_, err = parser.Parse([]string{"help", "project"})
	require.NoError(t, err)
}

func TestHelpProjectCmdHelpText(t *testing.T) {
	output := captureHelpOutput(&Cmd{}, []string{"help", "project", "--help"})
	assert.Contains(t, output, "Display the project file guide")
}

// TestPrintProjectDocumentationNonInteractive asserts redirected output is the
// guide rendered as plain text with no ANSI escapes.
func TestPrintProjectDocumentationNonInteractive(t *testing.T) {
	var buf bytes.Buffer
	err := printDocumentation(&buf, "project", projectfile.ProjectDocumentation())
	require.NoError(t, err)

	rendered := buf.String()
	assert.NotContains(t, rendered, "\x1b[", "redirected output must not carry ANSI escapes")
	assert.Contains(t, rendered, "Project Files")
	assert.Contains(t, rendered, "ralph projects/report-export.yaml")
}

// TestProjectDocumentationEmbedded asserts the embedded guide covers the
// sections a reader of `ralph help project` expects.
func TestProjectDocumentationEmbedded(t *testing.T) {
	doc := projectfile.ProjectDocumentation()
	require.NotEmpty(t, doc)
	for _, section := range []string{"# Project Files", "## A Simple Project", "## What Ralph Does", "## Check the Project"} {
		assert.Contains(t, doc, section)
	}
	assert.Contains(t, doc, "ralph validate", "the guide must describe how to check a project")
}

// TestProjectPagerRendersProjectGuide exercises the Bubble Tea pager with the
// project guide: resizing fills the viewport and the content is visible.
func TestProjectPagerRendersProjectGuide(t *testing.T) {
	p := newDocumentationPager(projectfile.ProjectDocumentation())
	_, _ = p.Update(tea.WindowSizeMsg{Width: 100, Height: 20})
	require.True(t, p.ready)
	assert.Contains(t, p.viewport.View(), "Project")
}
