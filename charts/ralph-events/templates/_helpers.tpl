{{- define "ralph-events.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "ralph-events.fullname" -}}
{{- if .Values.fullnameOverride -}}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- $name := default .Chart.Name .Values.nameOverride -}}
{{- if contains $name .Release.Name -}}
{{- .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}
{{- end -}}

{{- define "ralph-events.labels" -}}
helm.sh/chart: {{ printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" }}
app.kubernetes.io/name: {{ include "ralph-events.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end -}}

{{- define "ralph-events.selectorLabels" -}}
app.kubernetes.io/name: {{ include "ralph-events.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}

{{- define "ralph-events.natsUrl" -}}
{{- required "nats.url is required" .Values.nats.url -}}
{{- end -}}

{{- define "ralph-events.natsAuthSecretName" -}}
{{- required "nats.auth.secret.name is required" .Values.nats.auth.secret.name -}}
{{- end -}}

{{- define "ralph-events.natsAuthUsernameKey" -}}
{{- default "username" .Values.nats.auth.secret.usernameKey -}}
{{- end -}}

{{- define "ralph-events.natsAuthPasswordKey" -}}
{{- default "password" .Values.nats.auth.secret.passwordKey -}}
{{- end -}}

{{- define "ralph-events.githubEventSourceName" -}}
{{- default (include "ralph-events.fullname" . | replace "." "-") .Values.github.eventSourceName -}}
{{- end -}}

{{- define "ralph-events.webhookHost" -}}
{{- $url := .Values.github.webhook.url -}}
{{- if $url -}}
{{- (urlParse $url).hostname -}}
{{- end -}}
{{- end -}}

{{- define "ralph-events.validateWebhookTLS" -}}
{{- $tls := default dict .Values.github.webhook.tls -}}
{{- $url := .Values.github.webhook.url -}}
{{- if and $tls.secretName $url (not (regexMatch "(?i)^https://" $url)) -}}
{{- fail "github.webhook.url must use the https scheme when github.webhook.tls is set" -}}
{{- end -}}
{{- end -}}

{{- define "ralph-events.repoSubject" -}}
{{- $repo := .repo -}}
{{- if $repo.subject -}}
{{- $repo.subject -}}
{{- else -}}
{{- printf "ralph.repo.%s.%s" ($repo.owner | lower | replace "." "-" | replace "_" "-") ($repo.name | lower | replace "." "-" | replace "_" "-") -}}
{{- end -}}
{{- end -}}

{{- define "ralph-events.sensorName" -}}
{{- $repo := .repo -}}
{{- printf "%s-%s-%s" (include "ralph-events.fullname" .root) $repo.owner $repo.name | lower | replace "_" "-" | replace "." "-" | trunc 63 | trimSuffix "-" -}}
{{- end -}}
