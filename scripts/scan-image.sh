#!/usr/bin/env bash
set -euo pipefail

if [ "$#" -eq 0 ]; then
  echo "usage: $(basename "$0") <image>..." >&2
  exit 2
fi

if ! command -v trivy >/dev/null 2>&1; then
  echo "error: trivy is required to scan an image before pushing" >&2
  echo "install: https://trivy.dev/latest/docs/installation/" >&2
  exit 1
fi

for image in "$@"; do
  echo "Scanning ${image} with Trivy..."
  trivy image --severity HIGH,CRITICAL --exit-code 1 "${image}"
done
