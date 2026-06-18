# TOPF Example Cluster

This directory contains a complete example TOPF cluster configuration. It demonstrates a multi-node Talos cluster with layered patches, templates, and per-node overrides.

## Layout

```text
.
├── topf.yaml                     # Main cluster configuration
├── manifest.yaml.tpl             # Schematic definition (referenced by schematicId in topf.yaml)
├── all/                          # Patches applied to every node
│   ├── 00-install-disk.yaml      # Install disk
│   ├── 01-base.yaml.tpl          # Common labels and kubelet settings
│   ├── 05-hostname.yaml.tpl      # Use nodes[].host as Talos hostname
│   ├── 09-logging.yaml.tpl       # Optional remote logging (env-driven)
│   └── 10-registry.yaml.tpl      # Optional registry mirror (env-driven)
├── control-plane/                # Patches applied to control-plane nodes
│   └── 01-base.yaml              # Talos API access and discovery settings
├── node/                         # Per-node patches
│   └── node5/
│       └── 00-local-storage.yaml.tpl  # Storage configuration for node5
```

## Important: hostnames

The `host` value in `topf.yaml` is used by TOPF for display, logging, and node selection. It is **not** automatically applied as the Talos hostname. Talos defaults to auto-generated hostnames such as `talos-XXX-XXX` unless you override them.

The patch `all/05-hostname.yaml.tpl` sets the Talos hostname from `nodes[].host`:

```yaml
apiVersion: v1alpha1
kind: HostnameConfig
auto: "off"
hostname: {{ .Node.Host }}
```

If you want Kubernetes node names to match the `host` values, keep this patch in your configuration.

## Render the example

To render machine configs without applying them:

```bash
cd example/
topf render
```

The generated files in `output/` are ignored by git. Do not commit them.

## Apply the example

This example uses placeholder IPs and secrets. Before applying, update `topf.yaml`, the patches, and generate real secrets with `topf secrets`.
