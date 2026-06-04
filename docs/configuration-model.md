# Configuration Model

The final machine configuration for each node is assembled from a series of configuration documents (patches) which are merged together. This is similar to how `talosctl` applies `--config-patch` flags — each patch is a partial Talos machine configuration that gets strategically merged into the result.

## Merge Order

Patches are applied in the following order:

1. **`all/`** — applied to all nodes
2. **`<role>/`** — applied to nodes matching the role (`control-plane` or `worker`)
3. **`node/<host>/`** — applied only to that specific node

!!! tip
    The per-host directory is `node/` (singular), not `nodes/` as each `<host>/` subfolder targets a single node at a time.

Within each folder, patches are applied in **lexicographical order**. Later patches take precedence over earlier ones when the same field is set. This layering is useful for per-node overrides — for example, pinning a different [installer image](commands/upgrade.md#installer-image) on a single host.

A typical cluster folder looks like this:

```text
.
├── all
│   └── 01-installation.yaml
├── control-plane
│   ├── 01-vip.yaml
│   ├── 02-disable-discovery.yaml
│   └── 03-allow-cp-scheduling.yaml
├── worker
│   └── 01-worker-taints.yaml
├── node
│   └── node1
│       └── 01-some-nodespecific-patch.yaml
└── topf.yaml
```

!!! tip
    Prefix patch filenames with a number (e.g. `01-`, `02-`) to make the merge order explicit and predictable.

## Patch Formats

Patches can be provided in several formats:

| Extension            | Format                             |
| -------------------- | ---------------------------------- |
| `.yaml` / `.yml`     | Strategic merge patch              |
| `.yaml.tpl` / `.yml.tpl` | Go-templated strategic merge patch |

!!! warning
    JSON patches (RFC 6902) are **not supported**. They have been deprecated in Talos starting from v1.12.

Empty patches (comments only, whitespace, `{}`, `[]`, `null`) are automatically skipped.

## Secret Resolution

TOPF reads all non-template files (including `topf.yaml` itself) through a two-stage pipeline:

1. **SOPS decryption** — if a file is SOPS-encrypted, it is decrypted automatically. If SOPS is not installed, unencrypted files are read as-is.
2. **vals evaluation** — after decryption, any [vals](https://github.com/helmfile/vals) references (e.g. `ref+vault://`, `ref+file://`) are resolved. If no vals references are present, this step is skipped. The `vals` binary must be on `PATH` when vals references are used.

Template files (ending with `.tpl`) skip this pipeline entirely and are rendered through [Go templates](#templating) instead.

This is useful for keeping sensitive values (e.g. private keys, credentials, tokens) out of version control — either by encrypting the entire file with SOPS, or by referencing secrets from an external store via [vals](https://github.com/helmfile/vals).

## Templating

Patches ending with `.yaml.tpl` support [Go templating](https://pkg.go.dev/text/template). The following context fields are available:

| Field | Description |
|-------|-------------|
| `.ClusterName` | Cluster name from `topf.yaml` |
| `.ClusterEndpoint` | Cluster endpoint URL |
| `.KubernetesVersion` | Kubernetes version |
| `.TalosVersion` | Talos version (if set in `topf.yaml`) |
| `.SchematicID` | Schematic ID (if set in `topf.yaml`) |
| `.Data.<key>` | Arbitrary global data from `topf.yaml` (see [configuration](configuration.md#configuration-fields)) |
| `.Node.Host` | Node hostname |
| `.Node.Role` | Node role (`control-plane` or `worker`) |
| `.Node.IP` | Node IP address (if set) |
| `.Node.Data.<key>` | Per-node data (if set) |

### Template Functions

In addition to the [built-in Go template functions](https://pkg.go.dev/text/template#hdr-Functions), the full [sprig](https://masterminds.github.io/sprig/) function library is available. This provides `env`, `default`, `b64enc`/`b64dec`, `toYaml`, `indent`/`nindent`, `regexReplaceAll`, `trunc`, `trimAll`, and many more.

A few commonly used functions:

| Function | Description |
|----------|-------------|
| `env "VAR"` | Returns the value of the environment variable `VAR`, or an empty string if unset |
| `default "x" .Val` | Returns `.Val`, falling back to `"x"` if `.Val` is empty |
| `b64enc` / `b64dec` | Base64 encode / decode |

### Examples

Use per-node data in a patch:

```yaml
machine:
  kubelet:
    extraArgs:
      provider-id: {{ .Node.Data.uuid }}
```

Use the `env` function in a multi-document template to conditionally configure a registry mirror:

```yaml
{{ if env "REGISTRY_MIRROR" -}}
apiVersion: v1alpha1
kind: RegistryMirrorConfig
name: ghcr.io
endpoints:
  - url: {{ env "REGISTRY_MIRROR" }}
{{- end }}
```
