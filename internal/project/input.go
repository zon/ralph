package project

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/zon/ralph/internal/git"
)

type InputFile struct {
	path    string
	kind    inputFileKind
	project *Project
}

type inputFileKind int

const (
	inputProject inputFileKind = iota
	inputOrchestration
	inputSpec
)

func (f *InputFile) IsProject() bool       { return f.kind == inputProject }
func (f *InputFile) IsSpec() bool          { return f.kind == inputSpec }
func (f *InputFile) IsOrchestration() bool { return f.kind == inputOrchestration }
func (f *InputFile) Project() *Project     { return f.project }
func (f *InputFile) Path() string          { return f.path }

func (f *InputFile) Slug() string {
	if f.kind == inputProject {
		return f.project.Slug
	}
	dir := filepath.Dir(f.path)
	base := filepath.Base(dir)
	return git.SanitizeBranchName(base)
}

// ResolveInputFile classifies the file at path as a project document, an
// orchestration, or a spec, and resolves a project document through the client
// so the returned InputFile carries its Project.
func (c *Client) ResolveInputFile(path string) (*InputFile, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve input file path: %w", err)
	}
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("input file not found: %s", absPath)
	}

	base := filepath.Base(absPath)
	ext := strings.ToLower(filepath.Ext(base))

	if ext == ".yaml" || ext == ".yml" || ext == ".json" {
		// A project file is not schema-checked here: any document that parses
		// and yields items resolves, so the input kind is decided by extension
		// alone and the slug and title are derived from the document's metadata
		// with the file name as the fallback.
		proj, err := c.Resolve(absPath, ".")
		if err != nil {
			return nil, err
		}
		return &InputFile{
			path:    absPath,
			kind:    inputProject,
			project: proj,
		}, nil
	}

	if strings.ToLower(base) == "orchestration.md" {
		return &InputFile{
			path: absPath,
			kind: inputOrchestration,
		}, nil
	}

	if strings.ToLower(base) == "spec.md" {
		return &InputFile{
			path: absPath,
			kind: inputSpec,
		}, nil
	}

	return nil, fmt.Errorf("unrecognized input file type: %s", absPath)
}
