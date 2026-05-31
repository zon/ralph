package config

func Any() *RalphConfig {
	cfg := &RalphConfig{
		Instructions:        defaultInstructions,
		CommentInstructions: defaultCommentInstructions,
		MergeInstructions:   defaultMergeInstructions,
	}
	applyDefaults(cfg)
	return cfg
}

func WithVariant(v string) *RalphConfig {
	cfg := Any()
	cfg.Variant = v
	return cfg
}
