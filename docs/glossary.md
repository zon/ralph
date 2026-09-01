# Glossary

## Project

Any YAML or JSON file that contains an array of work items. Ralph imposes no schema on a project file. It reads an array out of it with the item query and treats everything else as opaque context for the AI agent. See [Project Files](projects.md).

## Item

One element of a project's resolved array, and the unit of one iteration. An item can be any value: a string, a mapping, a nested structure. Ralph identifies it by the hash of its text, which is stable because the project file is not written to during a run.

## Item Query

A [jq](https://jqlang.org/manual/) expression that resolves a project file to its item array. Defaults to `.`, which fits a file whose top level is already an array. Configured with `items` in `.ralph/config.yaml` or `--items`.

## Item Key

The value of an item's `slug`, `id`, or `name` field, checked in that order. A convenience label that names the item in logs and picker output. It is not an identifier. Items are tracked by hash whether or not they have a key.

## Completion Trailer

The last line of an iteration's commit message, written by the development agent to record that an item is finished: a bare `<branch>-<hash>` line, for example `csv-export-IYAWN02`, where the branch is the project branch and the hash is a 7-character base-62 encoding of a SHA-256 digest of the item's text, normalized by trimming surrounding whitespace and lower-casing. Only trailers naming the current branch count. The set of trailers on the branch is ralph's only record of progress. See [Iterations](iterations.md).
