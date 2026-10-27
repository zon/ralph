# Validate Design

`ralph validate <project>` checks that a project file is usable by a run. Validation is optional. A run makes the same checks, so a file that has never been validated still runs.

## Checks

Validation checks exactly three things, in order:

1. The file parses as YAML or JSON.
2. The item query evaluates against the parsed document.
3. The query resolves to at least one non-empty item.

There is no schema check. Unknown fields are accepted, and any shape of item is accepted. The item query resolves as it does for a run, from `--items`, else the config, else `.`. See [Project Design](project.md).

A query that cannot be evaluated, or that yields no items, fails without invoking an agent, because the file itself is not the problem.

## Fix Loop

When the file does not parse, Ralph invokes an AI agent locally to repair it in place, then retries. The loop allows 10 parse attempts: the first read plus up to 9 repairs. The parse error is reported before each repair. A repair that changes nothing fails immediately.

The repair runs on the local machine, never in a workflow. It uses `validate.model` from the config, falling back to the top-level `model`. It omits the configured agent, so an agent that denies writes cannot block the repair. See [Config Design](app/ralph-config.md).

After the file parses, a query that yields no items fails validation without re-entering the fix loop.

## Formatting

When all checks pass, Ralph rewrites the file in canonical YAML. The rewrite preserves the full document and key order. It drops no field and adds none. A `.json` file is written to a sibling `.yaml` file and the original is removed.

## Output

On success, Ralph prints the validated path and the item count, and exits 0. On failure, it exits non-zero with the parse or query error.

## Related

* [Project Design](project.md)
* [Config Design](app/ralph-config.md)
* [Run Design](run.md)
