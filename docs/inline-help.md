# Inline Help

How to write the doc files built into the Ralph app: the markdown documents that Ralph embeds in its binary and renders as help inside the running app. It applies the [Prose Guidelines](zpecs/prose.md) to prose shown in the terminal.

## Scope

The guide covers the markdown documents the app itself displays, such as the configuration reference behind `ralph help config`, embedded from [../internal/config/config.md](../internal/config/config.md). It does not cover repository guides like [projects.md](projects.md) or prompt instructions fed to agents.

## Examples

Inline help renders in a narrow terminal, so large blocks are hard to read. Avoid one big code block with comments doing the explaining. Split it into small examples, each next to a description of what it shows.
