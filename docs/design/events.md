# Events Design

`ralph-events` is one cluster-wide release. It receives GitHub webhooks, routes each event by repository, and publishes it to a NATS JetStream subject. A `ralph-launcher` release in each Ralph namespace consumes the subject for its repository.

The chart lives in `charts/ralph-events`.

## Flow

1. GitHub calls the event source webhook.
2. An Argo Events `EventSource` receives the webhook and publishes the event to the cluster event bus.
3. One sensor per repository filters the event by `repository.full_name`.
4. The sensor publishes an envelope with the GitHub event type and body to `ralph.repo.<owner>.<name>`.
5. The `RALPH` JetStream stream retains the message for `stream.maxAge`.

The event source and the routing sensors share an Argo Events `EventBus` in the release namespace. The chart assumes a NATS JetStream server already runs on the cluster and connects to it with `nats.url`. It does not install NATS. Each `ralph-launcher` release creates the `EventBus` for its own namespace.

## NATS Authentication

The routing sensor and the stream bootstrap job authenticate with a Kubernetes service account token. The token namespace maps to a NATS account in the auth callout. The callout exchanges the token for a short-lived NATS user JWT.

The token must be long-lived. The routing sensor reconnects with the same token, and a token from `kubectl create token` or copied from a projected volume expires. The token in a `kubernetes.io/service-account-token` Secret never expires.

Set `nats.auth.serviceAccount.create` to have the chart create a dedicated ServiceAccount and its token Secret. The ServiceAccount name defaults to the release full name plus `-nats`. The ServiceAccount is only an identity, so it needs no RBAC permissions.

Set `nats.auth.tokenSecret` to use an existing Secret instead. Its key defaults to `token`. The chart ignores `nats.auth.serviceAccount` when `tokenSecret.name` is set.

The EventBus reads its own credentials file from `eventbus.accessSecret`. These credentials use basic authentication and are separate from the service account token.

## Subjects

The default subject is the repository path in dot form:

```text
ralph.repo.zon.ralph
```

Dots and underscores in the owner or name become hyphens so the subject keeps the expected token count. A repository entry can override the subject. The stream subjects must cover every override.

The sensor publishes a JSON envelope:

```json
{
  "event": "push",
  "payload": {
    "ref": "refs/heads/main",
    "repository": {
      "full_name": "zon/ralph"
    }
  }
}
```

`event` holds the GitHub event name from the `X-GitHub-Event` header. `payload` holds the GitHub event body. The launcher filters and reads fields below both keys.

A repository can filter the events it routes:

```yaml
repositories:
  - owner: zon
    name: ralph
    events:
      - push
      - pull_request
```

The routing sensor compares `events` with the `headers.X-Github-Event` field. An empty list routes every event.

## GitHub Setup

The chart registers webhooks when `github.apiTokenSecret` names a Secret with a GitHub token and `github.webhook.url` is set. Without a token, create the webhook in GitHub by hand and point it at the event source service or the optional Ingress.

`github.webhookSecret` names the Secret that holds the HMAC secret. Set it so the event source rejects unsigned requests.

## Retention

The `RALPH` stream keeps events for `stream.maxAge`, one week by default. The launcher event source subscribes with core NATS, so a launcher that is down does not replay retained events on its own. The stream keeps the history for inspection and manual replay.

## Prerequisites

The cluster needs a NATS JetStream server, Argo Events, and Argo Workflows. The chart creates `EventSource`, `Sensor`, and `EventBus` resources, so those CRDs must already exist.

Every namespace that runs a launcher must reach NATS, and every `ralph-launcher` release must use the same NATS server.
