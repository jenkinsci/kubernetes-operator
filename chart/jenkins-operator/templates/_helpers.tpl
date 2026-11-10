{{/* vim: set filetype=mustache: */}}
{{/*
Expand the name of the chart.
*/}}
{{- define "jenkins-operator.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "jenkins-operator.fullname" -}}
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
{{- define "jenkins-operator.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Render a container image reference as <registry>/<repository>:<tag>.
Accepts a map with registry, repository and tag keys (registry is optional), or a plain
string which is returned unchanged for backwards compatibility.
Returns an empty string when no repository is set, so optional images can be omitted.
*/}}
{{- define "jenkins-operator.image" -}}
{{- if kindIs "string" . -}}
{{- . -}}
{{- else if and . .repository -}}
{{- $ref := .repository -}}
{{- if .registry -}}
{{- $ref = printf "%s/%s" .registry .repository -}}
{{- end -}}
{{- if .tag -}}
{{- $ref = printf "%s:%s" $ref (toString .tag) -}}
{{- end -}}
{{- $ref -}}
{{- end -}}
{{- end -}}

{{/*
Common labels
*/}}
{{- define "jenkins-operator.labels" -}}
app.kubernetes.io/name: {{ include "jenkins-operator.name" . }}
helm.sh/chart: {{ include "jenkins-operator.chart" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end -}}
