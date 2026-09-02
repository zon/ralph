# Project Files

A project is a YAML or JSON file listing the work to do.

## A Simple Project

A simple project is a list of items in `projects/`, named `<slug>.yaml`. The slug names the branch and the pull request:

```yaml
- Reports can be exported as CSV from GET /reports/:id/export
- A request for a missing report ID returns 404
- A malformed report ID returns 400 with an error message
```

Run the project by path. The extension picks the parser: `.yaml` or `.yml`, and `.json`:

```bash
ralph projects/report-export.yaml
```

## What Ralph Does

Each list entry is one item. Ralph creates a branch named after the file, `report-export`, and implements the items, one per iteration. A finished item is marked by a completion trailer in its commit message, so completion lives in the branch history and the file is never written back. When every item is complete, Ralph opens a pull request.

## Check the Project

`ralph validate` parses the file and checks the list holds at least one item. There is no schema check:

```bash
ralph validate projects/report-export.yaml
```
