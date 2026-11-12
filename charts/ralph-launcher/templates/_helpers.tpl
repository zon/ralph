{{- define "ralph-launcher.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "ralph-launcher.fullname" -}}
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

{{- define "ralph-launcher.labels" -}}
helm.sh/chart: {{ printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" }}
app.kubernetes.io/name: {{ include "ralph-launcher.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end -}}

{{- define "ralph-launcher.selectorLabels" -}}
app.kubernetes.io/name: {{ include "ralph-launcher.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}

{{- define "ralph-launcher.natsAuthSecretName" -}}
{{- required "nats.auth.secret.name is required" .Values.nats.auth.secret.name -}}
{{- end -}}

{{- define "ralph-launcher.natsAuthUsernameKey" -}}
{{- default "username" .Values.nats.auth.secret.usernameKey -}}
{{- end -}}

{{- define "ralph-launcher.natsAuthPasswordKey" -}}
{{- default "password" .Values.nats.auth.secret.passwordKey -}}
{{- end -}}

{{- define "ralph-launcher.subject" -}}
{{- if .Values.nats.subject -}}
{{- .Values.nats.subject -}}
{{- else -}}
{{- printf "ralph.repo.%s.%s" (.Values.repo.owner | lower | replace "." "-" | replace "_" "-") (.Values.repo.name | lower | replace "." "-" | replace "_" "-") -}}
{{- end -}}
{{- end -}}

{{- define "ralph-launcher.validatePromptEntries" -}}
{{- $entries := required "prompt.entries is required" .Values.prompt.entries -}}
{{- $slugs := dict -}}
{{- range $entry := $entries -}}
{{- $slug := required "prompt.entries[].slug is required" $entry.slug -}}
{{- if hasKey $slugs $slug -}}
{{- fail (printf "duplicate prompt entry slug %q" $slug) -}}
{{- end -}}
{{- $_ := set $slugs $slug true -}}
{{- $_ := required (printf "prompt.entries[%s].text is required" $slug) $entry.text -}}
{{- $_ := include "ralph-launcher.entrySchedules" $entry | fromJsonArray -}}
{{- end -}}
{{- end -}}

{{- define "ralph-launcher.entrySchedules" -}}
{{- $schedules := list -}}
{{- with .schedule -}}
{{- if kindIs "string" . -}}
{{- $schedules = list . -}}
{{- else if kindIs "slice" . -}}
{{- $schedules = . -}}
{{- else -}}
{{- fail (printf "prompt entry %q schedule must be one cron expression or a list of expressions" $.slug) -}}
{{- end -}}
{{- end -}}
{{- toJson $schedules -}}
{{- end -}}

{{- define "ralph-launcher.entryEvents" -}}
{{- $events := list "push" -}}
{{- if hasKey . "events" -}}
{{- $events = .events -}}
{{- else if include "ralph-launcher.entrySchedules" . | fromJsonArray -}}
{{- $events = list -}}
{{- end -}}
{{- if and (hasKey . "events") (not $events) -}}
{{- $events = list "" -}}
{{- end -}}
{{- toJson $events -}}
{{- end -}}

{{- define "ralph-launcher.scheduleTimezone" -}}
{{- $schedule := default dict .Values.schedule -}}
{{- default "" $schedule.timezone -}}
{{- end -}}

{{- define "ralph-launcher.safeName" -}}
{{- . | lower | replace "/" "-" | replace "_" "-" | replace "." "-" | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "ralph-launcher.entryName" -}}
{{- $name := .slug -}}
{{- if .event -}}
{{- $name = printf "%s-%s" .slug .event -}}
{{- end -}}
{{- include "ralph-launcher.safeName" $name -}}
{{- end -}}

{{- define "ralph-launcher.mutexName" -}}
{{- $root := .root -}}
{{- if $root.Values.workflow.mutex.name -}}
{{- $root.Values.workflow.mutex.name -}}
{{- else -}}
{{- include "ralph-launcher.safeName" .entry.slug -}}
{{- end -}}
{{- end -}}

{{- define "ralph-launcher.sensorName" -}}
{{- printf "%s-%s" (include "ralph-launcher.fullname" .) .Release.Namespace | replace "." "-" | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "ralph-launcher.eventSourceName" -}}
{{- default (printf "%s-events" (include "ralph-launcher.sensorName" .)) .Values.sensor.eventSource.name -}}
{{- end -}}

{{- define "ralph-launcher.serviceAccountName" -}}
{{- default (include "ralph-launcher.fullname" .) .Values.sensor.serviceAccount.name -}}
{{- end -}}

{{- define "ralph-launcher.pushRef" -}}
{{- if .Values.sensor.branch -}}
{{- .Values.sensor.branch -}}
{{- else -}}
{{- printf "refs/heads/%s" .Values.repo.baseBranch -}}
{{- end -}}
{{- end -}}

{{- define "ralph-launcher.workflowLabels" -}}
app.kubernetes.io/managed-by: ralph
app.kubernetes.io/name: {{ include "ralph-launcher.name" . | quote }}
app.kubernetes.io/instance: {{ .Release.Name | quote }}
{{ with .Values.workflow.labels -}}
{{ toYaml . }}
{{- end }}
{{- end -}}

{{- define "ralph-launcher.workflowSpec" -}}
{{- $root := .root -}}
{{- $entry := .entry -}}
workflowTemplateRef:
  name: {{ include "ralph-launcher.fullname" $root }}
arguments:
  parameters:
    {{- include "ralph-launcher.workflowParameters" (dict "root" $root "entry" $entry) | nindent 4 }}
ttlStrategy:
  secondsAfterCompletion: {{ $root.Values.workflow.ttlSecondsAfterCompletion }}
podGC:
  strategy: {{ $root.Values.workflow.podGC.strategy | quote }}
  deleteDelayDuration: {{ $root.Values.workflow.podGC.deleteDelayDuration | quote }}
{{- if $root.Values.workflow.mutex.enabled }}
synchronization:
  mutexes:
    - name: {{ include "ralph-launcher.mutexName" (dict "root" $root "entry" $entry) | quote }}
{{- end }}
{{- end -}}

{{- define "ralph-launcher.workflowParameters" -}}
{{- $root := .root -}}
{{- $entry := .entry -}}
- name: repo
  value: {{ printf "%s/%s" $root.Values.repo.owner $root.Values.repo.name | quote }}
- name: clone-branch
  value: {{ $root.Values.repo.cloneBranch | quote }}
- name: prefix
  value: {{ $entry.slug | quote }}
- name: base-branch
  value: {{ $root.Values.repo.baseBranch | quote }}
- name: prompt
  valueFrom:
    configMapKeyRef:
      name: {{ include "ralph-launcher.fullname" $root }}-prompt
      key: {{ $entry.slug | quote }}
- name: model
  value: {{ $root.Values.workflow.model | quote }}
- name: variant
  value: {{ $root.Values.workflow.variant | quote }}
- name: agent
  value: {{ $root.Values.workflow.agent | quote }}
- name: no-services
  value: {{ $root.Values.workflow.noServices | quote }}
- name: verbose
  value: {{ $root.Values.workflow.verbose | quote }}
{{- end -}}

{{- define "ralph-launcher.dependencyFilters" -}}
{{- $root := .root -}}
{{- $event := .event -}}
{{- $pushBranch := and (eq $event "push") $root.Values.sensor.defaultBranchOnly -}}
{{- if or $event $pushBranch $root.Values.sensor.ignoreUsers $root.Values.sensor.filters -}}
filters:
  data:
    {{- if $event }}
    - path: body.event
      type: string
      value:
        - {{ $event | quote }}
    {{- end }}
    {{- if $pushBranch }}
    - path: body.payload.ref
      type: string
      value:
        - {{ include "ralph-launcher.pushRef" $root | quote }}
    {{- end }}
    {{- range $root.Values.sensor.ignoreUsers }}
    - path: body.payload.sender.login
      type: string
      comparator: "!="
      value:
        - {{ . | quote }}
    {{- end }}
    {{- with $root.Values.sensor.filters }}
    {{- toYaml . | nindent 4 }}
    {{- end }}
{{- end }}
{{- end -}}
