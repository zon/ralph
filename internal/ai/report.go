package ai

import (
	"fmt"
	"os"
	"strings"
)

// nothingToDo is the exact report content the agent writes when no work was
// needed.
const nothingToDo = "NOTHING_TO_DO"

// reportFile is the agent's report file.
const reportFile = "report.md"

// Report is the content of the agent's report.md.
type Report struct {
	Content string
}

// IsNothingToDo reports whether the report says nothing to do. The comparison
// ignores surrounding whitespace.
func (r Report) IsNothingToDo() bool {
	return strings.TrimSpace(r.Content) == nothingToDo
}

// ReadReport reads the agent's report from report.md in the working directory.
func ReadReport() (Report, error) {
	data, err := os.ReadFile(reportFile)
	if err != nil {
		return Report{}, fmt.Errorf("failed to read %s: %w", reportFile, err)
	}
	return Report{Content: string(data)}, nil
}
