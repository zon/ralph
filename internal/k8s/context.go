package k8s

import (
	"context"
	"fmt"
	"strings"
)

type Context struct {
	Name      string
	Namespace string
}

func (c *client) GetCurrentContext(ctx context.Context) (Context, error) {
	stdout, err := runKubectl(ctx, nil, "config", "current-context")
	if err != nil {
		return Context{}, fmt.Errorf("failed to get current context: %w", err)
	}

	name := strings.TrimSpace(stdout.String())

	// --minify keeps only the current context, so the query does not need to
	// embed the context name. Embedding it would break the filter for names
	// containing quotes and would allow jsonpath injection through the
	// kubeconfig.
	args := []string{"config", "view", "--minify", "-o", "jsonpath={.contexts[0].context.namespace}"}
	stdout, err = runKubectl(ctx, nil, args...)
	if err != nil {
		return Context{}, fmt.Errorf("failed to get namespace for context %q: %w", name, err)
	}
	namespace := strings.TrimSpace(stdout.String())

	return Context{
		Name:      name,
		Namespace: namespace,
	}, nil
}
