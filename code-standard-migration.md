# Code Standard Migration

Recommendations for bringing the codebase in line with our coding standards (shallow functions, deep modules) and testing conventions.

## Shallow Functions

Functions should be simple, avoid deep nesting, and have a single responsibility.

### Long functions with multiple responsibilities

These functions do too much. Each numbered sub-task should be extracted into a well-named helper.

**`project/execute.go:Execute`** (152 lines, 5+ responsibilities)
- Decides local vs. remote execution
- Loads project and config, validates git state
- Switches branches
- Runs iteration loop
- Generates PR summary, creates PR, sends notification

Extract: `validateGitState()`, `switchToBranch()`, `createPullRequest()`.

**`requirement/execute.go:Execute`** (126 lines, 4 responsibilities)
- Validates project file and checks for blocked.md
- Starts services and handles service failures with a fix prompt
- Generates and runs the AI prompt
- Cleans up logs, normalizes and stages the project file

Extract: `checkBlocked()`, `startAndHandleServices()`, `cleanupAfterAgent()`.

**`project/iteration.go:RunIterationLoop`** (105 lines)
- Manages iteration counting and completion detection
- Runs each iteration via `requirement.Execute`
- Commits changes and reports newly passing requirements

Extract: `runSingleIteration()`, `reportNewlyPassingRequirements()`.

**`ai/ai.go:GeneratePRSummary`** (99 lines, 4 responsibilities)
- Loads project file and config
- Gets git commit log
- Builds the prompt string
- Runs opencode and reads result from temp file

Extract: `buildPRSummaryPrompt()`, `runOpenCodeToFile()`.

**`cmd/run.go:RunCmd.Run`** (88 lines)
- Handles version flag
- Validates all inputs and flag combinations
- Delegates to project.Execute or requirement.Execute

Extract input validation into a helper.

**`cmd/config_webhook.go:ConfigWebhookConfigCmd.Run`** (84 lines), **`ConfigWebhookSecretCmd.Run`** (80 lines)

Both mix reading existing state, generating new config, registering webhooks, and writing K8s resources. Break into discrete steps.

**`github/app.go:ConfigureGitAuth`** (78 lines), **`github/github.go:CreatePR`** (77 lines)

`CreatePR` handles both creation and updating an existing PR in one function. Extract the "PR already exists" fallback into `updateExistingPR()`.

**Other long functions** (30-60 lines each):
- `cmd/config_github.go:ConfigGithubCmd.Run` (76 lines)
- `cmd/context.go:loadContextAndNamespace` (63 lines)
- `cmd/merge.go:MergeCmd.runLocal` (63 lines)
- `cmd/config_opencode.go:ConfigOpencodeCmd.Run` (55 lines)
- `webhook/server.go:handleWebhook` (60 lines)
- `services/services.go:startService` (50 lines)
- `workflow/scripts.go:buildVolumes` (59 lines)
- `project/iteration.go:CommitChanges` (68 lines) -- reads report, stages, commits, pulls, pushes

### Deeply nested functions (3+ levels)

**`github/github.go:CreatePR`** (4 levels deep)
The "PR already exists -> extract URL -> update" path is nested inside error handling. Extract the fallback into its own function.

**`git/git.go:generateCommitMessage`** and **`categorizeFiles`** (3 levels)
The categorization logic inside `generateCommitMessage` has nested if/else/for. Extract `categorizeFile()` as a helper.

**`workflow/scripts.go:buildVolumeMounts`** and **`buildVolumes`** (3 levels, duplicated)
Nearly identical loops for ConfigMaps and Secrets. Extract a shared `buildMountForItem(name, destFile, destDir, index)` helper to eliminate both nesting and duplication.

## Deep Modules

Modules should hide complexity behind simple interfaces.

### Exported fields that should be private

**`workflow.Workflow`** and **`workflow.MergeWorkflow`** -- all fields exported but only set by constructor functions within the same package. Make fields unexported. The one external write (`webhook/events.go:57` setting `mw.PRNumber`) should use a constructor parameter instead.

**`services.Process`** -- `Cmd` (`*exec.Cmd`), `PID`, and `Service` are exported. Callers only need `Name`, `Stop()`, and `IsRunning()`. Make `Cmd`, `PID`, `Service`, and `logFile` unexported.

