package cmd

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	styles "github.com/charmbracelet/glamour/styles"
	"github.com/mattn/go-isatty"
	"github.com/muesli/termenv"

	"github.com/zon/ralph/internal/config"
)

// ConfigCmd displays the .ralph/config.yaml reference documentation.
type ConfigCmd struct{}

// Run prints the configuration documentation. When the process runs on an
// interactive terminal, glamour renders the markdown and Bubble Tea pages it;
// otherwise the markdown is rendered as plain text.
func (c *ConfigCmd) Run() error {
	return printConfigDocumentation(os.Stdout, interactiveTerminal())
}

// defaultConfigDocWidth is the word-wrap width used when rendering the
// documentation outside an interactive terminal.
const defaultConfigDocWidth = 80

func printConfigDocumentation(out io.Writer, interactive bool) error {
	markdown := config.ConfigDocumentation()
	if !interactive {
		return writeRenderedConfig(markdown, defaultConfigDocWidth, styles.NoTTYStyle, out)
	}

	p := tea.NewProgram(newConfigPager(markdown), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("failed to display configuration documentation: %w", err)
	}
	return nil
}

// interactiveTerminal reports whether the process runs on an interactive
// terminal: stdin and stdout are terminals and TERM is not dumb.
func interactiveTerminal() bool {
	if os.Getenv("TERM") == "dumb" {
		return false
	}
	return isatty.IsTerminal(os.Stdin.Fd()) && isatty.IsTerminal(os.Stdout.Fd())
}

// renderConfig renders markdown through glamour wrapped to width using the
// named standard style.
func renderConfig(markdown string, width int, style string) (string, error) {
	renderer, err := glamour.NewTermRenderer(
		glamour.WithWordWrap(width),
		glamour.WithStandardStyle(style),
	)
	if err != nil {
		return "", fmt.Errorf("failed to create markdown renderer: %w", err)
	}
	rendered, err := renderer.Render(markdown)
	if err != nil {
		return "", fmt.Errorf("failed to render configuration documentation: %w", err)
	}
	return rendered, nil
}

func writeRenderedConfig(markdown string, width int, style string, out io.Writer) error {
	rendered, err := renderConfig(markdown, width, style)
	if err != nil {
		return err
	}
	_, err = io.WriteString(out, trimTrailingWhitespace(rendered))
	return err
}

// trimTrailingWhitespace drops the padding glamour appends to each line so
// output redirected to a file or pipe stays clean.
func trimTrailingWhitespace(s string) string {
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " \t")
	}
	return strings.Join(lines, "\n")
}

// configPager pages the glamour-rendered configuration documentation in a
// viewport.
type configPager struct {
	viewport viewport.Model
	markdown string
	style    string
	width    int
	ready    bool
}

// configStyleMsg reports whether the terminal uses a dark background. It is
// sent asynchronously so the pager can paint immediately with the dark style
// instead of blocking on termenv's terminal query.
type configStyleMsg struct {
	dark bool
}

func newConfigPager(markdown string) *configPager {
	return &configPager{markdown: markdown, style: styles.DarkStyle}
}

func (m *configPager) Init() tea.Cmd {
	return detectConfigStyle
}

// detectConfigStyle runs termenv's background color query off the render path.
// Terminals that never answer keep the query running until its timeout and the
// pager stays on the dark style.
func detectConfigStyle() tea.Msg {
	return configStyleMsg{dark: termenv.HasDarkBackground()}
}

func (m *configPager) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		if msg.Width <= 0 || msg.Height <= 0 {
			return m, nil
		}
		if !m.ready {
			// The footer occupies the last row.
			m.viewport = viewport.New(msg.Width, msg.Height-1)
			m.ready = true
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - 1
		}
		m.width = msg.Width
		if err := m.refresh(); err != nil {
			return m, tea.Quit
		}
		return m, nil

	case configStyleMsg:
		if !msg.dark && m.style != styles.LightStyle {
			m.style = styles.LightStyle
			if m.ready {
				if err := m.refresh(); err != nil {
					return m, tea.Quit
				}
			}
		}
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit
		}
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		return m, cmd
	}
	return m, nil
}

// refresh re-renders the markdown at the current width and style.
func (m *configPager) refresh() error {
	content, err := renderConfig(m.markdown, m.width, m.style)
	if err != nil {
		m.viewport.SetContent(err.Error())
		return err
	}
	m.viewport.SetContent(content)
	return nil
}

func (m *configPager) View() string {
	if !m.ready {
		return ""
	}
	return m.viewport.View() + "\n" + m.footer()
}

func (m *configPager) footer() string {
	help := []rune("↑/↓ or j/k: scroll · space/pgdn: page down · pgup: page up · q: quit")
	if m.width > 0 && len(help) > m.width {
		help = help[:m.width]
	}
	return string(help)
}
