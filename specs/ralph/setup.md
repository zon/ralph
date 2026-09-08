# Setup Specification

## Purpose

Confirm that the local machine is ready to run Ralph with git, the GitHub CLI (`gh`), and OpenCode. When a namespace is available from the `--namespace` flag or `workflow.namespace` in `.ralph/config.yaml`, additionally prepare that Kubernetes namespace for Ralph's remote execution on Argo Workflows by writing the GitHub and OpenCode credentials remote workflows consume. Context targeting for the namespace preparation step is defined in [kube-options.md](kube-options.md), with the exception that `ralph setup` prepares a namespace only when the namespace comes from the flag or the config value, never from the kubeconfig context's default namespace.

## Requirements

### Requirement: Sequential Local Readiness Confirmation

The system SHALL confirm via `ralph setup` that the local environment is ready to run Ralph, running confirmation steps in order: (1) git, (2) the GitHub CLI, (3) OpenCode. If any step fails, the command SHALL exit immediately without proceeding to subsequent steps or to namespace preparation.

The git step SHALL confirm that git is installed, that the current directory is inside a git repository, and that `user.name` and `user.email` are configured. The GitHub CLI step SHALL confirm that `gh` is installed and that a GitHub token is available to it, from the `gh` login or the `GITHUB_TOKEN` environment variable. The OpenCode step SHALL confirm that OpenCode is installed and that its `auth.json` contains credentials.

#### Scenario: Local environment fully ready

- GIVEN the current directory is a git repository with `user.name` and `user.email` configured
- AND `gh` is installed and authenticated
- AND OpenCode is installed and authenticated
- WHEN the user runs `ralph setup`
- THEN the command confirms that git, the GitHub CLI, and OpenCode are ready
- AND the command exits with success

#### Scenario: Not a git repository

- GIVEN the current directory is not inside a git repository
- WHEN the user runs `ralph setup`
- THEN an error is returned after the git step
- AND the GitHub CLI and OpenCode steps are not attempted

#### Scenario: Git not installed

- GIVEN git is not on the PATH
- WHEN the user runs `ralph setup`
- THEN an error is returned after the git step
- AND no further confirmation or namespace preparation is attempted

#### Scenario: Git identity not configured

- GIVEN the current directory is a git repository
- AND `user.name` or `user.email` is not configured
- WHEN the user runs `ralph setup`
- THEN an error is returned telling the user to configure a git identity

#### Scenario: GitHub CLI not authenticated

- GIVEN the git step succeeds
- AND neither a `gh` login nor a `GITHUB_TOKEN` environment variable is available
- WHEN the user runs `ralph setup`
- THEN an error is returned after the GitHub CLI step
- AND the OpenCode step is not attempted

#### Scenario: OpenCode not authenticated

- GIVEN the git step succeeds
- AND the GitHub CLI step succeeds
- AND no OpenCode `auth.json` with credentials exists
- WHEN the user runs `ralph setup`
- THEN an error is returned after the OpenCode step
- AND no namespace preparation is attempted

### Requirement: Namespace-Triggered Preparation

The system SHALL prepare a Kubernetes namespace for Ralph workflows only after all local readiness confirmation steps succeed and a namespace value is available from the `--namespace` (`-n`) flag or `workflow.namespace` in `.ralph/config.yaml`. The flag SHALL take precedence over the config value. When no namespace value is available from either source, the command SHALL skip namespace preparation and exit successfully on the local readiness confirmation alone.

#### Scenario: Namespace from config triggers preparation

- GIVEN `workflow.namespace: argo` is set in `.ralph/config.yaml`
- AND the local readiness confirmation succeeds
- WHEN the user runs `ralph setup`
- THEN the namespace `argo` is prepared for Ralph workflows

#### Scenario: Namespace flag overrides the config value

- GIVEN `workflow.namespace: argo` is set in `.ralph/config.yaml`
- AND the user passes `-n staging`
- WHEN `ralph setup` runs
- THEN the namespace `staging` is prepared instead of the config value

#### Scenario: No namespace skips preparation

- GIVEN neither `--namespace` nor `workflow.namespace` is set
- AND the local readiness confirmation succeeds
- WHEN the user runs `ralph setup`
- THEN namespace preparation is skipped
- AND the command exits with success

#### Scenario: Failed confirmation halts before preparation

- GIVEN `workflow.namespace: argo` is set in `.ralph/config.yaml`
- AND the local readiness confirmation fails
- WHEN the user runs `ralph setup`
- THEN namespace preparation is not attempted

### Requirement: Sequential Namespace Preparation

When namespace preparation runs, the system SHALL run preparation steps in order: (1) resolve the Kubernetes context, (2) validate and write GitHub credentials, (3) read and write OpenCode credentials. If any step fails, the command SHALL exit immediately without proceeding to subsequent steps.

#### Scenario: App credentials prepared successfully

