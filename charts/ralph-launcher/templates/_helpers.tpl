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

{{- define "ralph-launcher.delivery" -}}
{{- include "ralph-launcher.validateDelivery" . -}}
{{- default "subject" .Values.delivery -}}
{{- end -}}

{{- define "ralph-launcher.validateDelivery" -}}
{{- $delivery := default "subject" .Values.delivery -}}
{{- if not (has $delivery (list "subject" "bus")) -}}
{{- fail (printf "delivery must be subject or bus, got %v" $delivery) -}}
{{- end -}}
{{- if eq $delivery "bus" -}}
{{- $intake := default dict .Values.intake -}}
{{- $eventSource := default dict $intake.eventSource -}}
{{- $_ := required "intake.eventSource.name is required when delivery is bus" $eventSource.name -}}
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

{{- define "ralph-launcher.hasTargets" -}}
{{- $found := false -}}
{{- range .Values.prompt.entries -}}
{{- if .namespace -}}
{{- $found = true -}}
{{- end -}}
{{- end -}}
{{- if $found -}}true{{- end -}}
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

{{- define "ralph-launcher.dependencyEventSourceName" -}}
{{- if eq (include "ralph-launcher.delivery" .) "bus" -}}
{{- .Values.intake.eventSource.name -}}
{{- else -}}
{{- include "ralph-launcher.eventSourceName" . -}}
{{- end -}}
{{- end -}}

{{- define "ralph-launcher.dependencyEventName" -}}
{{- if eq (include "ralph-launcher.delivery" .) "bus" -}}
{{- default "github" .Values.intake.eventName -}}
{{- else -}}
repo
{{- end -}}
{{- end -}}

{{- define "ralph-launcher.serviceAccountName" -}}
{{- default (include "ralph-launcher.fullname" .) .Values.sensor.serviceAccount.name -}}
{{- end -}}

{{- define "ralph-launcher.workflowServiceAccountName" -}}
{{- default "default" .Values.workflow.serviceAccountName -}}
{{- end -}}

{{- define "ralph-launcher.targetNamespaces" -}}
{{- $namespaces := dict -}}
{{- range .Values.prompt.entries -}}
{{- if .namespace -}}
{{- $_ := set $namespaces .namespace true -}}
{{- end -}}
{{- end -}}
{{- toJson (keys $namespaces | sortAlpha) -}}
{{- end -}}

{{- define "ralph-launcher.promptTemplate" -}}
{{- $root := .root -}}
{{- $target := .target -}}
{{- $repo := "{{inputs.parameters.repo}}" -}}
{{- $cloneBranch := "{{inputs.parameters.clone-branch}}" -}}
{{- $prefix := "{{inputs.parameters.prefix}}" -}}
{{- $baseBranch := "{{inputs.parameters.base-branch}}" -}}
{{- $prompt := "{{inputs.parameters.prompt}}" -}}
- name: {{ if $target }}ralph-prompt-target{{ else }}ralph-prompt{{ end }}
  {{- with $root.Values.workflow.labels }}
  metadata:
    labels:
      {{- toYaml . | nindent 6 }}
  {{- end }}
  inputs:
    parameters:
      - name: repo
      - name: clone-branch
      - name: prefix
      - name: base-branch
      - name: prompt
      {{- if $target }}
      - name: namespace
      {{- end }}
      - name: model
      - name: variant
      - name: agent
      - name: no-services
      - name: verbose
  {{- if $root.Values.workflow.serviceAccountName }}
  serviceAccountName: {{ $root.Values.workflow.serviceAccountName | quote }}
  {{- end }}
  container:
    image: "{{ $root.Values.image.repository }}:{{ $root.Values.image.tag | default $root.Chart.AppVersion }}"
    imagePullPolicy: {{ $root.Values.image.pullPolicy }}
    command:
      - ralph
    args:
      - workflow
      - prompt
      - --repo
      - {{ $repo | quote }}
      - --clone
      - {{ $cloneBranch | quote }}
      - --prefix
      - {{ $prefix | quote }}
      - --base
      - {{ $baseBranch | quote }}
      - --prompt
      - {{ $prompt | quote }}
      {{- if $target }}
      - --namespace
      - {{ "{{inputs.parameters.namespace}}" | quote }}
      {{- end }}
      {{- if $root.Values.workflow.model }}
      - --model
      - {{ "{{inputs.parameters.model}}" | quote }}
      {{- end }}
      {{- if $root.Values.workflow.variant }}
      - --variant
      - {{ "{{inputs.parameters.variant}}" | quote }}
      {{- end }}
      {{- if $root.Values.workflow.agent }}
      - --agent
      - {{ "{{inputs.parameters.agent}}" | quote }}
      {{- end }}
      {{- if $root.Values.workflow.noServices }}
      - --no-services
      {{- end }}
      {{- if $root.Values.workflow.verbose }}
      - --verbose
      {{- end }}
    {{- with $root.Values.workflow.env }}
    env:
      {{- toYaml . | nindent 6 }}
    {{- end }}
    resources:
      {{- toYaml $root.Values.workflow.resources | nindent 6 }}
    {{- if or $root.Values.workflow.credentials.githubSecret $root.Values.workflow.credentials.opencodeSecret $root.Values.workflow.extraVolumeMounts }}
    volumeMounts:
      {{- if $root.Values.workflow.credentials.githubSecret }}
      - name: github-credentials
        mountPath: /secrets/github
        readOnly: true
      {{- end }}
      {{- if $root.Values.workflow.credentials.opencodeSecret }}
      - name: opencode-credentials
        mountPath: /secrets/opencode
        readOnly: true
      {{- end }}
      {{- with $root.Values.workflow.extraVolumeMounts }}
      {{- toYaml . | nindent 6 }}
      {{- end }}
    {{- end }}
  {{- if or $root.Values.workflow.credentials.githubSecret $root.Values.workflow.credentials.opencodeSecret $root.Values.workflow.extraVolumes }}
  volumes:
    {{- if $root.Values.workflow.credentials.githubSecret }}
    - name: github-credentials
      secret:
        secretName: {{ $root.Values.workflow.credentials.githubSecret | quote }}
    {{- end }}
    {{- if $root.Values.workflow.credentials.opencodeSecret }}
    - name: opencode-credentials
      secret:
        secretName: {{ $root.Values.workflow.credentials.opencodeSecret | quote }}
    {{- end }}
    {{- with $root.Values.workflow.extraVolumes }}
    {{- toYaml . | nindent 4 }}
    {{- end }}
  {{- end }}
{{- end -}}

