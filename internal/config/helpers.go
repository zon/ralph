package config

func Any() *RalphConfig {
	cfg := &RalphConfig{
		CommentInstructions: defaultCommentInstructions,
	}
	applyDefaults(cfg)
	return cfg
}

func WithVariant(v string) *RalphConfig {
	cfg := Any()
	cfg.Variant = v
	return cfg
}

// WithMode returns a config with the given execution mode.
func WithMode(mode string) *RalphConfig {
	cfg := Any()
	cfg.Mode = mode
	return cfg
}

func WithExtraIterations(n int) *RalphConfig {
	cfg := Any()
	v := n
	cfg.ExtraIterations = &v
	return cfg
}

// WithItems returns a config whose item query selects the given jq expression.
func WithItems(query string) *RalphConfig {
	cfg := Any()
	cfg.Items = query
	return cfg
}

// WithBase returns a config whose resolved base branch is the given branch.
func WithBase(branch string) *RalphConfig {
	cfg := Any()
	cfg.Base = branch
	return cfg
}

// WithCleanup returns a config with project file cleanup enabled.
func WithCleanup() *RalphConfig {
	cfg := Any()
	cfg.Cleanup = true
	return cfg
}

// WithItems chains the item query onto a config.
func (c *RalphConfig) WithItems(query string) *RalphConfig {
	c.Items = query
	return c
}

// WithBase chains the base branch onto a config.
func (c *RalphConfig) WithBase(branch string) *RalphConfig {
	c.Base = branch
	return c
}

// WithCleanup chains project file cleanup onto a config.
func (c *RalphConfig) WithCleanup() *RalphConfig {
	c.Cleanup = true
	return c
}

// WithExtraIterations chains the extra iteration count onto a config.
func (c *RalphConfig) WithExtraIterations(n int) *RalphConfig {
	v := n
	c.ExtraIterations = &v
	return c
}
