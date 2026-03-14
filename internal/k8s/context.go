package k8s

import (
	"context"
	"fmt"
	"strings"
)

// Context represents a Kubernetes context
type Context struct {
	Name      string
	Namespace string
}

// GetCurrentContext gets the current Kubernetes context
func GetCurrentContext(ctx context.Context) (Context, error) {
	stdout, err := runKubectl(ctx, nil, "config", "current-context")
	if err != nil {
		return Context{}, fmt.Errorf("failed to get current context: %w", err)
	}

	name := strings.TrimSpace(stdout.String())

	// Get the namespace for the current context
	args := []string{"config", "view", "-o", fmt.Sprintf("jsonpath='{.contexts[?(@.name==\"%s\")].context.namespace}'", name)}
	stdout, err = runKubectl(ctx, nil, args...)

	var namespace string
	if err == nil {
		namespace = strings.Trim(strings.TrimSpace(stdout.String()), "'")
	}

	return Context{
		Name:      name,
		Namespace: namespace,
	}, nil
}
