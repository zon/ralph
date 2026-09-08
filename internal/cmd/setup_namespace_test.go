package cmd

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/k8s"
	"github.com/zon/ralph/internal/orchestration/setup"
)

func TestSetupContextClientResolveNamespaceTargeting(t *testing.T) {
	tests := []struct {
		name          string
		ralphConfig   *config.RalphConfig
		flagNamespace string
		current       k8s.Context
		expectErr     error
		wantName      string
		wantNamespace string
	}{
		{
			name:          "namespace from config triggers preparation",
			ralphConfig:   &config.RalphConfig{Workflow: config.WorkflowConfig{Namespace: "argo"}},
			current:       k8s.Context{Name: "prod", Namespace: "kube-default"},
			wantName:      "prod",
			wantNamespace: "argo",
		},
		{
			name:          "namespace flag overrides the config value",
			ralphConfig:   &config.RalphConfig{Workflow: config.WorkflowConfig{Namespace: "argo"}},
			flagNamespace: "staging",
			current:       k8s.Context{Name: "prod", Namespace: "kube-default"},
			wantName:      "prod",
			wantNamespace: "staging",
		},
		{
			name:          "config context and namespace avoid the kubeconfig lookup",
			ralphConfig:   &config.RalphConfig{Workflow: config.WorkflowConfig{Context: "lab", Namespace: "ralph"}},
			wantName:      "lab",
			wantNamespace: "ralph",
		},
		{
			name:        "kubeconfig context default namespace does not trigger preparation",
			ralphConfig: &config.RalphConfig{},
			current:     k8s.Context{Name: "prod", Namespace: "argo"},
			expectErr:   setup.ErrNoTargetedNamespace,
		},
		{
			name:        "no config and no flag namespace skips preparation",
			ralphConfig: nil,
			current:     k8s.Context{Name: "prod", Namespace: "argo"},
			expectErr:   setup.ErrNoTargetedNamespace,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			currentCalls := 0
			k8sClient := &k8s.MockClient{
				GetCurrentContextFunc: func(ctx context.Context) (k8s.Context, error) {
					currentCalls++
					return tt.current, nil
				},
			}

			client := &setupContextClient{ctx: context.Background(), k8sClient: k8sClient, ralphConfig: tt.ralphConfig}

			k8sCtx, err := client.Resolve("", tt.flagNamespace)
			if tt.expectErr != nil {
				require.ErrorIs(t, err, tt.expectErr)
				require.Zero(t, currentCalls, "no namespace must not require a kubectl lookup")
				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.wantName, k8sCtx.Name)
			require.Equal(t, tt.wantNamespace, k8sCtx.Namespace)
		})
	}
}
