package context

import (
	"os"
	"strings"
)

// Context holds the execution context for ralph commands
type Context struct {
	projectFile    string
	maxIterations  int
	verbose        bool
	noNotify       bool
	noServices     bool
	local          bool
	follow         bool
	notes          []string // Runtime notes to pass to the agent
	instructions   string   // Path to an instructions file that overrides the default instructions
	instructionsMD string   // Inline instructions content; overrides .ralph/instructions.md when set
	repo           string   // owner/repo override (e.g., "zon/ralph"); skips local git remote detection
	branch         string   // Branch override; skips local git GetCurrentBranch + sync check
	debugBranch    string   // When set, workflows checkout this ralph repo branch and invoke ralph via `go run`
	baseBranch     string   // Base branch override; overrides baseBranch from .ralph/config.yaml for PR creation
}

// IsVerbose returns true if verbose logging is enabled
func (c *Context) IsVerbose() bool {
	return c.verbose
}

// ShouldNotify returns true if notifications should be sent
func (c *Context) ShouldNotify() bool {
	// Disable notifications if submitting a remote workflow without following
	if !c.local && !c.follow {
		return false
	}
	return !c.noNotify
}

// NoServices returns true if services should be skipped
func (c *Context) NoServices() bool {
	return c.noServices
}

// IsLocal returns true if running locally instead of submitting to Argo Workflows
func (c *Context) IsLocal() bool {
	return c.local
}

// ShouldFollow returns true if workflow logs should be followed after submission
func (c *Context) ShouldFollow() bool {
	return c.follow
}

// IsWorkflowExecution returns true if running inside a workflow container
// This is detected via the RALPH_WORKFLOW_EXECUTION environment variable
func (c *Context) IsWorkflowExecution() bool {
	return os.Getenv("RALPH_WORKFLOW_EXECUTION") == "true"
}

// RepoOwnerAndName returns the owner and repository name.
// It uses ctx.repo ("owner/repo") when set, otherwise falls back to the
// GITHUB_REPO_OWNER and GITHUB_REPO_NAME environment variables injected by
// the workflow container.
func (c *Context) RepoOwnerAndName() (owner, name string) {
	if c.repo != "" {
		parts := strings.SplitN(c.repo, "/", 2)
		if len(parts) == 2 {
			return parts[0], parts[1]
		}
	}
	return os.Getenv("GITHUB_REPO_OWNER"), os.Getenv("GITHUB_REPO_NAME")
}

// AddNote adds a runtime note to be passed to the agent
func (c *Context) AddNote(note string) {
	if c.notes == nil {
		c.notes = []string{}
	}
	c.notes = append(c.notes, note)
}

// HasNotes returns true if there are any notes
func (c *Context) HasNotes() bool {
	return len(c.notes) > 0
}

func (c *Context) SetVerbose(verbose bool) {
	c.verbose = verbose
}

func (c *Context) SetNoNotify(noNotify bool) {
	c.noNotify = noNotify
}

func (c *Context) SetNoServices(noServices bool) {
	c.noServices = noServices
}

func (c *Context) SetLocal(local bool) {
	c.local = local
}

func (c *Context) SetFollow(follow bool) {
	c.follow = follow
}

func (c *Context) SetProjectFile(projectFile string) {
	c.projectFile = projectFile
}

func (c *Context) SetMaxIterations(maxIterations int) {
	c.maxIterations = maxIterations
}

func (c *Context) SetInstructions(instructions string) {
	c.instructions = instructions
}

func (c *Context) SetInstructionsMD(instructionsMD string) {
	c.instructionsMD = instructionsMD
}

func (c *Context) SetRepo(repo string) {
	c.repo = repo
}

func (c *Context) SetBranch(branch string) {
	c.branch = branch
}

func (c *Context) SetDebugBranch(debugBranch string) {
	c.debugBranch = debugBranch
}

func (c *Context) SetBaseBranch(baseBranch string) {
	c.baseBranch = baseBranch
}

func (c *Context) Repo() string {
	return c.repo
}

func (c *Context) Branch() string {
	return c.branch
}

func (c *Context) DebugBranch() string {
	return c.debugBranch
}

func (c *Context) BaseBranch() string {
	return c.baseBranch
}

func (c *Context) ProjectFile() string {
	return c.projectFile
}

func (c *Context) InstructionsMD() string {
	return c.instructionsMD
}

func (c *Context) MaxIterations() int {
	return c.maxIterations
}

func (c *Context) Instructions() string {
	return c.instructions
}

func (c *Context) Notes() []string {
	return c.notes
}
