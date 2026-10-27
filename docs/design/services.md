# Services Design

A run often needs a database, a queue, or another process to exercise the code it changes. Ralph starts those processes for the agent and stops them afterwards. A separate set of commands prepares the environment before anything starts.

Both are configured in `.ralph/config.yaml`. See [Config Design](app/ralph-config.md).

## Before Commands

`before` commands run once per run, before the first iteration and before any service starts.

* They run in order and must exit successfully, unless marked `optional`.
* An optional failure logs a warning and the run continues.
* Typical uses are compilation, code generation, dependency installation, and migrations.

## Services

`services` are long-running processes. Ralph starts them at the beginning of each iteration and stops them when the iteration ends.

* They start in order.
* A service with a `port` is not ready until a TCP connection to that port succeeds.
* Service output goes to `<name>.log` in the repository root, where the agent can read it. Ralph removes the log at the end of the iteration.
* Stop sends SIGTERM, waits, then sends SIGKILL if the process is still running.
* A service with a `port` is also checked for port release after it stops.

## Startup Failure

When a service fails to start or does not become healthy, Ralph invokes the development agent with a prompt to diagnose and fix the startup failure. The prompt receives the service and its error. The iteration then continues, and the development prompt can read the service logs.

If the fix prompt fails, the iteration fails. A non-fatal failure follows [Iteration Recovery Design](iteration-recovery.md).

## Skipping

`ralph run --no-services` skips both service startup and the fix prompt. The remote workflow passes the same choice into the container.

## Related

* [Run Design](run.md)
* [Config Design](app/ralph-config.md)
* [Remote Mode Design](remote.md)
