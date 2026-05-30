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
