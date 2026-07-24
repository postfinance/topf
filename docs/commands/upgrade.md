# Upgrade Command

The `upgrade` command upgrades Talos OS on each node to the desired version specified in the installer image.

## Flags

All flags can also be set via environment variables using the `TOPF_` prefix and uppercasing the flag name (e.g. `--reboot-mode` → `TOPF_REBOOT_MODE`).

| Flag | Default | Description |
|------|---------|-------------|
| `--dry-run` | `false` | Only show what upgrades would be performed without actually upgrading |
| `--max-parallel` | `1` | Number of worker nodes to upgrade concurrently, as an integer (e.g. `5`) or a percentage of the total node count (e.g. `25%`); control-plane nodes are always upgraded one at a time |
| `--reboot-mode` | `default` | Reboot mode during upgrade: `default` uses kexec, `powercycle` does a full reboot |
| `--drain` | `true` | Cordon and drain the Kubernetes node before rebooting, then uncordon after stabilization |
| `--drain-timeout` | `5m` | Maximum time to wait for pod evictions to complete during drain |
| [`--nodes-filter`](../configuration.md#filtering-nodes) | - | Regex pattern to filter which nodes to operate on (global flag) |

> **Removed:** The `--force` flag (which skipped etcd health checks under the
> legacy `MachineService.Upgrade` RPC) has been removed. The new
> `LifecycleService.Upgrade` API does not expose a force knob; etcd health is
> validated server-side. Scripts or env files setting `TOPF_FORCE` should be
> updated, otherwise the flag will be rejected as unknown.

## Behavior

1. **Pre-flight checks**: Ensures all nodes are in the `Running` stage
2. **Version comparison**: Extracts schematic and version from the installer image and only upgrades nodes where either differs from the current state
3. **Per-node confirmation**: Before each upgrade (unless `--confirm=false`, see [global flags](../configuration.md#global-flags))
4. **Image pull**: Pre-pulls the installer image via the `ImageService.Pull` streaming RPC
5. **Upgrade**: Installs the upgrade artifacts via the `LifecycleService.Upgrade` streaming RPC
6. **Drain**: Cordons and drains the Kubernetes node (evicts pods gracefully) if `--drain` is enabled
7. **Reboot**: Issues a separate `Reboot` with the selected reboot mode (default: kexec)
8. **Stabilization**: Waits 30 seconds after the reboot for the node to stabilize
9. **Uncordon**: Uncordons the Kubernetes node so the scheduler can place pods on it again

## Installer Image

### Using `talosVersion` and `schematicId` (recommended)

To override the installer image for the entire cluster, set `talosVersion` and optionally `schematicId` in `topf.yaml`. Topf will automatically generate a cluster-level patch:

```yaml
talosVersion: 1.12.7
schematicId: 376567988ad370138ad8b2698212367b8edcb69b5fd68c80be1f2ec7d603b4ba
```

This generates `factory.talos.dev/metal-installer/<schematicId>:v<talosVersion>` as the base installer image. Since this patch is applied first, any subsequent `machine.install.image` patch (shared or node-level) will override it. The factory and platform can be customized via `factory` and `platform` in `topf.yaml` (or per node).

### Manual installer image patch

Alternatively, manage the installer image explicitly via a patch. The target image for each node comes from the `machine.install.image` field in the assembled node configuration (i.e. the last patch takes precedence). The patch looks like:

`all/00-install.yaml`:

```yaml
machine:
  install:
    image: factory.talos.dev/metal-installer/376567988ad370138ad8b2698212367b8edcb69b5fd68c80be1f2ec7d603b4ba:v1.12.0
```

### Per-node override

To upgrade a single node to a different version or schematic, add a node-specific patch that overrides the image:

`node/node1/installer.yaml`:

```yaml
machine:
  install:
    image: factory.talos.dev/metal-installer/376567988ad370138ad8b2698212367b8edcb69b5fd68c80be1f2ec7d603b4ba:v1.13.0
```

Because node-level patches are merged last (see [Configuration Model](../configuration-model.md)), this override applies only to that host.

## Example Usage

```bash
# Upgrade with confirmation (default)
topf upgrade

# Upgrade without confirmation
topf upgrade --confirm=false

# Preview what would be upgraded
topf upgrade --dry-run

# Upgrade without draining the Kubernetes node
topf upgrade --drain=false

# Upgrade up to 3 worker nodes concurrently
topf upgrade --max-parallel=3

# Upgrade with a custom drain timeout
topf upgrade --drain-timeout=10m
```
