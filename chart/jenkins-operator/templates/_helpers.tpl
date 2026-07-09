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
Returns "true" when the operator should watch all namespaces (WATCH_NAMESPACE="").
This happens when the bundled Jenkins is enabled with an empty jenkins.namespace, or
when jenkins is disabled and operator.watchNamespace is explicitly set to "".
An empty string is falsy in templates, so operator.watchNamespace is detected with
hasKey to distinguish "set to empty (all namespaces)" from "unset (own namespace)".
*/}}
{{- define "jenkins-operator.watchAllNamespaces" -}}
{{- if .Values.jenkins.enabled -}}
{{- if eq .Values.jenkins.namespace "" -}}true{{- end -}}
{{- else if hasKey .Values.operator "watchNamespace" -}}
{{- if eq .Values.operator.watchNamespace "" -}}true{{- end -}}
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
