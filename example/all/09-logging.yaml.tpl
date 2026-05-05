{{ if env "LOG_ENDPOINT" -}}
machine:
  logging:
    destinations:
      - endpoint: {{ env "LOG_ENDPOINT" }}
        format: json_lines
        extraTags:
          node: {{ .Node.Host }}
{{- end }}