**`context.Context`** -- exports both fields and getter methods (`IsDryRun()`, `IsVerbose()`, etc.) for the same data. Pick one pattern: either make fields unexported and use getters, or drop the getter methods.

**`webhook.GithubPayload`** -- only used within the `webhook` package. Make it unexported.

### Bloated API surfaces

**`config` package** (21+ exports) -- many types (`Before`, `ConfigMapMount`, `SecretMount`, `ImageConfig`, `AppInfo`) are pure data containers consumed by only 1-2 packages. The default instruction strings (`DefaultCommentInstructions`, `DefaultMergeInstructions`, `DefaultFixServiceInstructions`) could be unexported and accessed through the loaded `RalphConfig`.

**`git` package** (24 exports) -- mixes primitive operations with the higher-level `CommitChanges`. Also `PushBranch` and `PushCurrentBranch` overlap significantly; consolidate into one function.

**`github` package** (16 exports) -- `IsGHInstalled` (taking `*context.Context`) and `IsGHCLIAvailable` (taking `context.Context`) are near-duplicates. Consolidate. The package also mixes two concerns: GitHub App authentication and gh CLI operations.

**`services` package** (13 exports) -- `CheckPort`, `WaitForPort`, `WaitForPortRelease` are only used internally. Make them unexported.

### Misplaced functionality

**`k8s.GenerateSSHKeyPair`** is a crypto utility, not a Kubernetes operation. Move it to a more appropriate location.

**`project.CommitChanges`** is a git-level operation (reads report, stages, commits, pushes). It belongs closer to the `git` package or should be clearly named to indicate its orchestration role.

**`cmd.WebhookConfigResult`** and **`cmd.WebhookSecretsResult`** are only used in tests. Move to a test file or make them unexported.

## Testing Conventions

Tests should use table-driven tests with `t.Run()`, testify for assertions, `t.TempDir()` for filesystem work, and never call external tools without dry-run.

### Use `t.TempDir()` instead of manual temp directories

**`internal/config/config_test.go`** -- 14 tests use `os.MkdirTemp` + `defer os.RemoveAll`. Replace all with `t.TempDir()`.

### Use `t.Setenv()` instead of `os.Setenv`

**`internal/cmd/merge_test.go`** -- uses `os.Setenv("PATH", ...)` without cleanup. Use `t.Setenv` which automatically restores the original value.

### Don't mutate global state

**`internal/github/app_test.go`** -- mutates `http.DefaultTransport` globally, preventing parallel test execution. Use a local `http.Client` with a custom transport instead.

### Migrate `os.Chdir` to `t.Chdir()`

15+ test files use `os.Chdir` with `defer` to restore the original directory. Go 1.24 introduced `t.Chdir()` which is parallel-safe and automatically restores the directory. Migrate all occurrences.

### Deduplicate test helpers

A `contains` helper function is defined identically in 3 separate test files. Extract it to `internal/testutil`.

### Standardize assertion style

13 test files use testify (`assert`/`require`), 21 use plain `if` checks, and 3 mix both. The testing doc specifies testify. Migrate the plain `if`-check tests to use `assert`/`require`.

### Improve low-value tests

**`internal/logger/logger_test.go`** and **`internal/notify/notify_test.go`** only assert "no panic" with no behavioral checks. Add assertions on actual output.

**`internal/services/services_test.go`** uses hardcoded ports and `time.Sleep` for timing, which can cause flakiness. Use dynamic port allocation and polling instead.

### Use `testutil.NewContext()` consistently

**`internal/project/complete_test.go`** constructs `context.Context` directly instead of using `testutil.NewContext()`. Use the helper for consistency.

## Priority Order

1. **Extract long functions** -- `project/Execute`, `requirement/Execute`, `project/RunIterationLoop`, `ai/GeneratePRSummary`, `github/CreatePR`
2. **Unexport internal fields** -- `workflow.Workflow`, `services.Process`, `context.Context` (pick fields or getters), `webhook.GithubPayload`
3. **Reduce API surfaces** -- `config`, `git`, `github`, `services` packages
4. **Fix test hygiene** -- `t.TempDir()` migration, `t.Setenv()`, testify adoption, `t.Chdir()` migration
5. **Flatten nested functions** -- `CreatePR`, `buildVolumeMounts`/`buildVolumes`, `generateCommitMessage`
6. **Relocate misplaced code** -- `k8s.GenerateSSHKeyPair`, `project.CommitChanges`, test-only exports in `cmd`
