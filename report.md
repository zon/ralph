`ralph loop` opens a pull request for the loop branch when the loop ends

When the loop ends, `ralph loop` now opens a pull request from `loop-<slug>`
to the branch the loop branch was created from, but only when the loop branch
has at least one commit ahead of that base. When no commits were made, the
command exits successfully without opening a pull request. The new
`github.Client.OpenLoopPullRequest` skips PR creation (without error) when the
loop branch has no commits ahead of the base, reuses the existing PR creation
path via a shared `openPullRequest` helper, and wires a `PullRequestOpener`
interface through `LoopCmd` and `loop.Cmd` with fake-based test coverage.
Exported `git.LoopBranch` for reuse.

Tests cover the delegation to the existing PR creation path, the skip when no
commits are ahead, the no-commits sentinel, workflow git-auth refresh, error
propagation, and the wired command paths.

Ralph item 6 completed
