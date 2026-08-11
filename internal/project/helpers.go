package project

func ForProjectInput(p *Project) *InputFile {
	return &InputFile{
		path:    p.Path,
		kind:    inputProject,
		project: p,
	}
}

func ForOrchestrationInput(path string) *InputFile {
	return &InputFile{
		path: path,
		kind: inputOrchestration,
	}
}

func ForSpecInput(path string) *InputFile {
	return &InputFile{
		path: path,
		kind: inputSpec,
	}
}

var anyPathValue = "/workspace/repo/projects/test-project.yaml"

func AnyPath() string {
	return anyPathValue
}

var anyJSONPathValue = "/workspace/repo/projects/test-project.json"

func AnyJSONPath() string {
	return anyJSONPathValue
}

var lastSavedValue *Project

func LastSaved() *Project {
	return lastSavedValue
}

func SetLastSaved(p *Project) {
	lastSavedValue = p
}

