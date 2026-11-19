# Milestones

## Repository Prompts

The launcher reads its prompts from `.ralph/config.yaml` instead of holding them in the chart. Each GitHub event and the optional schedule is its own work stream, and one target namespace serves every run. A prompt change reaches the next run without a launcher release.

* [Launcher Design](design/launcher.md)
* [Launcher Schedule Design](design/launcher-schedule.md)
* [Launcher Target Design](design/launcher-target.md)
* [Prompt Design](design/prompt.md)
* [Config Design](design/app/ralph-config.md#prompts)

## Container Scope

The Ralph image carries Ralph, the argo command line, and OpenCode with only the runtime they need. Language toolchains move to images under `containers/`, starting with `ralph-go` for the Go toolchain, air, and templ.

* [Container Design](design/container.md)
* [Deployment Design](design/deployment.md)
* [Workflow Container Design](design/workflow.md#debug)
