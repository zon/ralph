# Regressions

## Restore Pebble when the base image ships a fixed build

The runtime removes `/usr/bin/pebble` in `containers/ralph/Containerfile`. The `ubuntu:26.10` base image ships Pebble v1.33.2 built with a Go standard library that has CVE-2026-97054, and the runtime never starts the service manager.

Remove the `/usr/bin/pebble` removal and this entry once an `ubuntu:26.10` image ships a Pebble built with a fixed standard library, Go 1.26.11 or 1.27.4 and later.

## Install Open Code Review from its release once it is not vulnerable

The review image builds the Open Code Review CLI from source in the `ocr` stage of `containers/ralph-review/Containerfile`, with `golang.org/x/net` raised to v0.62.0. The pinned release, v1.14.3, publishes an `opencodereview-linux-amd64` asset built with v0.60.0, which carries CVE-2026-78661, a HIGH denial of service fixed in v0.62.0. A released binary cannot be patched, so the stage builds the command instead.

Return to the released binary once a release ships `golang.org/x/net` v0.62.0 or later. Download `opencodereview-linux-amd64` for the pinned `OCR_VERSION`, verify the SHA-256 checksum published beside it, mark it executable, and copy it to `/usr/local/bin/ocr`. Then delete the Go build stage and this entry, and restore the `internal/docker` test for the download.
