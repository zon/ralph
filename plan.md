# Ralph Go CLI - Development Plan

## Overview
Port ralph.sh and develop.sh from bash to a unified Go CLI tool that can be `go install`ed and executed from any working directory.

**Reference Implementations**: 
- `../slow-choice/ralph.sh` - Orchestration script
- `../slow-choice/develop.sh` - Development workflow script
- `../slow-choice/projects/` - Example project YAML files with requirements and steps

**Goal**: Create a robust, cross-platform CLI tool that orchestrates coding agents across a project by:
1. Starting/stopping optional services (defined in .ralph/config.yaml)
2. Gathering context and running AI coding agents
3. Creating branches from project files
4. Running development iterations in a loop
5. Generating PR summaries with AI
6. Submitting GitHub pull requests

---

## Development Steps

### Phase 1: Project Setup & Structure

1. **Initialize Go module**
   - Run `go mod init github.com/zon/ralph`
   - Create basic directory structure: `cmd/ralph/`, `internal/`, `pkg/`
   - Add `.gitignore` for Go projects

2. **Define CLI structure with kong**
   - Install kong CLI framework (github.com/alecthomas/kong)
   - Define CLI struct with Run and Once commands
   - Create commands: `ralph run <project-file>` and `ralph once <project-file>`
   - Add `--help` flag support (automatic with kong)
   - Add `--version` flag support
   - Add `--dry-run` flag for simulation mode (outputs planned actions without executing)
   - Add `--no-notify` flag for disabling notifications

3. **Create configuration types and dry-run mode**
   - Define structs for project YAML parsing (Project, Requirement with steps array)
   - Requirement includes: category, description, steps (array of strings), passing (bool)
   - See `../slow-choice/projects/*.yaml` for reference examples of the format
   - Define struct for ralph config (RalphConfig with Services list)
   - Define struct for secrets YAML (API keys for LLM providers)
   - Add YAML unmarshaling support (gopkg.in/yaml.v3)
   - Add function: `LoadConfig() (*Config, error)` - loads .ralph/config.yaml in cwd
   - Add function: `LoadRalphSecrets() (*RalphSecrets, error)` - checks ~/.ralph/secrets.yaml, then .ralph/secrets.yaml in cwd
   - Create validation logic for project files
   - Support reading/writing project files with requirement status updates
   - Add DryRun context/flag that gets passed to all operations
   - When DryRun=true, operations log what they would do instead of executing

### Phase 2: Service Management (from develop.sh)

4. **Implement service orchestration with dry-run support**
   - Create `internal/services` package
   - Define Service struct: name, command, args, port (optional)
   - Add function: `StartService(svc Service, dryRun bool) (*Process, error)`
   - In dry-run mode: log "Would start service: <name> with command: <cmd> <args>"
   - Track service PIDs/processes (skip in dry-run)
   - Redirect stdout/stderr to /dev/null or log files
   - Return Process handle for later cleanup
   - Services loaded from .ralph/config.yaml in working directory

5. **Add health checking for services**
   - Add function: `WaitForPort(port int, timeout time.Duration) error`
   - Add function: `CheckPort(port int) bool`
   - Use net.Dial to check TCP ports if specified in service config
   - Handle services without ports (just verify process is running)

6. **Implement graceful service shutdown**
   - Add function: `StopAllServices(processes []*Process)`
   - Send SIGTERM, wait for graceful shutdown
   - Send SIGKILL if still running after timeout
   - Use signal handlers (SIGINT, SIGTERM) to cleanup
   - Register cleanup function with defer/signal handling

### Phase 3: Core Functionality - Git Operations

7. **Implement git wrapper utilities with dry-run support**
   - Create `internal/git` package
   - Add function: `GetCurrentBranch(dryRun bool) (string, error)`
   - Add function: `BranchExists(name string, dryRun bool) bool`
   - Add function: `CreateBranch(name string, dryRun bool) error`
   - Add function: `CheckoutBranch(name string, dryRun bool) error`
   - In dry-run mode: log actions like "Would create branch: <name>" without executing

8. **Implement git push functionality**
   - Add function: `PushBranch(branch string) (string, error)`
   - Add function: `HasCommits() bool`
   - Handle error cases and output parsing

9. **Add git diff/log operations**
   - Add function: `GetRecentCommits(count int) ([]string, error)`
   - Add function: `GetCommitsSince(base string) ([]string, error)`
   - Add function: `GetDiffSince(base string) (string, error)`
   - Support for generating change summaries

### Phase 4: Project File Handling

10. **Implement project file validation**
    - Create `internal/project` package
    - Add function: `LoadProject(path string) (*Project, error)`
    - Add function: `ValidateProject(p *Project) error`
    - Add function: `SaveProject(path string, p *Project) error`
    - Extract branch name from file basename