- GIVEN `workflow.namespace: argo` is set in `.ralph/config.yaml`
- AND the local readiness confirmation succeeds
- AND a valid GitHub App private key `.pem` file is present
- WHEN the user runs `ralph setup --github-key <key.pem>`
- THEN the GitHub App credentials are validated against the GitHub API
- AND a Kubernetes Secret is written with `app-id` and `private-key`
- AND the OpenCode `auth.json` is written as a Kubernetes Secret
- AND the command exits with success

#### Scenario: Token credentials prepared successfully

- GIVEN `workflow.namespace: argo` is set in `.ralph/config.yaml`
- AND the local readiness confirmation succeeds
- AND a GitHub personal access token is available
- WHEN the user runs `ralph setup --github-token <token>`
- THEN a Kubernetes Secret is written with `token`
- AND the OpenCode `auth.json` is written as a Kubernetes Secret
- AND the command exits with success

#### Scenario: GitHub credential failure halts preparation

- GIVEN `workflow.namespace: argo` is set in `.ralph/config.yaml`
- AND the local readiness confirmation succeeds
- AND an invalid or empty GitHub App private key file is present
- WHEN the user runs `ralph setup --github-key <key.pem>`
- THEN an error is returned after the GitHub credential step
- AND the OpenCode credential step is not attempted

### Requirement: Kubernetes Context Targeting

During namespace preparation, the command SHALL target the cluster given by the `--context` flag, falling back to `workflow.context` in `.ralph/config.yaml`, then to the current kubeconfig context. The namespace comes from the `--namespace` flag or `workflow.namespace` in `.ralph/config.yaml` as described in [Namespace-Triggered Preparation](#requirement-namespace-triggered-preparation). The kubeconfig context's default namespace SHALL NOT be used. See [kube-options.md](kube-options.md) for the shared targeting contract.

#### Scenario: Context override

- GIVEN `--context staging --namespace argo` is passed
- WHEN `ralph setup` runs
- THEN both credential secrets are written to the `staging` context in the `argo` namespace

### Requirement: GitHub Key Flag

When namespace preparation runs, the system SHALL accept an optional `--github-key` flag pointing to an existing `.pem` file containing the GitHub App private key. When provided, the key SHALL be validated against the GitHub API and written to the credentials Secret as `app-id` and `private-key`. The flag SHALL be mutually exclusive with `--github-token`.

#### Scenario: Flag provided writes the secret with a new key

- GIVEN a namespace is targeted via config or `--namespace`
- AND `--github-key <key.pem>` is provided
- WHEN `ralph setup` runs
- THEN the key is validated against the GitHub API
- AND the credentials secret is created or updated with the new key

#### Scenario: Flag omitted reuses the existing secret

- GIVEN a namespace is targeted via config or `--namespace`
- AND `--github-key` is not provided
- AND the GitHub credentials secret already exists in the target namespace
- WHEN `ralph setup` runs
- THEN the GitHub credential step succeeds without reading a local key file

#### Scenario: Credential flag without a targeted namespace

- GIVEN `--github-key <key.pem>` is provided
- AND neither `--namespace` nor `workflow.namespace` is set
- WHEN the user runs `ralph setup`
- THEN an error is returned telling the user that writing credentials requires a targeted namespace

### Requirement: GitHub Token Flag

When namespace preparation runs, the system SHALL accept an optional `--github-token` flag containing a GitHub personal access token. When provided, the token SHALL be written to the `token` key of the credentials Secret without validation. The flag SHALL be mutually exclusive with `--github-key`.

When neither `--github-key` nor `--github-token` is provided and no credentials Secret exists in the target namespace, the system SHALL write the token stored by the `gh` CLI login, falling back to the `GITHUB_TOKEN` environment variable, to the Secret.

#### Scenario: Flag provided writes the token secret

- GIVEN a namespace is targeted via config or `--namespace`
- AND `--github-token <token>` is provided
- WHEN the user runs `ralph setup`
- THEN the credentials secret is created or updated with the `token` key set to the token

#### Scenario: Both flags are mutually exclusive

- GIVEN both `--github-key <key.pem>` and `--github-token <token>` are provided
- WHEN the user runs `ralph setup`
- THEN an error is returned before any step is attempted

#### Scenario: No flags falls back to gh login token

- GIVEN a namespace is targeted via config or `--namespace`
- AND neither flag is provided
- AND no GitHub credentials secret exists in the target namespace
- AND the `gh` CLI login holds a token
- WHEN the user runs `ralph setup`
- THEN the token is read from the `gh` CLI login
- AND the credentials secret is created with the `token` key set to that token

#### Scenario: No flags falls back to environment token

- GIVEN a namespace is targeted via config or `--namespace`
- AND neither flag is provided
- AND no GitHub credentials secret exists in the target namespace
- AND no `gh` CLI login token is available
- AND `GITHUB_TOKEN` is set
- WHEN the user runs `ralph setup`
- THEN the credentials secret is created with the `token` key set to the environment token
