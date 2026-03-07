//go:build e2e

package e2e

import (
	"fmt"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestNamespacePreflight verifies that all resources required by the E2E tests
// are present before any workflow is submitted. This catches missing secrets,
// missing repo files, and other setup issues up-front rather than letting
// workflows run for minutes before failing with a cryptic error.
//
// Required Kubernetes secrets (in the Argo namespace):
//   - github-credentials  — GitHub App private key used by ralph set-github-token
//   - opencode-credentials — OpenCode auth.json for AI calls
//
// Required files in the test GitHub repository:
//   - test-data/e2e-noop-run.yaml  — pre-completed project file used by E2E workflows
//   - test-data/e2e-resume-run.yaml — pre-completed project file for TestRun_ResumesExistingBranch
func TestNamespacePreflight(t *testing.T) {
	cfg := resolveConfig(t)

	t.Run("kubernetes_secrets", func(t *testing.T) {
		requiredSecrets := []string{
			"github-credentials",
			"opencode-credentials",
		}

		for _, secret := range requiredSecrets {
			secret := secret
			t.Run(secret, func(t *testing.T) {
				out, err := exec.Command(
					"kubectl", "get", "secret", secret,
					"-n", cfg.Namespace,
					"--ignore-not-found",
					"-o", "name",
				).CombinedOutput()
				assert.NoError(t, err, "kubectl get secret %s failed: %s", secret, out)
				assert.NotEmpty(t, strings.TrimSpace(string(out)),
					"secret %q not found in namespace %q — copy it with:\n  kubectl get secret %s -n ralph -o json | jq 'del(.metadata.namespace,.metadata.resourceVersion,.metadata.uid,.metadata.creationTimestamp,.metadata.annotations,.metadata.managedFields)' | kubectl apply -n %s -f -",
					secret, cfg.Namespace, secret, cfg.Namespace,
				)
			})
		}
	})

	t.Run("repo_files", func(t *testing.T) {
		requiredFiles := []string{
			"test-data/e2e-noop-run.yaml",
			"test-data/e2e-resume-run.yaml",
		}

		for _, path := range requiredFiles {
			path := path
			t.Run(strings.ReplaceAll(path, "/", "_"), func(t *testing.T) {
				out, err := exec.Command(
					"gh", "api",
					fmt.Sprintf("repos/%s/contents/%s", cfg.Repo, path),
					"--jq", ".name",
				).CombinedOutput()

				trimmed := strings.TrimSpace(string(out))
				if err != nil || trimmed == "" || strings.Contains(trimmed, "Not Found") {
					assert.Fail(t,
						fmt.Sprintf("file %q not found in repo %q", path, cfg.Repo),
						"Push it with:\n  gh api repos/%s/contents/%s --method PUT --field message='add e2e test data' --field content=$(base64 -w0 <local-copy>)",
						cfg.Repo, path,
					)
				}
			})
		}
	})
}
