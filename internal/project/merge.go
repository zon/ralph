package project

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

func LoadAll(dir string) ([]*Project, error) {
	var projects []*Project
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		ext := filepath.Ext(path)
		if ext != ".yaml" && ext != ".yml" {
			return nil
		}
		proj, err := LoadProject(path)
		if err != nil {
			return nil
		}
		projects = append(projects, proj)
		return nil
	})
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to walk projects directory: %w", err)
	}
	return projects, nil
}

func FilterPassing(projects []*Project) []*Project {
	var passing []*Project
	for _, p := range projects {
		if IsProjectComplete(p) {
			passing = append(passing, p)
		}
	}
	return passing
}

func DeleteAll(projects []*Project) error {
	for _, p := range projects {
		if err := os.Remove(p.Path); err != nil {
			return fmt.Errorf("failed to delete project file %s: %w", p.Path, err)
		}
	}
	return nil
}
