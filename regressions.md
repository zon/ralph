# Regressions

## `--debug` does not run the branch

`ralph run --mode remote --debug <branch>` and `ralph prompt --mode remote --debug <branch>` should check out that Ralph branch in the container and run it with `go run`. The [workflow container design](docs/design/workflow.md), [run-remote](specs/ralph/run-remote.md), and [workflow-run](specs/ralph/workflow-run.md) say so. The container runs the installed binary instead.

Found when a prompt run with `--debug fix-service-group-stop` showed the old behavior that the branch fixed. The pod log had no checkout and no build.

Cause: 03f1683 deleted `internal/workflow/debug.sh`, the script that cloned the branch into `/workspace/ralph` and wrapped `go run`. Workflows now call `ralph workflow` subcommands directly. The generator still passes `--debug <branch>` to `workflow run` and `workflow prompt` and sets `RALPH_DEBUG_BRANCH`. The `debugAdapter` in `internal/cmd/workflow_run_adapters.go` and the `workflow prompt` flags only store the branch. Nothing uses it to check out or build.

Effect: a change to Ralph cannot be tried in a workflow before it is released and the image is rebuilt.

Fix: restore the checkout and the `go run` wrapper in the container, as the specs say. Add a test that fails when the flag has no effect.
