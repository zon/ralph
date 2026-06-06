package workflowtoken

type RepoClient interface {
	Resolve(owner, repo string) (string, string, error)
}

type GitHubClient interface {
	GenerateToken(owner, repo, secretsDir string) (string, error)
}

type GitClient interface {
	ConfigureAuth(token string) error
}

type WorkflowTokenCmd struct {
	Repo   RepoClient
	GitHub GitHubClient
	Git    GitClient
}

type Flags struct {
	Owner      string
	Repo       string
	SecretsDir string
}

func (c *WorkflowTokenCmd) Run(flags Flags) error {
	owner, repo, err := c.Repo.Resolve(flags.Owner, flags.Repo)
	if err != nil {
		return err
	}

	token, err := c.GitHub.GenerateToken(owner, repo, flags.SecretsDir)
	if err != nil {
		return err
	}

	return c.Git.ConfigureAuth(token)
}
