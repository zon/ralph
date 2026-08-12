# Glossary

## Project

Any YAML or JSON file that contains an array of work items. Ralph imposes no schema on a project file — it reads an array out of it with the item query and treats everything else as opaque context for the AI agent. See [Project Files](projects.md).

## Item

One element of a project's resolved array, and the unit of one iteration. An item can be any value — a string, a mapping, a nested structure. Ralph identifies it by its index in the array, which is stable because the project file is not written to during a run.

## Item Query

A [jq](https://jqlang.org/manual/) expression that resolves a project file to its item array. Defaults to `.`, which fits a file whose top level is already an array. Configured with `items` in `.ralph/config.yaml` or `--items`.

## Item Key

The value of an item's `slug`, `id`, or `name` field, checked in that order. A convenience label that names the item in commit messages, logs, and picker output. It is not an identifier — items are tracked by index whether or not they have a key.

## Completion Trailer

The last line of an iteration's commit message, written by the development agent to record that an item is finished — `Ralph item <index> completed`, or `Ralph item <index> (<key>) completed` when the item has a key. The set of trailers on a branch is ralph's only record of progress. See [Iterations](iterations.md).
