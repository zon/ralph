package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/zon/ralph/internal/config"
	execcontext "github.com/zon/ralph/internal/context"
	"github.com/zon/ralph/internal/github"
	"github.com/zon/ralph/internal/project"
)

const (
	DefaultSecretsDir         = "/secrets/github"
	DefaultOpenCodeSecretsDir = "/secrets/opencode"
)

type WorkflowCmd struct {
	RepoURL        string `help:"Git repository URL to clone" optional:""`
	Branch         string `help:"Branch to clone or create" optional:""`
	ProjectPath    string `help:"Path to project YAML file within the repository" optional:""`
	BaseBranch     string `help:"Base branch for PR creation (default: detected dynamically)" name:"base" short:"B"`
	BotName        string `help:"Git user name for commits" default:"ralph[bot]"`
	BotEmail       string `help:"Git user email for commits" default:"ralph[bot]@users.noreply.github.com"`
	DebugBranch    string `help:"Ralph branch to use for debug mode (clones ralph from this branch and runs via go run)" name:"debug" optional:""`
	Verbose        bool   `help:"Enable verbose logging" default:"false"`
	NoServices     bool   `help:"Skip service startup" default:"false"`
	Local          bool   `help:"Run locally instead of in workflow container" default:"false"`
	InstructionsMD string `help:"Inline instructions content" name:"instructions-md" optional:""`
	MaxIterations  int    `help:"Maximum number of iterations" name:"max-iterations" default:"0"`

	cleanupRegistrar func(func()) `kong:"-"`
}

func (w *WorkflowCmd) Run() error {
	w.resolveFromEnvVars()

	if w.RepoURL == "" || w.Branch == "" || w.ProjectPath == "" {
		return fmt.Errorf("repo URL, branch, and project path are required (use flags or set environment variables)")
	}

	return w.runLocally()
}

func (w *WorkflowCmd) resolveFromEnvVars() {
	if w.RepoURL == "" {
		w.RepoURL = os.Getenv("GIT_REPO_URL")
	}
	if w.Branch == "" {
		w.Branch = os.Getenv("PROJECT_BRANCH")
	}
	if w.BaseBranch == "" {
		w.BaseBranch = os.Getenv("BASE_BRANCH")
	}
	if w.ProjectPath == "" {
		w.ProjectPath = os.Getenv("PROJECT_PATH")
	}
	if w.DebugBranch == "" {
		w.DebugBranch = os.Getenv("RALPH_DEBUG_BRANCH")
	}
	if !w.Verbose {
		w.Verbose = os.Getenv("RALPH_VERBOSE") == "true"
	}
	if !w.NoServices {
		w.NoServices = os.Getenv("RALPH_NO_SERVICES") == "true"
	}
	if w.InstructionsMD == "" {
		w.InstructionsMD = os.Getenv("INSTRUCTIONS_MD")
	}
	if w.MaxIterations == 0 {
		if val := os.Getenv("RALPH_MAX_ITERATIONS"); val != "" {
			fmt.Sscanf(val, "%d", &w.MaxIterations)
		}
	}
}

func (w *WorkflowCmd) runLocally() error {
	fmt.Println("Running workflow in local mode...")

	if err := w.setupGitHubAuth(); err != nil {
		return fmt.Errorf("failed to setup GitHub auth: %w", err)
	}

	if err := w.setupOpenCodeCredentials(); err != nil {
		return fmt.Errorf("failed to setup OpenCode credentials: %w", err)
	}

	w.configureGitUser()

	if err := w.cloneAndSetupRepo(); err != nil {
		return fmt.Errorf("failed to clone and setup repo: %w", err)
	}

	if err := w.syncBaseBranch(); err != nil {
		return fmt.Errorf("failed to sync base branch: %w", err)
	}

	if err := w.runRalph(); err != nil {
		return fmt.Errorf("failed to run ralph: %w", err)
	}

	w.displayStats()

	return nil
}

func (w *WorkflowCmd) setupGitHubAuth() error {
	owner, repo := w.parseOwnerRepo()
	if owner == "" || repo == "" {
		return fmt.Errorf("failed to parse owner/repo from URL: %s", w.RepoURL)
	}

	fmt.Println("Setting up GitHub App token and configuring git authentication...")
	return github.ConfigureGitAuth(context.Background(), owner, repo, DefaultSecretsDir)
}

