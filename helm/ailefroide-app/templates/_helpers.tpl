{{/* vim: set filetype=mustache: */}}
{{/*
Expand the name of the chart.
*/}}
{{- define "helm.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "helm.fullname" -}}
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

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "helm.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Common labels
*/}}
{{- define "helm.labels" -}}
app.kubernetes.io/name: {{ include "helm.name" . }}
helm.sh/chart: {{ include "helm.chart" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
application.giantswarm.io/team: {{ index .Chart.Annotations "application.giantswarm.io/team" | quote }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end -}}

{{/*
Create the name of the service account to use
*/}}
{{- define "helm.serviceAccountName" -}}
{{- if .Values.serviceAccount.create -}}
    {{ default (include "helm.fullname" .) .Values.serviceAccount.name }}
{{- else }}
    {{ default "default" .Values.serviceAccount.name }}
{{- end -}}
{{- end -}}

{{- define "ailefroide.image" -}}
{{- if .Values.image.tag -}}
{{ .Values.image.repository }}:{{ .Values.image.tag }}
{{- else -}}
{{ .Values.image.repository }}
{{- end -}}
{{- end -}}

{{/*
Lookup secret values if the secret exists or create them from values file otherwise
*/}}
{{- define "ailefroide.secret" -}}
{{- $secret := lookup "v1" "Secret" .Release.Namespace .Values.envSecret.name -}}
{{- if $secret -}}
{{- range $k, $v := $secret.data }}
{{ $k }}: {{ $v }}
{{- end -}}
{{- else -}}
{{- range $k, $v := .Values.envSecret.values }}
{{- if $v }}
{{ $k }}: {{ printf "%s" $v | b64enc }}
{{- else }}
{{ $k }}: {{ printf "%q" "" }}
{{- end -}}
{{- end -}}
{{- end -}}
{{- end -}}

{{/*
Lookup configmap values if the configmap exists or create them from values file
*/}}
{{- define "ailefroide.config" -}}
{{- $config := lookup "v1" "Secret" .Release.Namespace .Values.configMap.name -}}
{{- if $config -}}
{{- range $k, $v := $config.data }}
{{ $k }}: {{ $v }}
{{- end -}}
{{- else -}}
{{- range $k, $v := .Values.configMap.values }}
{{- if $v }}
{{ $k }}: |
{{ $v | indent 2 }}
{{- end }}
{{- end -}}
{{- end -}}
{{- end -}}
