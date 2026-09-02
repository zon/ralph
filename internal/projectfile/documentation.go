package projectfile

import _ "embed"

//go:embed project.md
var projectDocumentation string

// ProjectDocumentation returns the embedded project file guide shown by
// `ralph help project`.
func ProjectDocumentation() string {
	return projectDocumentation
}
