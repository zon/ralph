package workflow

import (
	"github.com/zon/ralph/internal/config"
)

type WorkflowOptions struct {
	Image           Image
	ConfigMaps      []config.ConfigMapMount
	Secrets         []config.SecretMount
	Env             map[string]string
	DefaultBranch   string
	KubeContext string
	Namespace       string
}
