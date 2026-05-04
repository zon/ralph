package setskills

import (
	"github.com/zon/ralph/internal/git"
)

func SetSkills(branch string) error {
	repoRoot, err := git.FindRepoRoot()
	if err != nil {
		return err
	}

	skills, err := DiscoverSkills(branch)
	if err != nil {
		return err
	}

	contents, err := FetchSkillContents(skills, branch)
	if err != nil {
		return err
	}

	rewritten := RewriteLinks(contents, branch)

	if err := RemoveStaleSkills(repoRoot, skills); err != nil {
		return err
	}

	return WriteSkills(repoRoot, rewritten)
}