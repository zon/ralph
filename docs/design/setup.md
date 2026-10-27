# Setup Design

`ralph setup` confirms that the local machine can run Ralph. When a Kubernetes namespace is targeted, it also prepares that namespace for remote execution. Each step runs in order and the command stops at the first failure.

## Local Readiness

The local checks run first:

1. **git** - installed, inside a repository, and `user.name` and `user.email` set.
2. **GitHub CLI** - `gh` installed and a token available from the `gh` login or `GITHUB_TOKEN`.
3. **OpenCode** - installed and holding stored credentials.

These checks always run. A machine with no Kubernetes namespace still needs all three.

## Namespace Preparation

Namespace preparation runs only when `--namespace` or `workflow.namespace` names one. The flag wins. When neither is set, Ralph skips preparation and succeeds on the local checks alone. It never falls back to the kubeconfig context's default namespace.

Preparation then:

1. Resolves the Kubernetes context. `--context` wins over `workflow.context`, which wins over the current context.
2. Confirms kubectl and the argo CLI are installed.
3. Writes GitHub credentials to the `github-credentials` Secret.
4. Writes OpenCode credentials to the `opencode-credentials` Secret.

### GitHub Credentials

* `--github-key <file>` validates a GitHub App private key and writes `app-id` and `private-key`.
* `--github-token <token>` writes the token without validation.
* The flags are mutually exclusive.
* With neither flag, an existing `github-credentials` Secret is reused.
* When there is no Secret, Ralph writes the `gh` login token, then `GITHUB_TOKEN`.

A credential flag with no targeted namespace fails, because there is nowhere to write it.

### OpenCode Credentials

Ralph always writes the local OpenCode credentials to the `opencode-credentials` Secret, as `auth.json`. `workflow.opencodeSecret` chooses the Secret a workflow mounts. When it names another Secret, the output says so, and the local credentials still go to `opencode-credentials`.

## Related

* [Remote Mode Design](remote.md)
* [Config Design](app/ralph-config.md)