func (w *WorkflowCmd) setupOpenCodeCredentials() error {
	fmt.Println("Setting up OpenCode credentials...")

	openCodeDir := filepath.Join(os.Getenv("HOME"), ".local", "share", "opencode")
	if err := os.MkdirAll(openCodeDir, 0755); err != nil {
		return fmt.Errorf("failed to create OpenCode directory: %w", err)
	}

	authFile := filepath.Join(DefaultOpenCodeSecretsDir, "auth.json")
	if _, err := os.Stat(authFile); err == nil {
		destPath := filepath.Join(openCodeDir, "auth.json")
		data, err := os.ReadFile(authFile)
		if err != nil {
			return fmt.Errorf("failed to read auth file: %w", err)
		}
		if err := os.WriteFile(destPath, data, 0644); err != nil {
			return fmt.Errorf("failed to write auth file: %w", err)
		}
		fmt.Printf("Copied OpenCode credentials to %s\n", destPath)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to check auth file: %w", err)
	}

	return nil
}

func (w *WorkflowCmd) configureGitUser() {
	fmt.Println("Configuring git user...")
	runGitCmd("config", "--global", "user.name", w.BotName)
	runGitCmd("config", "--global", "user.email", w.BotEmail)
}

func (w *WorkflowCmd) cloneAndSetupRepo() error {
	fmt.Printf("Cloning repository: %s\n", w.RepoURL)

	workDir := "/workspace/repo"
	if err := os.MkdirAll(filepath.Dir(workDir), 0755); err != nil {
		return fmt.Errorf("failed to create work dir: %w", err)
	}

	if _, err := os.Stat(workDir); err == nil {
		os.RemoveAll(workDir)
	}

	if err := runGitCmd("clone", "-b", w.Branch, w.RepoURL, workDir); err != nil {
		if err := runGitCmd("clone", w.RepoURL, workDir); err != nil {
			return fmt.Errorf("failed to clone repository: %w", err)
		}
	}

	if err := os.Chdir(workDir); err != nil {
		return fmt.Errorf("failed to change to work dir: %w", err)
	}

	setupCmd := &SetupWorkspaceCmd{WorkspaceDir: "/workspace"}
	if err := setupCmd.Run(); err != nil {
		return fmt.Errorf("failed to setup workspace: %w", err)
	}

	return nil
}

func (w *WorkflowCmd) syncBaseBranch() error {
	ralphConfig, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	baseBranch := w.BaseBranch
	if baseBranch == "" {
		currentBranch, err := getCurrentBranch()
		if err != nil {
			return fmt.Errorf("failed to get current branch: %w", err)
		}
		projectBranch := w.Branch

		baseBranch = resolveBaseBranchForWorkflow(currentBranch, projectBranch, ralphConfig.DefaultBranch)
	}

	fmt.Printf("Base branch: %s\n", baseBranch)

	if err := w.fetchBaseBranch(baseBranch); err != nil {
		fmt.Printf("Warning: failed to fetch base branch: %v\n", err)
		return nil
	}

	needsMerge, err := w.checkIfMergeNeeded(baseBranch)
	if err != nil {
		return fmt.Errorf("failed to check if merge needed: %w", err)
	}

	if !needsMerge {
		fmt.Println("Project branch is up-to-date with base branch")
		return nil
	}

	fmt.Println("Project branch is behind base branch, attempting merge...")
	return w.mergeBaseBranch(baseBranch)
}

func (w *WorkflowCmd) fetchBaseBranch(baseBranch string) error {
	fmt.Printf("Fetching base branch: %s\n", baseBranch)
	runGitCmd("fetch", "origin", baseBranch+":"+baseBranch)
	runGitCmd("fetch", "origin", baseBranch)
	return nil
}

func (w *WorkflowCmd) checkIfMergeNeeded(baseBranch string) (bool, error) {
	if err := runGitCmd("rev-parse", "--verify", baseBranch); err != nil {
		return false, nil
	}

	mergeBase, err := runGitCmdOutput("merge-base", "HEAD", baseBranch)
	if err != nil {
		return false, err
	}

	baseCommit, err := runGitCmdOutput("rev-parse", baseBranch)
	if err != nil {
		return false, err
	}

	return strings.TrimSpace(mergeBase) != strings.TrimSpace(baseCommit), nil
}

func (w *WorkflowCmd) mergeBaseBranch(baseBranch string) error {
	if err := runGitCmd("merge", baseBranch, "--no-edit"); err != nil {
		fmt.Println("Merge had conflicts - resolving with AI...")
		runGitCmd("merge", "--abort")

		return w.resolveConflictsWithAI(baseBranch)
	}

	fmt.Println("Merge successful (fast-forward or no conflicts)")
	return nil
}

