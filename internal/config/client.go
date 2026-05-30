package config

type Client struct{}

func (c *Client) Load() (*RalphConfig, error) {
	return LoadConfig()
}
