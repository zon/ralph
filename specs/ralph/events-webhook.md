# Events Webhook Specification

## Purpose

The `ralph-events` intake can terminate TLS on its webhook server, attach to a Gateway API gateway, and delete its GitHub hooks when it stops. Each option is off until a release sets it, so a release that sets none renders as it does without this spec.

## Requirements

### Requirement: Server TLS

The `ralph-events` chart SHALL read `github.webhook.tls`. When `github.webhook.tls.secretName` is set, the chart SHALL pass the Secret to the GitHub event source as the server certificate and key, and the event source SHALL serve its webhook over TLS. The chart SHALL read the certificate from `certKey` and the key from `keyKey`, which default to `tls.crt` and `tls.key`.

When `github.webhook.tls` is set, `github.webhook.url` SHALL use the `https` scheme. The chart SHALL reject any other scheme. When `github.webhook.tls` is unset, the event source SHALL serve plain HTTP as it does without this requirement.

#### Scenario: TLS is off by default

- GIVEN `github.webhook.tls` is unset
- WHEN the chart renders
- THEN the event source has no server certificate and no server key

#### Scenario: The Secret becomes the server certificate

- GIVEN `github.webhook.tls.secretName` is `webhook-tls`
- AND `certKey` and `keyKey` are unset
- WHEN the chart renders
- THEN the event source reads its certificate from key `tls.crt` of `webhook-tls`
- AND it reads its key from key `tls.key` of `webhook-tls`

#### Scenario: Custom Secret keys

- GIVEN `github.webhook.tls.certKey` is `cert.pem` and `keyKey` is `key.pem`
- WHEN the chart renders
- THEN the event source reads the certificate and the key from those keys

#### Scenario: A plain HTTP URL with TLS rejected

- GIVEN `github.webhook.tls.secretName` is set
- AND `github.webhook.url` is `http://hooks.example.com/github`
- WHEN the chart renders
- THEN rendering fails with an error

---

### Requirement: Certificate

When `github.webhook.tls.certificate` is set, the chart SHALL render a cert-manager `Certificate` that fills the `github.webhook.tls.secretName` Secret. `issuerRef` SHALL name the issuer and its kind. `dnsNames` SHALL default to the host of `github.webhook.url`.

When `github.webhook.tls.certificate` is unset, the chart SHALL render no `Certificate`, and the operator SHALL supply the Secret.

#### Scenario: The chart creates the certificate

- GIVEN `github.webhook.tls.certificate.issuerRef` names an issuer
- WHEN the chart renders
- THEN the release has a `Certificate` that writes `github.webhook.tls.secretName`
- AND it references that issuer

#### Scenario: DNS names default to the URL host

- GIVEN `github.webhook.url` is `https://hooks.example.com/github`
- AND `dnsNames` is unset
- WHEN the chart renders
- THEN the `Certificate` lists `hooks.example.com`

#### Scenario: No certificate by default

- GIVEN `github.webhook.tls.certificate` is unset
- WHEN the chart renders
- THEN the release has no `Certificate`

---

### Requirement: Gateway route

When `gatewayRoute` is set, the `ralph-events` chart SHALL render a Gateway API `TLSRoute` that passes TLS through by SNI to the event source service on the webhook port. It SHALL pass each `parentRefs` entry through as written, including `sectionName`. `hostnames` SHALL default to the host of `github.webhook.url`. `apiVersion` SHALL default to `gateway.networking.k8s.io/v1`.

The chart SHALL reject `gatewayRoute` without `github.webhook.tls`, because the server must terminate TLS itself. `ingress` SHALL keep its behavior, and a release MAY use either route or neither.

#### Scenario: The route passes TLS through to the service

- GIVEN `gatewayRoute.parentRefs` names a gateway
- AND `github.webhook.tls` is set
- WHEN the chart renders
- THEN the release has a `TLSRoute` attached to that gateway
- AND its backend is the event source service on the webhook port

#### Scenario: Parent references pass through

- GIVEN a `parentRefs` entry sets `sectionName`
- WHEN the chart renders
- THEN the `TLSRoute` parent reference has that `sectionName`

#### Scenario: Hostnames default to the URL host

- GIVEN `gatewayRoute.hostnames` is unset
- AND `github.webhook.url` is `https://hooks.example.com/github`
- WHEN the chart renders
- THEN the `TLSRoute` lists `hooks.example.com`

#### Scenario: A cluster serving an older API version

- GIVEN `gatewayRoute.apiVersion` is `gateway.networking.k8s.io/v1alpha2`
- WHEN the chart renders
- THEN the `TLSRoute` uses that `apiVersion`

#### Scenario: A route without TLS rejected

- GIVEN `gatewayRoute` is set
- AND `github.webhook.tls` is unset
- WHEN the chart renders
- THEN rendering fails with an error

---

### Requirement: Hook cleanup

The `ralph-events` chart SHALL read `github.webhook.deleteHookOnFinish`, which defaults to `false`. When `true`, the event source SHALL delete the webhooks it registered when it stops, so a repository removed from `repositories` loses its webhook on the next restart. When `false`, the event source SHALL leave its webhooks in place.

The option SHALL apply only when the chart registers hooks, which needs `github.apiTokenSecret` or `github.githubApp` together with `github.webhook.url`. The event source cannot clean up after a crash, so a crash MAY leave a hook behind, and the values comments and release notes SHALL say so.

#### Scenario: Hooks stay by default

- GIVEN `github.webhook.deleteHookOnFinish` is unset
- WHEN the chart renders
- THEN the event source does not delete its hooks on finish

#### Scenario: Hooks are deleted when the event source stops

- GIVEN `github.webhook.deleteHookOnFinish` is `true`
- AND the chart registers hooks
- WHEN the event source stops
- THEN it deletes the webhooks it registered

#### Scenario: A removed repository loses its hook

- GIVEN `deleteHookOnFinish` is `true`
- AND a repository is removed from `repositories`
- WHEN the event source restarts
- THEN the removed repository's webhook is deleted

#### Scenario: Cleanup without hook registration

- GIVEN `deleteHookOnFinish` is `true`
- AND the chart does not register hooks
- WHEN the chart renders
- THEN the event source registers and deletes no hooks
