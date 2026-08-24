package config

type Loader interface {
	Load() (*RalphConfig, error)
}

type Client struct{}

func (c *Client) Load() (*RalphConfig, error) {
	return LoadConfig()
}

// LoopSteps loads the config and returns the steps of the first loop config
// matching the slug, or an error when none matches.
func (c *Client) LoopSteps(slug string) ([]string, error) {
	cfg, err := LoadConfig()
	if err != nil {
		return nil, err
	}
	return cfg.LoopSteps(slug)
}
