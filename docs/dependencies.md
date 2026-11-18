# Dependencies

## Held Back Dependencies

### Ubuntu

- Hold `ubuntu:26.04`.
- Why - Ubuntu 26.10 is the newest stable release, but Playwright supports only 22.04, 24.04, and 26.04. Browser toolchain images install browsers through Playwright, so the runtime follows the releases it supports. 26.04 is also the current LTS.
- Lift when - Playwright supports a newer Ubuntu release. Prefer the next LTS, 28.04, over an interim release.
- Where - `containers/ralph/Containerfile`, in the git and runtime stages.
