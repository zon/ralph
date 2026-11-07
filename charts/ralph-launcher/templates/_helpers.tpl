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

{{- define "ralph-launcher.subject" -}}
{{- if .Values.nats.subject -}}
{{- .Values.nats.subject -}}
{{- else -}}
{{- printf "ralph.repo.%s.%s" (.Values.repo.owner | lower | replace "." "-" | replace "_" "-") (.Values.repo.name | lower | replace "." "-" | replace "_" "-") -}}
{{- end -}}
{{- end -}}

{{- define "ralph-launcher.promptBranch" -}}
{{- if .Values.prompt.branch -}}
{{- .Values.prompt.branch -}}
{{- else -}}
{{- printf "ralph-prompt-%s" .Release.Namespace -}}
{{- end -}}
{{- end -}}

{{- define "ralph-launcher.mutexName" -}}
{{- if .Values.workflow.mutex.name -}}
{{- .Values.workflow.mutex.name -}}
{{- else -}}
{{- include "ralph-launcher.promptBranch" . | lower | replace "/" "-" | replace "_" "-" | replace "." "-" | trunc 63 | trimSuffix "-" -}}
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

{{- define "ralph-launcher.workflowParameters" -}}
- name: repo
  value: {{ printf "%s/%s" .Values.repo.owner .Values.repo.name | quote }}
- name: clone-branch
  value: {{ .Values.repo.cloneBranch | quote }}
- name: prompt-branch
  value: {{ include "ralph-launcher.promptBranch" . | quote }}
- name: base-branch
  value: {{ .Values.repo.baseBranch | quote }}
- name: prompt
  valueFrom:
    configMapKeyRef:
      name: {{ include "ralph-launcher.fullname" . }}-prompt
      key: {{ .Values.prompt.configMapKey | quote }}
- name: model
  value: {{ .Values.workflow.model | quote }}
- name: variant
  value: {{ .Values.workflow.variant | quote }}
- name: agent
  value: {{ .Values.workflow.agent | quote }}
- name: no-services
  value: {{ .Values.workflow.noServices | quote }}
- name: verbose
  value: {{ .Values.workflow.verbose | quote }}
{{- end -}}
