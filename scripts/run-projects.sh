#!/bin/bash
set -e

# Submits every project file in projects/ as a remote Argo Workflow.
# Each invocation of `ralph` submits the workflow and returns immediately
# (--mode remote, no --follow), so projects run concurrently on the cluster.

PROJECTS_DIR="$(dirname "$0")/../projects"

shopt -s nullglob
projects=("$PROJECTS_DIR"/*.yaml)
shopt -u nullglob

if [ ${#projects[@]} -eq 0 ]; then
  echo "No project files found in ${PROJECTS_DIR}"
  exit 0
fi

failed=0
for project in "${projects[@]}"; do
  echo "Submitting $(basename "$project")..."
  if ! ralph --mode remote "$project"; then
    echo "error: failed to submit $(basename "$project")" >&2
    failed=$((failed + 1))
  fi
done

if [ "$failed" -ne 0 ]; then
  echo "error: ${failed} project(s) failed to submit" >&2
  exit 1
fi
