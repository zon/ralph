package workspace

type Client struct{}

func (c *Client) ChangeDirectory(path string) error {
	if path == "" {
		return nil
	}
	return Chdir(path)
}
