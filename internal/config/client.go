package config

type Loader interface {
	Load() (*RalphConfig, error)
}

type Client struct{}

func (c *Client) Load() (*RalphConfig, error) {
	return LoadConfig()
}
