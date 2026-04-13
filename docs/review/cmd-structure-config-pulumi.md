# cmd/structure: ConfigPulumiCmd mixes domain logic with infrastructure

**Module:** `internal/cmd`  
**Concern:** Structure

## Issues

- `internal/cmd/config_pulumi.go:21` - `Run()` is not a pure domain function; it embeds infrastructure details in helper methods: `printHeader()` (logging), `promptForToken()` (user interaction), and `createK8sSecret()` (K8s API calls)

The domain logic is "store Pulumi credentials in a Kubernetes secret for Argo Workflows." The current implementation mixes this with:
1. Presentation logic (`printHeader()` - line 51)
2. User interaction logic (`promptForToken()` - line 55, including env var reading and terminal prompting)
3. K8s infrastructure (`createK8sSecret()` - line 75)

Per the domain functions standard, `Run()` should orchestrate domain steps without implementation details. Extract infrastructure concerns so `Run()` reads as a pure domain process.
