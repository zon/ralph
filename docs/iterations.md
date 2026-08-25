# Iterations

A ralph run turns a project file into a sequence of iterations. Each iteration works on exactly one item and, when the work lands, records that item as complete in the commit message. The branch's commit log, not the project file, is the record of what is done, and the project file is never written to while the run is in progress.

## The Loop

```
resolve items ──► pick an item ──► develop it ──► commit (+ completion trailer)
                       ▲                                      │
                       └──────────────────────────────────────┘
                            all items complete? ──► remove project file (optional) ──► open PR
```

Each iteration:

1. **Resolve** — parse the project file and evaluate the [item query](projects.md#item-query) to get the item array. Empty outputs are dropped, and a run whose array comes back empty stops here.
2. **Read completions** — parse `git log <base>..HEAD` for completion trailers and mark the matching items complete (`ralph get complete`).
3. **Exit check** — if nothing is left, leave the loop (`ralph get incomplete` is empty).
4. **Start services** — run configured `before` commands and services (see [Configuration](config.md)).
5. **Pick** — the picker agent selects one incomplete item.
6. **Develop** — the development agent works the picked item.
7. **Commit** — commit whatever the agent produced, using the agent's `report.md` as the message. Its last line is the completion trailer when the item is done.
8. **Stop services**.

## The Project File Is Immutable

For the duration of a run, the project file is read-only:

- The development agent is instructed not to edit it, and has no reason to. It reports completion in its commit message.
- Ralph does not rewrite, normalize, reformat, or stage the project file between iterations.
- The item query is resolved once at the start of the run and reused for every iteration.

So the item array is identical on every iteration, and an item's index means the same thing from the first iteration to the last. This is what makes an index a sufficient identifier, and it is why there is no reconciliation step, no schema for ralph to keep in sync, and no way for a half-written project file to corrupt a run.

The only writes ralph makes to the project file are outside the loop: the optional [cleanup commit](#completing-a-project) after every item is complete, and `ralph validate`, which is a separate command that is not part of a run.

Editing the project file by hand while a run is in progress is unsupported. Nothing enforces it, and ralph will not notice, but indices resolved before the edit no longer refer to the same work.

## Picking

The picker agent receives the full project file, the incomplete items with their indices and keys (the output of `ralph get incomplete`), and the recent commit log. It selects one item based on dependencies between items, logical ordering, and impact.

The development agent then receives that one item verbatim, plus its index and key, plus the full project file for context.

Ralph does not pick in array order. Order the array however you like. The picker reads it as a set of available work, and dependency ordering is one of its inputs.

## Recording Completion

There is no command for this. The iteration prompt tells the development agent that when it has finished the item, the last line of its commit message must be the completion trailer. The agent writes that message itself, as `report.md`. Ralph commits `report.md` verbatim.

```
feat: add CSV serializer for report entries

Converts report entries to RFC 4180 CSV bytes and wires the
serializer into the reports package.

csv-export-2
```

The trailer format is:

```
<branch>-<index>
```

The branch is the project branch and the index identifies the item. The trailer is the message's own trailing paragraph, and one commit may carry several if the iteration finished more than the item it was assigned.

The prompt supplies the branch and index of the picked item, so the agent has the exact line to write rather than deriving it. Ralph does not append or rewrite anything. A trailer exists because the agent decided the item was done and said so.

### When no trailer is written

Not writing one is the normal way to report unfinished work:

- **The agent wrote `report.md` without a trailer** — the iteration commits, the item stays incomplete, and the picker can choose it again. Partial progress carries forward in the branch, and the next iteration sees it in the commit log.
- **The agent wrote no `report.md`** — ralph falls back to generating a changelog for the commit message. That path never produces a trailer, so it never completes an item.

A trailer with an index outside the resolved array is ignored with a warning. The run continues and the item it was aimed at stays incomplete.

## Reading Completion

At the start of each iteration ralph scans the commit messages on the project branch that are not on the base branch, collects every completion trailer, and marks the item at each trailer's index complete. Only trailers naming the current branch count. A trailer naming any other branch is ignored without a warning, so a project branched from another project's branch never inherits that project's completion. A trailer whose index is out of range for the current array is ignored with a warning.

Matching is by branch and index. Because [the project file is immutable](#the-project-file-is-immutable) during a run, the index resolved in iteration 1 refers to the same item in iteration 9, and no further reconciliation is needed.

The same two steps are exposed as commands, and they are the ones the loop itself uses:

```bash
$ ralph get complete                                  # indices, from the commit log
[0, 2]

$ ralph get incomplete projects/csv-export.yaml       # the items that are left
[{"slug": "export-endpoint", ...}, {"slug": "export-error-handling", ...}]

$ ralph get incomplete projects/csv-export.yaml --index
[1, 3]
```

`ralph get complete` needs only the branch and the base. It parses trailers and nothing else, so it works even after the project file has been removed. `ralph get incomplete` is that result subtracted from the resolved item array. An empty array from it is the loop's exit condition, and its non-empty output is what the picker chooses from. Both are read-only and make no AI calls, which makes them the way to check on a run in progress or debug a stuck one. See [CLI reference](cli.md#ralph-get).

### Resuming and re-running

The completion record belongs to the branch, so a run that is interrupted, stopped, or resubmitted picks up where it left off simply by reading the log again. Nothing needs to be restored.

Re-running against a branch after editing the project file *between* runs is where indices can go stale. An item inserted at the top shifts everything after it, and old trailers then point at different work. For a project file that has changed shape, start a fresh branch.

## Iteration Limit

The limit is the resolved item count plus the extra iteration count:

```
limit = len(items) + extraIterations
```

`extraIterations` comes from `--extra-iterations` or `.ralph/config.yaml`. When unset it defaults to 20% of the item count, rounded up. Reaching the limit with items still incomplete is an error — no pull request is opened, and the error names the incomplete items.

The loop also stops early when `blocked.md` exists at the start of an iteration, and stops fatally on billing or quota errors from the AI provider.

## Completing a Project

When every item is complete:

1. Ralph generates a PR summary from the branch's commit log.
2. If cleanup is enabled, ralph deletes the project file and commits the deletion on its own — `chore: clean up completed project <path>` — with no completion trailer.
3. Ralph opens a pull request from the project branch to the base branch.

Cleanup is off by default. Enable it with `--cleanup` or `cleanup: true` in `.ralph/config.yaml`. It is a separate commit so that the commit that removes the file does not also carry code changes, and so the completion history stays readable in the PR after the file is gone.

Opening the pull request is where ralph stops. It does not merge, and approving a PR does not trigger anything — review and merge are the repository's own process. Ralph does respond to review *comments* on an open PR. See [Workflows](workflows.md).

Cleaning up the project file does not erase the completion record — the trailers stay in the branch's history, so a re-run against the same branch still reads the project as complete.

## Foreign Project Files

Because ralph writes completion to git rather than to the project file, the file can belong to another tool entirely. A CI config, an exported issue list, or a task file checked in by a different system can drive a run without ralph mutating it, and without ralph needing to understand any of its fields beyond the item query. See [Choosing a query for foreign files](projects.md#choosing-a-query-for-foreign-files).
