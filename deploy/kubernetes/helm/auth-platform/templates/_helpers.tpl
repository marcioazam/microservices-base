{{/*
Expand the name of the chart.
*/}}
{{- define "auth-platform.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "auth-platform.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "auth-platform.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "auth-platform.labels" -}}
helm.sh/chart: {{ include "auth-platform.chart" . }}
{{ include "auth-platform.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/part-of: auth-platform
{{- end }}

{{/*
Selector labels
*/}}
{{- define "auth-platform.selectorLabels" -}}
app.kubernetes.io/name: {{ include "auth-platform.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "auth-platform.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "auth-platform.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Validate image tag is not 'latest' in production
*/}}
{{- define "auth-platform.validateImageTag" -}}
{{- $env := .Values.global.environment | default "development" -}}
{{- if and (eq $env "production") (or (eq .tag "latest") (empty .tag)) -}}
{{- fail (printf "Image tag cannot be 'latest' or empty in production environment. Service: %s" .service) -}}
{{- end -}}
{{- .tag | default "latest" -}}
{{- end }}

{{/*
Pod security context
*/}}
{{- define "auth-platform.podSecurityContext" -}}
runAsNonRoot: true
runAsUser: 65534
runAsGroup: 65534
fsGroup: 65534
seccompProfile:
  type: RuntimeDefault
{{- end }}

{{/*
Container security context
*/}}
{{- define "auth-platform.containerSecurityContext" -}}
allowPrivilegeEscalation: false
readOnlyRootFilesystem: true
capabilities:
  drop:
    - ALL
{{- end }}

{{/*
Common probe configuration
*/}}
{{- define "auth-platform.livenessProbe" -}}
{{- if .grpc }}
grpc:
  port: {{ .port }}
{{- else }}
httpGet:
  path: {{ .path | default "/health" }}
  port: {{ .port }}
{{- end }}
initialDelaySeconds: {{ .initialDelaySeconds | default 15 }}
periodSeconds: {{ .periodSeconds | default 20 }}
timeoutSeconds: {{ .timeoutSeconds | default 5 }}
failureThreshold: {{ .failureThreshold | default 3 }}
{{- end }}

{{- define "auth-platform.readinessProbe" -}}
{{- if .grpc }}
grpc:
  port: {{ .port }}
{{- else }}
httpGet:
  path: {{ .path | default "/ready" }}
  port: {{ .port }}
{{- end }}
initialDelaySeconds: {{ .initialDelaySeconds | default 5 }}
periodSeconds: {{ .periodSeconds | default 10 }}
timeoutSeconds: {{ .timeoutSeconds | default 3 }}
failureThreshold: {{ .failureThreshold | default 3 }}
{{- end }}

{{- define "auth-platform.startupProbe" -}}
{{- if .grpc }}
grpc:
  port: {{ .port }}
{{- else }}
httpGet:
  path: {{ .path | default "/health" }}
  port: {{ .port }}
{{- end }}
initialDelaySeconds: {{ .initialDelaySeconds | default 5 }}
periodSeconds: {{ .periodSeconds | default 5 }}
timeoutSeconds: {{ .timeoutSeconds | default 3 }}
failureThreshold: {{ .failureThreshold | default 30 }}
{{- end }}

{{/*
Pod anti-affinity
*/}}
{{- define "auth-platform.podAntiAffinity" -}}
podAntiAffinity:
  preferredDuringSchedulingIgnoredDuringExecution:
    - weight: 100
      podAffinityTerm:
        labelSelector:
          matchLabels:
            app: {{ .app }}
        topologyKey: kubernetes.io/hostname
{{- end }}
