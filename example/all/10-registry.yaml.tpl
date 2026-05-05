{{ if env "REGISTRY_MIRROR" -}}
apiVersion: v1alpha1
kind: RegistryMirrorConfig
name: ghcr.io
endpoints:
  - url: {{ env "REGISTRY_MIRROR" }}
{{- end }}
