package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/mattn/go-isatty"

	"github.com/zon/ralph/internal/config"
)

// PromptCmd runs one user-supplied prompt as a single OpenCode pass. The prompt
// comes from positional text, --file, or standard input. The command has no
// project or item options.
type PromptCmd struct {
	Prompt     []string `arg:"" optional:"" help:"Prompt text, joined with single spaces"`
	File       string   `help:"Read the prompt from a file, or - for standard input" name:"file" optional:""`
	Mode       string   `help:"Execution mode: local, worktree, or remote (default: local)" name:"mode" optional:""`
	Base       string   `help:"Override the base branch for PR creation (default: detects from current branch)" name:"base" optional:"" short:"B"`
	Model      string   `help:"The model to use in format of provider/model" name:"model" optional:""`
	Variant    string   `help:"The model variant (provider-specific reasoning effort, e.g., high, max, minimal)" name:"variant" optional:""`
	Agent      string   `help:"Override the opencode agent from config" name:"agent" optional:""`
	Context    string   `help:"The name of the Kubernetes context to use" name:"context" optional:""`
	Namespace  string   `help:"The name of the Kubernetes namespace to use" name:"namespace" short:"n" optional:""`
	NoServices bool     `help:"Skip service startup" default:"false"`
	NoFollow   bool     `help:"Do not follow workflow logs after submission (only applicable with --mode remote)" default:"false"`
	NoNotify   bool     `help:"Disable desktop notifications" default:"false"`
	Debug      string   `help:"Checkout the given Ralph repo branch in the workflow container and invoke Ralph via 'go run' instead of the built binary (only applicable with --mode remote)" name:"debug" optional:""`
	Verbose    bool     `help:"Enable verbose logging" default:"false"`

	stdin           io.Reader     `kong:"-"`
	stdinIsTerminal func() bool   `kong:"-"`
	config          config.Loader `kong:"-"`
}

// Validate rejects the conflicting prompt sources during command-line parsing.
// The remaining prompt checks need standard input or file access and run in
// Run before any mode work starts.
func (c *PromptCmd) Validate() error {
	if len(c.Prompt) > 0 && c.File != "" {
		return errors.New("cannot use positional prompt text with --file")
	}
	return nil
}

// Run validates the prompt source and the flags that depend on the resolved
// mode. The prompt flow itself is built on top of this resolution.
func (c *PromptCmd) Run() error {
	if _, err := c.readPrompt(); err != nil {
		return err
	}
	mode, err := c.resolveMode()
	if err != nil {
		return err
	}
	return validatePromptDebug(c.Debug, mode)
}

// readPrompt reads the prompt from the source selected by the flags. Tests
// inject standard input and terminal detection through the command fields.
func (c *PromptCmd) readPrompt() (string, error) {
	stdin := c.stdin
	if stdin == nil {
		stdin = os.Stdin
	}
	isTerminal := c.stdinIsTerminal
	if isTerminal == nil {
		isTerminal = stdinIsTerminal
	}
	return ReadPrompt(c.Prompt, c.File, stdin, isTerminal())
}

// resolveMode resolves the execution mode from the --mode flag, then the
// config mode, then local.
func (c *PromptCmd) resolveMode() (string, error) {
	loader := c.config
	if loader == nil {
		loader = &config.Client{}
	}
	cfg, err := loader.Load()
	if err != nil {
		return "", err
	}
	return cfg.ResolveMode(c.Mode)
}

// ReadPrompt returns the prompt from exactly one source: positional text
// joined with single spaces, a file, or standard input. Standard input is used
// when neither positional text nor --file is given and it is not a terminal,
// and also when --file is "-" regardless of the terminal. The prompt is
// returned as written; only its emptiness is checked after trimming space.
func ReadPrompt(text []string, file string, stdin io.Reader, terminal bool) (string, error) {
	if len(text) > 0 && file != "" {
		return "", errors.New("cannot use positional prompt text with --file")
	}
	if stdin == nil {
		stdin = os.Stdin
	}

	var prompt string
	switch {
	case file == "-":
		data, err := io.ReadAll(stdin)
		if err != nil {
			return "", fmt.Errorf("failed to read prompt from standard input: %w", err)
		}
		prompt = string(data)
	case file != "":
		data, err := os.ReadFile(file)
		if err != nil {
			return "", fmt.Errorf("failed to read prompt file %q: %w", file, err)
		}
		prompt = string(data)
	case len(text) > 0:
		prompt = strings.Join(text, " ")
	case !terminal:
		data, err := io.ReadAll(stdin)
		if err != nil {
			return "", fmt.Errorf("failed to read prompt from standard input: %w", err)
		}
		prompt = string(data)
	default:
		return "", errors.New("prompt is required: provide positional text, --file, or standard input")
	}

	if strings.TrimSpace(prompt) == "" {
		return "", errors.New("prompt is empty")
	}
	return prompt, nil
}

// validatePromptDebug rejects --debug outside remote mode, where there is no
// workflow container to run from a source branch.
func validatePromptDebug(debug, mode string) error {
	if debug != "" && (mode == config.ModeLocal || mode == config.ModeWorktree) {
		return fmt.Errorf("--debug flag is not applicable with --mode %s", mode)
	}
	return nil
}

// stdinIsTerminal reports whether standard input is an interactive terminal.
func stdinIsTerminal() bool {
	fd := os.Stdin.Fd()
	return isatty.IsTerminal(fd) || isatty.IsCygwinTerminal(fd)
}
