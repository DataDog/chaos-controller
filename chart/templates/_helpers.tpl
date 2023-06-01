
{{/* Generate a full container image name from an image context (repo/name/tag) */}}
{{/* To provide default values and partial override, do something similar to :
{{/* - {{ template "chaos-controller.format-image" deepCopy .Values.global.chaos.defaultImage | merge .Values.global.oci | merge .Values.controller.image) }} */}}
{{/* This will clone defaultImage dictionary and overwrite any existing values into it by values existing into .Values.global.oci then .Values.controller.image */}}
{{- define "chaos-controller.format-image" -}}
{{ .registry }}/{{ .repo }}{{ (not (empty .digest)) | ternary (printf "@%s" .digest) (printf ":%s" .tag) }}
{{- end }}
