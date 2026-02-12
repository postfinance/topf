# Configuration Model

The final machine configuration for each node is assembled from a series of configuration documents (patches) which are merged together. This is similar to how `talosctl` applies `--config-patch` flags — each patch is a partial Talos machine configuration that gets strategically merged into the result.

## Merge Order

Patches are applied in the following order:

1. **`patches/`** — applied to all nodes
2. **`<role>/`** — applied to nodes matching the role (`control-plane` or `worker`)
3. **`nodes/<host>/`** — applied only to that specific node

Within each folder, patches are applied in **lexicographical order**. Later patches take precedence over earlier ones when the same field is set. This layering is useful for per-node overrides — for example, pinning a different [installer image](commands/upgrade.md#installer-image) on a single host.

A typical cluster folder looks like this:

```text
.
├── control-plane
│   ├── 01-vip.yaml
│   ├── 02-disable-discovery.yaml
│   └── 03-allow-cp-scheduling.yaml
├── worker
│   └── 01-worker-taints.yaml
├── nodes
│   └── node1
│       └── 01-some-nodespecific-patch.yaml
├── patches
│   └── 01-installation.yaml
└── topf.yaml
```

!!! tip
Prefix patch filenames with a number (e.g. `01-`, `02-`) to make the merge order explicit and predictable.

## Patch Formats

Patches can be provided in several formats:

| Extension   | Format                               |
| ----------- | ------------------------------------ |
| `.yaml`     | Strategic merge patch                |
| `.yaml.tpl` | Go-templated strategic merge patch   |
| `.enc.yaml` | SOPS-encrypted strategic merge patch |

!!! warning
JSON patches (RFC 6902) are **not supported**. They have been deprecated in Talos starting from v1.12.

Empty patches (comments only, whitespace, `{}`, `[]`, `null`) are automatically skipped.

## Encryption

TOPF attempts to decrypt every file it loads — both `topf.yaml` and all patch files — using [SOPS](https://github.com/getsops/sops). If a file is not SOPS-encrypted, it is loaded as-is. This means any patch file can be encrypted, except template files (ending with `.tpl`).

This is useful for patches that contain sensitive values (e.g. private keys, credentials, tokens) that should not be stored in plaintext and for which you don't want to write a template.

## Templating

Patches ending with `.yaml.tpl` support Go templating. The following fields are available:

- `.ClusterName`
- `.ClusterEndpoint`
- `.Data.<key>` — arbitrary global data from `topf.yaml` (see [configuration](configuration.md#configuration-fields))
- `.Node.Host`
- `.Node.Role`
- `.Node.IP` (if set)
- `.Node.Data.<key>` — per-node data (if set)

Example:

```yaml
machine:
  kubelet:
    extraArgs:
      provider-id: { { .Node.Data.uuid } }
```
