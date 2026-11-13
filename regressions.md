# Regressions

## Return to the packaged GitHub CLI

The image builds the GitHub CLI from source in the `gh` stage of `containers/ralph/Containerfile`. The pinned version, v2.104.0, ships `golang.org/x/net` v0.61.0 with CVE-2026-78661, and a released binary cannot be patched.

Install `gh` from the GitHub CLI apt repository in the runtime stage again once a release ships x/net v0.62.0 or later. Then delete the `gh` stage and this entry. Check the latest release before changing `GH_VERSION`.
