{{- define "snakerule.name" -}}{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}{{- end -}}
{{- define "snakerule.fullname" -}}
{{- if .Values.fullnameOverride -}}{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" -}}
{{- else -}}{{- $name := default .Chart.Name .Values.nameOverride -}}{{- if contains $name .Release.Name -}}{{- .Release.Name | trunc 63 | trimSuffix "-" -}}{{- else -}}{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" -}}{{- end -}}{{- end -}}
{{- end -}}
{{- define "snakerule.labels" -}}
app.kubernetes.io/name: {{ include "snakerule.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
helm.sh/chart: {{ printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" }}
{{- end -}}
{{- define "snakerule.selectorLabels" -}}
app.kubernetes.io/name: {{ include "snakerule.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}
{{- define "snakerule.secretName" -}}
{{- default (printf "%s-backend" (include "snakerule.fullname" .)) .Values.backend.existingSecret -}}
{{- end -}}
{{- define "snakerule.secretChecksum" -}}
{{- if .Values.backend.existingSecret -}}
{{- .Values.backend.secretChecksum | default "external-secret" | sha256sum -}}
{{- else -}}
{{- include (print $.Template.BasePath "/secret.yaml") . | sha256sum -}}
{{- end -}}
{{- end -}}
{{- define "snakerule.nginxConfig" -}}
server {
  listen 80;
  server_name _;
  client_max_body_size 2m;
  location /api/ {
    proxy_pass http://{{ include "snakerule.fullname" . }}-backend:{{ .Values.backend.service.port }};
    proxy_http_version 1.1;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
  }
  location = /healthz {
    proxy_pass http://{{ include "snakerule.fullname" . }}-backend:{{ .Values.backend.service.port }}/healthz;
  }
  location / {
    proxy_pass http://{{ include "snakerule.fullname" . }}-admin:{{ .Values.admin.service.port }};
    proxy_set_header Host $host;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
  }
}
{{- end -}}
