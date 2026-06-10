#!/bin/bash
set -e

# Submits every project file in projects/ as a remote Argo Workflow.
# Each invocation of `ralph` submits the workflow and returns immediately
# (no --local, no --follow), so projects run concurrently on the cluster.

PROJECTS_DIR="$(dirname "$0")/../projects"

shopt -s nullglob
projects=("$PROJECTS_DIR"/*.yaml)
shopt -u nullglob

if [ ${#projects[@]} -eq 0 ]; then
  echo "No project files found in ${PROJECTS_DIR}"
  exit 0
fi

for project in "${projects[@]}"; do
  echo "Submitting $(basename "$project")..."
  ralph "$project"
done