func (w *WorkflowCmd) resolveConflictsWithAI(baseBranch string) error {
	fmt.Println("Running AI to resolve merge conflicts...")

	instructionsFile := "/tmp/merge-instructions.md"
	instructions := fmt.Sprintf(`You need to resolve merge conflicts between the base branch (%s) and the current branch (%s).

Steps:
1. Run 'git merge %s' to see the conflicts
2. Examine the conflicting files and resolve each conflict
3. After resolving, run 'git add <resolved-files>' and 'git commit'
4. Write a brief summary of the merge to 'report.md'

Focus on accepting the correct changes from both branches. If there are test failures after resolving, fix them.
`, baseBranch, w.Branch, baseBranch)

	if err := os.WriteFile(instructionsFile, []byte(instructions), 0644); err != nil {
		return fmt.Errorf("failed to write merge instructions: %w", err)
	}

	projectPath := w.ProjectPath
	if !filepath.IsAbs(projectPath) {
		projectPath = filepath.Join("/workspace/repo", projectPath)
	}

	ctx := &execcontext.Context{}
	ctx.SetProjectFile(projectPath)
	ctx.SetLocal(true)
	ctx.SetNoNotify(true)
	ctx.SetVerbose(w.Verbose)
	ctx.SetNoServices(w.NoServices)
	ctx.SetInstructions(instructionsFile)

	ralphConfig, _ := config.LoadConfig()
	maxIterations := w.MaxIterations
	if maxIterations == 0 && ralphConfig != nil {
		maxIterations = ralphConfig.MaxIterations
	}
	if maxIterations == 0 {
		maxIterations = 10
	}
	ctx.SetMaxIterations(maxIterations)

	os.Setenv("RALPH_WORKFLOW_EXECUTION", "true")

	// Use project.Execute to resolve conflicts and complete the project.
	// We ignore the error here as we'll check the file state and commit manually below.
	_ = project.Execute(ctx, w.cleanupRegistrar)

	if _, err := os.Stat("/workspace/repo/report.md"); err == nil {
		fmt.Println("AI generated merge summary")
	}

	if err := runGitCmd("diff", "--cached", "--quiet"); err != nil {
		fmt.Println("AI did not commit the merge - committing now...")
		runGitCmd("add", "-A")
		runGitCmd("commit", "-m", fmt.Sprintf("Merge %s into %s", baseBranch, w.Branch))
	}

	return nil
}

func (w *WorkflowCmd) runRalph() error {
	fmt.Println("Running ralph...")

	projectPath := w.ProjectPath
	if !filepath.IsAbs(projectPath) {
		projectPath = filepath.Join("/workspace/repo", projectPath)
	}

	ralphConfig, _ := config.LoadConfig()
	maxIterations := w.MaxIterations
	if maxIterations == 0 && ralphConfig != nil {
		maxIterations = ralphConfig.MaxIterations
	}
	if maxIterations == 0 {
		maxIterations = 10
	}

	ctx := &execcontext.Context{}
	ctx.SetProjectFile(projectPath)
	ctx.SetMaxIterations(maxIterations)
	ctx.SetLocal(true)
	ctx.SetNoNotify(true)
	ctx.SetVerbose(w.Verbose)
	ctx.SetNoServices(w.NoServices)
	ctx.SetInstructionsMD(w.InstructionsMD)
	ctx.SetDebugBranch(w.DebugBranch)
	ctx.SetBaseBranch(w.BaseBranch)

	os.Setenv("RALPH_WORKFLOW_EXECUTION", "true")

	if err := project.Execute(ctx, w.cleanupRegistrar); err != nil {
		return fmt.Errorf("ralph execution failed: %w", err)
	}

	return nil
}

func (w *WorkflowCmd) displayStats() {
	fmt.Println("Displaying OpenCode statistics...")
	cmd := exec.Command("opencode", "stats")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run()
}

func (w *WorkflowCmd) parseOwnerRepo() (string, string) {
	parts := strings.TrimPrefix(w.RepoURL, "https://github.com/")
	parts = strings.TrimSuffix(parts, ".git")
	parts = strings.TrimSuffix(parts, "/")

	if strings.Contains(parts, "/") {
		split := strings.SplitN(parts, "/", 2)
		return split[0], split[1]
	}

	return "", ""
}

func resolveBaseBranchForWorkflow(currentBranch, projectBranch, defaultBranch string) string {
	if currentBranch != projectBranch {
		return currentBranch
	}
	return defaultBranch
}

func getCurrentBranch() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func runGitCmd(args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func runGitCmdOutput(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}
