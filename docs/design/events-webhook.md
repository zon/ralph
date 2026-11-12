# Events Webhook Design

How the [intake](events.md) serves its webhook and registers it with GitHub. Each option below is off until a release sets it.

## Server TLS

By default the webhook server speaks plain HTTP and an Ingress or a proxy terminates TLS in front of it. `github.webhook.tls` makes the server terminate TLS itself.

* `secretName` names a Secret holding the certificate and the key.
* `certKey` and `keyKey` name the Secret keys. They default to `tls.crt` and `tls.key`.

The chart passes the Secret to the event source as the server certificate and key. With `tls` set, `github.webhook.url` must use `https`, and the chart rejects any other scheme.

### Certificate

`github.webhook.tls.certificate` asks the chart to create a cert-manager `Certificate` that fills the Secret.

* `issuerRef` names the issuer and its kind.
* `dnsNames` lists the hostnames. It defaults to the host of `github.webhook.url`.

Without it, the operator supplies the Secret.

## Routes

The webhook service needs a public route. The chart offers two, and a release can use either or neither.

### Ingress

`ingress` suits a proxy that terminates TLS.

### Gateway Route

`gatewayRoute` renders a Gateway API `TLSRoute` that passes TLS through by SNI to the webhook service. It needs `github.webhook.tls`, because the server must terminate TLS itself. The chart rejects `gatewayRoute` without it.

* `parentRefs` names the gateways the route attaches to. The chart passes each entry through as written, including `sectionName`.
* `hostnames` lists the SNI names. It defaults to the host of `github.webhook.url`.
* `apiVersion` defaults to `gateway.networking.k8s.io/v1`. A cluster that serves only an older version sets it, such as `gateway.networking.k8s.io/v1alpha2`.

The route targets the event source service on the webhook port.

## Hook Cleanup

`github.webhook.deleteHookOnFinish` defaults to `false`. When `true`, the event source deletes the webhooks it registered when it stops. A repository removed from `repositories` loses its webhook on the next restart. The event source cannot clean up after a crash, so a crash can leave a hook behind.

The option applies only when the chart registers hooks, which needs `github.apiTokenSecret` or `github.githubApp` with `github.webhook.url`.
