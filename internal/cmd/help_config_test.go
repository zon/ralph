package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/alecthomas/kong"
	tea "github.com/charmbracelet/bubbletea"
	styles "github.com/charmbracelet/glamour/styles"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zon/ralph/internal/config"
)

func TestHelpConfigSubcommandParsed(t *testing.T) {
	cmd := &Cmd{}
	parser, err := kong.New(cmd,
		kong.Name("ralph"),
		kong.Exit(func(int) {}),
	)
	require.NoError(t, err)

	_, err = parser.Parse([]string{"help", "config"})
	require.NoError(t, err)
}

func TestHelpConfigCmdHelpText(t *testing.T) {
	output := captureHelpOutput(&Cmd{}, []string{"help", "config", "--help"})
	assert.Contains(t, output, "Display the configuration reference")
}

// TestPrintConfigDocumentationNonInteractive asserts redirected output is the
// documentation rendered as plain text with no ANSI escapes.
func TestPrintConfigDocumentationNonInteractive(t *testing.T) {
	var buf bytes.Buffer
	err := printDocumentation(&buf, "configuration", config.ConfigDocumentation())
	require.NoError(t, err)

	rendered := buf.String()
	assert.NotContains(t, rendered, "\x1b[", "redirected output must not carry ANSI escapes")
	assert.Contains(t, rendered, "Configuration")
	assert.Contains(t, rendered, "worktree")
}

// TestConfigDocumentationEmbedded asserts the embedded documentation covers
// the configuration sections a reader of `ralph help config` expects.
func TestConfigDocumentationEmbedded(t *testing.T) {
	doc := config.ConfigDocumentation()
	require.NotEmpty(t, doc)
	for _, section := range []string{"# Configuration", "## Mode", "## Items", "## Iterations", "## Loops", "## Before", "## Services", "## Workflow", "## Custom Instructions"} {
		assert.Contains(t, doc, section)
	}
	assert.Contains(t, doc, "<branch>-<hash>", "the documentation must describe the completion trailer")
}

// TestConfigPagerScrollsAndQuits exercises the Bubble Tea pager model without
// a terminal: resizing fills the viewport, scrolling moves it, and the quit
// keys stop the program.
func TestConfigPagerScrollsAndQuits(t *testing.T) {
	p := newDocumentationPager(config.ConfigDocumentation())
	_, _ = p.Update(tea.WindowSizeMsg{Width: 100, Height: 20})
	require.True(t, p.ready)
	assert.Contains(t, p.viewport.View(), "Configuration")

	before := p.viewport.YOffset
	_, cmd := p.Update(tea.KeyMsg(tea.Key{Type: tea.KeyDown}))
	assert.Greater(t, p.viewport.YOffset, before, "down must scroll the viewport")
	assert.Nil(t, cmd, "scrolling must not stop the pager")

	for _, quitKey := range []tea.KeyMsg{
		tea.KeyMsg(tea.Key{Type: tea.KeyCtrlC}),
		tea.KeyMsg(tea.Key{Type: tea.KeyRunes, Runes: []rune("q")}),
		tea.KeyMsg(tea.Key{Type: tea.KeyEsc}),
	} {
		p := newDocumentationPager(config.ConfigDocumentation())
		_, _ = p.Update(tea.WindowSizeMsg{Width: 100, Height: 20})
		_, cmd := p.Update(quitKey)
		assert.NotNil(t, cmd, "%q must stop the pager", quitKey.String())
	}
}

// TestConfigPagerDetectsStyleAsynchronously asserts the pager paints with the
// dark style immediately and only swaps to the light style when the
// asynchronous probe reports a light terminal, never blocking first paint on
// the probe.
func TestConfigPagerDetectsStyleAsynchronously(t *testing.T) {
	p := newDocumentationPager(config.ConfigDocumentation())
	assert.Equal(t, styles.DarkStyle, p.style, "the pager must default to the dark style")
	assert.NotNil(t, p.Init(), "Init must run the background style probe")

	_, _ = p.Update(tea.WindowSizeMsg{Width: 100, Height: 20})
	_, _ = p.Update(documentationStyleMsg{dark: false})
	assert.Equal(t, styles.LightStyle, p.style, "a light terminal must flip the style to light")
	assert.Contains(t, p.viewport.View(), "Configuration")

	_, _ = p.Update(documentationStyleMsg{dark: true})
	assert.Equal(t, styles.LightStyle, p.style, "a dark terminal must not flip the style")
}

func TestConfigPagerFooterFitsHeight(t *testing.T) {
	p := newDocumentationPager(config.ConfigDocumentation())
	_, _ = p.Update(tea.WindowSizeMsg{Width: 100, Height: 20})
	require.True(t, p.ready)
	lines := strings.Split(p.View(), "\n")
	assert.Len(t, lines, 20, "the pager must fill exactly one terminal height, footer included")
	assert.Contains(t, lines[len(lines)-1], "q: quit")
}