11. **Implement requirement checking**
    - Add function: `CheckCompletion(p *Project) (bool, int, int)` - returns complete, passing, failing counts
    - Count passing/failing requirements
    - Add function: `UpdateRequirementStatus(p *Project, reqID string, passing bool) error`
    - Return detailed status information

### Phase 5: AI Agent Integration (from develop.sh)

12. **Implement prompt generation**
    - Create `internal/prompt` package
    - Add function: `BuildDevelopPrompt(projectFile string) (string, error)`
    - Include recent git history (last 20 commits)
    - Include project requirements from YAML
    - Read and include docs/develop-instructions.md
    - Format as structured context for AI agent

13. **Implement AI client for agent execution with dry-run support**
    - Create `internal/ai` package
    - Add function: `RunAgent(prompt string, dryRun bool) error`
    - Use gollm to send prompts to LLM
    - Configure gollm with API keys from ralph-secrets.yaml
    - In dry-run mode: log "Would run agent with prompt (first 200 chars): <prompt...>" and return success
    - Stream output to stdout for real-time feedback
    - Handle API errors and retries

14. **Create summary generation logic**
    - Add function: `GeneratePRSummary(projectFile string, iterations int) (string, error)`
    - Build prompt from project description and status
    - Include git commits and diff from main..HEAD
    - Request concise 3-5 paragraph summary
    - Use gollm to generate summary
    - Return summary text (no temp file needed)

### Phase 6: Once Command (replaces develop.sh)

15. **Implement `ralph once` command with dry-run**
    - Validate project file exists
    - Start all services defined in .ralph/config.yaml (if present)
    - Wait for services to be healthy (skip health check in dry-run)
    - Generate development prompt
    - Run AI agent with prompt
    - Stage project file after agent completes (git add)
    - Stop services on completion or error
    - Send desktop notifications if enabled (skip in dry-run)
    - In dry-run mode: output complete workflow plan without executing

16. **Add service configuration**
    - Read service definitions from .ralph/config.yaml in working directory
    - Each service has: name, command, args[], port (optional for health check)
    - Allow skipping service startup via flag (--no-services)
    - Fail gracefully if .ralph/config.yaml not found (skip services or warn user)

### Phase 7: Iteration Loop (from ralph.sh)

17. **Implement iteration loop logic**
    - Create `internal/iteration` package
    - Add function: `RunIterationLoop(projectFile string, maxIters int) (int, error)`
    - Each iteration: run develop command internally, commit changes, check completion
    - Track iteration count
    - Stop when all requirements pass OR max iterations reached
    - Return final iteration count

18. **Implement git commit functionality**
    - Add function: `CommitChanges() error` in git package
    - Generate commit message from changed files
    - Stage all changes (git add -A)
    - Commit with descriptive message
    - Handle case where there are no changes to commit

### Phase 8: Orchestration Command (replaces ralph.sh)

19. **Implement `ralph run` command with dry-run**
    - Validate project file exists
    - Extract branch name from project file basename
    - Create and checkout new branch
    - Run iteration loop (develop + commit until complete)
    - Generate PR summary using AI
    - Push branch to origin
    - Create GitHub pull request
    - Display PR URL on success
    - In dry-run mode: output full orchestration plan including all iterations, branches, and PR details

20. **Add branch management**
    - Check if branch already exists (error if it does)
    - Track initial branch for potential rollback
    - Validate git repository exists
    - Handle detached HEAD state

### Phase 9: GitHub Integration

21. **Implement gh CLI wrapper**
    - Create `internal/github` package
    - Add function: `IsGHInstalled() bool`
    - Add function: `IsAuthenticated() bool`
    - Add function: `CreatePR(title, body, base, head string) (string, error)`
    - Parse PR URL from gh output

22. **Add PR creation workflow**
    - Extract PR title from project.description field
    - Use generated summary as PR body
    - Default base branch: main (configurable)
    - Return PR URL for display

### Phase 10: Logging & User Experience

23. **Implement structured logging**
    - Create `internal/logger` package
    - Add colored output support (github.com/fatih/color)
    - Log levels: Info, Success, Warning, Error
    - Match bash script formatting: `[INFO]`, `[SUCCESS]`, etc.
    - Optional verbose mode flag (--verbose)

24. **Add progress indicators**
    - Show current iteration number (N/MAX)
    - Display separator lines between iterations
    - Show which service is starting/stopping
    - Show summary of completion status (X passing, Y failing)
    - Display agent execution progress

### Phase 11: Configuration & Cleanup

25. **Add configuration management**
    - Support for config file (.ralph/config.yaml in cwd) - optional
    - Support for global secrets file (~/.ralph/secrets.yaml) - optional
    - Support for project secrets file (.ralph/secrets.yaml in cwd) - optional
    - Config fields: services[] (name, command, args, port), maxIterations, baseBranch, llmProvider, llmModel
    - Secrets file fields: apiKeys (map of provider -> key)
    - Secrets priority: .ralph/secrets.yaml (cwd) > ~/.ralph/secrets.yaml
    - Command-line flag overrides (highest priority)

