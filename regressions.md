# Regressions

## Restore Pebble when the base image ships a fixed build

The runtime removes `/usr/bin/pebble` in `containers/ralph/Containerfile`. The `ubuntu:26.10` base image ships Pebble v1.33.2 built with a Go standard library that has CVE-2026-97054, and the runtime never starts the service manager.

Remove the `/usr/bin/pebble` removal and this entry once an `ubuntu:26.10` image ships a Pebble built with a fixed standard library, Go 1.26.11 or 1.27.4 and later.
