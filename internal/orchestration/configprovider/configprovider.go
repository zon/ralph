package configprovider

import "context"

type PromptClient interface {
	ProviderKey(provider string) (string, error)
}

type AuthClient interface {
	Load() (map[string]string, error)
	Write(keys map[string]string) error
}

type K8sClient interface {
	StoreProviderSecret(ctx context.Context, kubeContext, namespace string, keys map[string]string) error
}

type Runner struct {
	prompt PromptClient
	auth   AuthClient
	k8s    K8sClient
}

func New(prompt PromptClient, auth AuthClient, k8s K8sClient) *Runner {
	return &Runner{prompt: prompt, auth: auth, k8s: k8s}
}

func (r *Runner) Run(ctx context.Context, provider, kubeContext, namespace string) error {
	key, err := r.prompt.ProviderKey(provider)
	if err != nil {
		return err
	}
	existing, err := r.auth.Load()
	if err != nil {
		return err
	}
	existing[provider] = key
	if err := r.auth.Write(existing); err != nil {
		return err
	}
	return r.k8s.StoreProviderSecret(ctx, kubeContext, namespace, existing)
}