{{- define "ralph-launcher.pushRef" -}}
{{- if .Values.sensor.branch -}}
{{- .Values.sensor.branch -}}
{{- else -}}
{{- printf "refs/heads/%s" .Values.repo.baseBranch -}}
{{- end -}}
{{- end -}}

{{- define "ralph-launcher.workflowSettings" -}}
{{- $root := .root -}}
{{- $entry := .entry -}}
{{ .metadataKey }}:
  {{- with .generateName }}
  generateName: {{ . | quote }}
  {{- end }}
  labels:
    app.kubernetes.io/managed-by: ralph
    app.kubernetes.io/name: {{ include "ralph-launcher.name" $root | quote }}
    app.kubernetes.io/instance: {{ $root.Release.Name | quote }}
    {{- with $root.Values.workflow.labels }}
    {{- toYaml . | nindent 4 }}
    {{- end }}
{{ .specKey }}:
  {{- if $entry.namespace }}
  entrypoint: ralph-prompt-target
  {{- end }}
  workflowTemplateRef:
    name: {{ include "ralph-launcher.fullname" $root }}
  arguments:
    parameters:
      {{- include "ralph-launcher.workflowParameters" (dict "root" $root "entry" $entry) | nindent 6 }}
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
{{- if $entry.namespace }}
- name: namespace
  value: {{ $entry.namespace | quote }}
{{- end }}
{{- end -}}

{{- define "ralph-launcher.dependencyFilters" -}}
{{- $root := .root -}}
{{- $event := .event -}}
{{- $bus := eq (include "ralph-launcher.delivery" $root) "bus" -}}
{{- $pushBranch := and (eq $event "push") $root.Values.sensor.defaultBranchOnly -}}
{{- if or $bus $event $pushBranch $root.Values.sensor.ignoreUsers $root.Values.sensor.filters -}}
filters:
  data:
    {{- if $bus }}
    - path: body.repository.full_name
      type: string
      value:
        - {{ printf "%s/%s" $root.Values.repo.owner $root.Values.repo.name | quote }}
    {{- end }}
    {{- if $event }}
    - path: {{ if $bus }}headers.X-Github-Event{{ else }}body.event{{ end }}
      type: string
      value:
        - {{ $event | quote }}
    {{- end }}
    {{- if $pushBranch }}
    - path: {{ if $bus }}body.ref{{ else }}body.payload.ref{{ end }}
      type: string
      value:
        - {{ include "ralph-launcher.pushRef" $root | quote }}
    {{- end }}
    {{- range $root.Values.sensor.ignoreUsers }}
    - path: {{ if $bus }}body.sender.login{{ else }}body.payload.sender.login{{ end }}
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
