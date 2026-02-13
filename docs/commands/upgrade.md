# Upgrade Command

The `upgrade` command upgrades Talos OS on each node to the desired version specified in the installer image.

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--confirm` | `true` | Ask for user confirmation before upgrading |
| `--dry-run` | `false` | Only show what upgrades would be performed without actually upgrading |
| `--force` | `false` | Force the upgrade (skip checks on etcd health and members, might lead to data loss) |
| `--reboot-mode` | `default` | Reboot mode during upgrade: `default` uses kexec, `powercycle` does a full reboot |
| [`--nodes-filter`](../configuration.md#filtering-nodes) | - | Regex pattern to filter which nodes to operate on (global flag) |

## Behavior

1. **Pre-flight checks**: Ensures all nodes are in the `Running` stage
2. **Version comparison**: Extracts schematic and version from the installer image and only upgrades nodes where either differs from the current state
3. **Per-node confirmation**: Before each upgrade (unless `--confirm=false`)
4. **Upgrade**: Issues the upgrade command with the selected reboot mode (default: kexec)
5. **Stabilization**: Waits 30 seconds after upgrade for the node to stabilize

## Installer Image

The target image for each node comes from the `machine.install.image` field in the assembled node configuration. To upgrade all nodes, bump the tag in your shared install patch:

`all/01-installation.yaml`:

```yaml
machine:
  install:
    disk: /dev/vda
    image: factory.talos.dev/metal-installer/376567988ad370138ad8b2698212367b8edcb69b5fd68c80be1f2ec7d603b4ba:v1.12.0
```

To upgrade a single node to a different version (or a different schematic), add a node-specific patch that overrides the image:

`node/node1/installer.yaml`:

```yaml
machine:
  install:
    image: factory.talos.dev/metal-installer/376567988ad370138ad8b2698212367b8edcb69b5fd68c80be1f2ec7d603b4ba:v1.12.1
```

Because node-level patches are merged after global and role patches (see [Configuration Model](../configuration-model.md)), this override applies only to that host.

## Example Usage

```bash
# Upgrade with confirmation (default)
topf upgrade

# Upgrade without confirmation
topf upgrade --confirm=false

# Preview what would be upgraded
topf upgrade --dry-run

# Force upgrade (skip etcd health checks)
topf upgrade --force
```
