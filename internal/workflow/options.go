package workflow

import (
	"github.com/zon/ralph/internal/config"
)

type WorkflowOptions struct {
	ImageRepository string
	ImageTag        string
	ConfigMaps      []config.ConfigMapMount
	Secrets         []config.SecretMount
	Env             map[string]string
	DefaultBranch   string
	WorkflowContext string
	Namespace       string
}
