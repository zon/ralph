Apply a semver minor bump to the app version

Bump the app version to 21.1.0: update internal/version/VERSION and the
chart's appVersion. Patch-bump the ralph-webhook chart version to 2.0.97.
The existing version package tests assert the VERSION file stays valid
semver and the chart appVersion matches the app version; they pass after
the bump.

Ralph item 7 completed
