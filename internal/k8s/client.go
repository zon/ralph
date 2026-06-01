package k8s

import "context"

type Client interface {
	GetCurrentContext(ctx context.Context) (Context, error)
	CreateOrUpdateConfigMap(ctx context.Context, name, namespace, kubeContext string, data map[string]string) error
	CreateOrUpdateSecret(ctx context.Context, name, namespace, kubeContext string, data map[string]string) error
}

type client struct{}

func NewClient() Client {
	return &client{}
}

var _ Client = (*client)(nil)
