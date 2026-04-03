Explore the codebase and identify two types of code components:

1. **Modules**: Packages, libraries, or logical groupings of code that provide functionality (no main entry point). Examples: internal/auth, internal/api, vendor/logging.

2. **Apps**: Executables, services, or entry points that can be run. Examples: cmd/ralph (CLI), internal/webhook (HTTP server), internal/worker (background service).

Exclude documentation files (README, docs/, *.md) and planning files (projects/, TODO, planning.*).

Write the overview to {{.OverviewPath}} in JSON format with two top-level lists: "modules" and "apps".
Each entry should have "name", "path", and "summary" fields.
