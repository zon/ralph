#!/usr/bin/env bash
set -euo pipefail

# Scan for vulnerabilities in two passes and stop at the first failure:
#   1. govulncheck against the Go dependencies reachable from this module
#   2. Trivy against the working tree, or a target passed on the command line
#
# Usage:
#   scan.sh                    # scan the working tree with Trivy
#   scan.sh image <image>...   # scan container images with Trivy
#
# govulncheck has no severity levels: any reachable vulnerability fails the run.
# Trivy fails only on HIGH and CRITICAL findings.

cd "$(dirname "$0")/.."

if ! command -v govulncheck >/dev/null 2>&1; then
  echo "error: govulncheck is required to scan Go dependencies" >&2
  echo "install: go install golang.org/x/vuln/cmd/govulncheck@latest" >&2
  exit 1
fi

if ! command -v trivy >/dev/null 2>&1; then
  echo "error: trivy is required to scan for vulnerabilities" >&2
  echo "install: https://trivy.dev/latest/docs/installation/" >&2
  exit 1
fi

echo "Scanning Go dependencies with govulncheck..."
status=0
govulncheck ./... || status=$?
if [ "$status" -eq 3 ]; then
  echo "error: govulncheck found vulnerabilities" >&2
  exit 1
fi
if [ "$status" -ne 0 ]; then
  echo "error: govulncheck failed with exit code ${status}" >&2
  exit "$status"
fi

if [ "$#" -eq 0 ]; then
  set -- fs .
fi

target="$1"
shift

echo "Scanning with Trivy (${target})..."
trivy "$target" --severity HIGH,CRITICAL --exit-code 1 "$@"