26. **Implement cleanup handlers**
    - Create temp directory for prompt/summary files
    - Register signal handlers (SIGINT, SIGTERM)
    - Always stop services on exit (defer pattern)
    - Clean up temp files on exit

### Phase 12: Cross-Platform Support

27. **Handle platform-specific differences**
    - Use `filepath.Join()` for all paths
    - Use `os/exec` for running commands (git, gh, etc.)
    - Handle signal behavior (SIGINT, SIGTERM)

28. **Add desktop notifications**
    - Use `github.com/gen2brain/beeep` for cross-platform notifications
    - Success: "Ralph completed successfully for {project}"
    - Error: "Ralph failed for {project}"
    - Respect --no-notify flag
    - Gracefully handle when notify-send not available

### Phase 13: Documentation & Distribution

29. **Create documentation**
    - Write comprehensive README.md
    - Add usage examples for both commands
    - Document configuration options
    - Document required dependencies (git, gh, LLM API key)
    - Document project YAML format
    - Create installation instructions

30. **Prepare for go install**
    - Ensure main.go is in cmd/ralph/
    - Test `go install github.com/zon/ralph/cmd/ralph`
    - Verify binary works from $GOPATH/bin
    - Test from different working directories

---

## CLI Command Structure

```
ralph run <project-file>              # Full orchestration (branch, iterate, PR)
  --max-iterations int                # Override max iterations (default: 10)
  --dry-run                           # Simulate execution, output plan without running
  --no-notify                         # Disable desktop notifications
  --verbose                           # Enable verbose logging

ralph once <project-file>             # Single development iteration
  --dry-run                           # Simulate execution, output plan without running
  --no-notify                         # Disable desktop notifications
  --no-services                       # Skip service startup
  --verbose                           # Enable verbose logging

ralph version                         # Show version
ralph help                            # Show help
```

---

## Key Dependencies

- **CLI Framework**: `github.com/alecthomas/kong`
- **YAML Parsing**: `gopkg.in/yaml.v3`
- **Colored Output**: `github.com/fatih/color`
- **Notifications**: `github.com/gen2brain/beeep`
- **LLM Client**: `github.com/teilomillet/gollm`

## External Tool Dependencies

- `git` CLI (required)
- `gh` CLI (required for PR creation)
- LLM API keys (via ~/.ralph/secrets.yaml or .ralph/secrets.yaml in cwd)

## Configuration Files

### .ralph/config.yaml (in working directory, optional)

Configuration settings and services. Example:

```yaml
maxIterations: 10
baseBranch: main
llmProvider: anthropic  # or openai, ollama, etc. (gollm supported providers)
llmModel: claude-3-5-sonnet-20241022  # optional, provider-specific model

services:
  - name: database
    command: docker
    args: [compose, up, -d, db]
    port: 5432
  - name: api-server
    command: npm
    args: [run, dev]
    port: 3000
  - name: worker
    command: python
    args: [worker.py]
    # no port = no health check, just verify process is running
```

### ~/.ralph/secrets.yaml (global, optional)

Stores API keys for LLM providers globally. Example:

```yaml
apiKeys:
  anthropic: sk-ant-xxxxxxxxxxxxx
  openai: sk-xxxxxxxxxxxxx
  # Add other provider keys as needed
```

### .ralph/secrets.yaml (project-specific, in working directory, optional)

Overrides global secrets for project-specific API keys. Same format as ~/.ralph/secrets.yaml.

**Note**: Add `.ralph/secrets.yaml` to `.gitignore` to avoid committing secrets.

**Priority**: .ralph/secrets.yaml (cwd) > ~/.ralph/secrets.yaml

---

## Notes

- Maintain compatibility with existing project YAML format
- Configuration stored in .ralph/config.yaml in working directory (optional)
- API keys stored in ~/.ralph/secrets.yaml (global) or .ralph/secrets.yaml (project-specific)
- Secrets priority: .ralph/secrets.yaml (cwd) > ~/.ralph/secrets.yaml
- Agent execution is synchronous (blocking) for now
- gollm supports multiple LLM providers (Anthropic, OpenAI, Ollama, etc.)
- **Dry-run mode**: `--dry-run` flag available on all commands - outputs execution plan without side effects
- Dry-run should be implemented early (step 3) and threaded through all operations for testing
- The `run` command orchestrates everything; `once` is for a single development iteration
- Internal development logic uses "develop" terminology in code
- commit.sh from original is replaced by inline git commit logic
- If .ralph/config.yaml is missing, use default values and skip services
