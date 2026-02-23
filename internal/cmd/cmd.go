package cmd

// Cmd defines the command-line arguments and execution context
type Cmd struct {
	// Subcommands
	Run         RunCmd         `cmd:"" default:"withargs" help:"Execute ralph with a project file (default command)"`
	Comment     CommentCmd     `cmd:"" help:"Run a comment-triggered development iteration"`
	Merge       MergeCmd       `cmd:"" help:"Submit an Argo workflow to merge a completed PR"`
	Config      ConfigCmd      `cmd:"" help:"Configure credentials for remote execution"`
	GithubToken GithubTokenCmd `cmd:"github-token" help:"Generate a GitHub App installation token"`

	version          string       `kong:"-"`
	date             string       `kong:"-"`
	cleanupRegistrar func(func()) `kong:"-"`
}

// ConfigCmd defines the config subcommand group
type ConfigCmd struct {
	Github        ConfigGithubCmd        `cmd:"" help:"Configure GitHub credentials for remote execution"`
	Opencode      ConfigOpencodeCmd      `cmd:"" help:"Configure OpenCode credentials for remote execution"`
	WebhookConfig ConfigWebhookConfigCmd `cmd:"" name:"webhook" help:"Provision webhook-config secret into Kubernetes"`
	WebhookSecret ConfigWebhookSecretCmd `cmd:"webhook-secret" help:"Provision webhook-secrets secret into Kubernetes"`
}

// GithubTokenCmd generates a GitHub App installation token
type GithubTokenCmd struct {
	Owner      string `help:"Repository owner (default: autodetected from git remote)" short:"o"`
	Repo       string `help:"Repository name (default: autodetected from git remote)" short:"r"`
	SecretsDir string `help:"Directory containing GitHub App credentials (default: /secrets/github)" default:"/secrets/github"`
}

// SetVersion sets the version information
func (c *Cmd) SetVersion(version, date string) {
	c.version = version
	c.date = date
	c.Run.version = version
	c.Run.date = date
}

// SetCleanupRegistrar sets the cleanup registrar function
func (c *Cmd) SetCleanupRegistrar(cleanupRegistrar func(func())) {
	c.cleanupRegistrar = cleanupRegistrar
	c.Run.cleanupRegistrar = cleanupRegistrar
	c.Comment.cleanupRegistrar = cleanupRegistrar
	c.Merge.cleanupRegistrar = cleanupRegistrar
}
