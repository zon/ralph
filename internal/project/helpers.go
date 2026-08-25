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

