Analyze multiple project files to identify cross-component refactoring themes.

## Project Files to Analyze

Read all YAML files in the projects/ directory. For each project file, extract:
- Project name and description
- All requirements and their categories
- Individual requirement items

## Synthesis Task

Identify requirements that share a common refactoring theme across multiple projects. Examples of common themes:
- "remove inline exec" - requirements about extracting inline command execution
- "consolidate URL parsing" - requirements about unifying URL handling
- "standardize error handling" - requirements about error handling patterns
- "extract shared utilities" - requirements about moving duplicated code

## Output

If you find requirements with common themes across multiple project files:
1. Create a new consolidated project file (e.g., `projects/cross-component-theme.yaml`) with a descriptive name
2. Merge the related requirements into this new project
3. Delete the original per-component project files that were merged
4. Write a brief summary of what was consolidated to {{.SummaryPath}}

If no common themes are found across projects, write "No cross-component themes identified" to {{.SummaryPath}} and do not modify any files.

## Project Naming

Use a descriptive hyphen-separated name that reflects the cross-component theme (e.g., "remove-inline-exec", "consolidate-url-parsing").