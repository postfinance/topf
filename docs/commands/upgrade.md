# Upgrade Command

The `upgrade` command upgrades Talos OS on each node to the desired version specified in the installer image.

## Flags

All flags can also be set via environment variables using the `TOPF_` prefix and uppercasing the flag name (e.g. `--reboot-mode` → `TOPF_REBOOT_MODE`).

| Flag | Default | Description |
|------|---------|-------------|
| `--dry-run` | `false` | Only show what upgrades would be performed without actually upgrading |
| `--max-parallel` | `1` | Number of worker nodes to upgrade concurrently, as an integer (e.g. `5`) or a percentage of the total node count (e.g. `25%`); control-plane nodes are always upgraded one at a time |
| `--reboot-mode` | `default` | Reboot mode during upgrade: `default` uses kexec, `powercycle` does a full reboot |
| `--drain` | `true` | Cordon and drain the Kubernetes node before rebooting, then uncordon after stabilization *(modern flow only; ignored on legacy nodes, where Talos drains and uncordons server-side)* |
| `--drain-timeout` | `5m` | Maximum time to wait for pod evictions (and, with `--delete-if-eviction-fails`, deletions) to complete during drain *(modern flow only)* |
| `--delete-if-eviction-fails` | `false` | If graceful drain fails (e.g. a PodDisruptionBudget blocks eviction), retry by deleting pods directly (DELETE instead of EVICT, bypassing PDBs); reuses `--drain-timeout` for the delete fallback *(modern flow only)* |
| `--force` | `false` | Skip etcd health checks; only applies to nodes running Talos < 1.13 (legacy `MachineService.Upgrade` RPC); has no effect on Talos >= 1.13, where the `LifecycleService.Upgrade` RPC validates etcd health server-side |
| `--stage` | `false` | Install upgrade artifacts without rebooting; the node is left running and can be labeled/annotated/tainted (see `--stage-label`/`--stage-annotation`/`--stage-taint`) so an external controller or human reboots it later |
| `--stage-label` | - | Kubernetes node label to apply after staging (`key=value`); can be repeated; requires `--stage` |
| `--stage-annotation` | - | Kubernetes node annotation to apply after staging (`key=value`); can be repeated; requires `--stage` |
| `--stage-taint` | - | Kubernetes node taint to apply after staging (`key=value:Effect`); can be repeated; requires `--stage` |
| [`--nodes-filter`](../configuration.md#filtering-nodes) | - | Regex pattern to filter which nodes to operate on (global flag) |

> **Upgrade API selection.** Nodes running Talos >= 1.13 use the modern
> `LifecycleService.Upgrade` streaming RPC (pre-pull, install, separate
> reboot). Nodes running Talos < 1.13 fall back to the legacy
> `MachineService.Upgrade` RPC, which installs, drains, and reboots in a
> single server-side sequence. The `--force` flag is only meaningful on
> the legacy path; `--drain` and `--drain-timeout` are only meaningful on
> the modern path.
>
> **When to use `--delete-if-eviction-fails`.** The graceful drain uses the
> Kubernetes eviction API, which respects PodDisruptionBudgets. A PDB with
> `minAvailable: 1` on a single-replica pod (e.g. a standalone database
> StatefulSet) will block eviction until `--drain-timeout` expires and the
> drain fails. `--delete-if-eviction-fails` retries the drain with direct
> pod deletion (DELETE instead of EVICT), bypassing PDBs — the pod is
> killed and its controller reschedules it elsewhere. The delete fallback
> reuses `--drain-timeout` as its timeout.
>
> Note that the drain always runs with `kubectl drain --force` semantics
> (unmanaged pods are deleted, emptyDir data is removed) — this is the
> default behavior for a rebooting node and is not controlled by
> `--delete-if-eviction-fails`. The flag only switches the eviction API to
> direct deletion for the fallback attempt, which is what lets it bypass
> PDBs.
>
> **Staging upgrades with `--stage`** *(Talos >= 1.13 only)*. Sometimes you
> want to install new Talos artifacts on nodes without immediately rebooting
> them — e.g. to spread reboots over a maintenance window or let an external
> controller (such as a drain scheduler) reboot nodes one at a time. `--stage`
> installs the upgrade artifacts but skips the drain, reboot, and uncordon
> steps. The node continues running on its current kernel until it is manually
> rebooted, at which point the staged upgrade takes effect.
>
> `--stage-label`, `--stage-annotation`, and `--stage-taint` mark the
> Kubernetes node so controllers or humans can identify nodes with a pending
> reboot. For example,
> `--stage-taint topf.postfinance.ch/staged-upgrade=true:PreferNoSchedule`
> discourages new pods from scheduling on the node until it is rebooted and
> the taint is removed. All three flags can be repeated and require `--stage`.
> `--stage` is incompatible with `--drain` and `--delete-if-eviction-fails`.

## Behavior

1. **Pre-flight checks**: Ensures all nodes are in the `Running` stage
2. **Version comparison**: Extracts schematic and version from the installer image and only upgrades nodes where either differs from the current state
3. **Per-node confirmation**: Before each upgrade (unless `--confirm=false`, see [global flags](../configuration.md#global-flags))
4. **API selection**: Per node, if the running Talos version is >= 1.13.0, the modern flow (a) is used; otherwise the legacy flow (b) is used

**Modern flow** *(Talos >= 1.13)*:

1. Resolve the Kubernetes node name (if `--drain` is enabled, or if `--stage` with labels/taints)
2. Pre-pull the installer image via `ImageService.Pull`
3. Install the upgrade artifacts via `LifecycleService.Upgrade`
4. **If `--stage` is set**: apply labels/annotations/taints (if any) and stop here — the node is not rebooted
5. Cordon and drain the Kubernetes node if `--drain` is enabled. **If the drain fails** (e.g. a pod cannot be evicted within `--drain-timeout`), the upgrade aborts unless `--delete-if-eviction-fails` is set: in that case, the drain retries with pod deletion (DELETE instead of EVICT, bypassing PodDisruptionBudgets, reusing `--drain-timeout`). If the forced drain also fails, the node is left cordoned with the new artifacts installed but not rebooted, and no further nodes are upgraded. In-flight upgrades on other nodes (when `--max-parallel > 1`) are allowed to complete, but no new ones are started. The node must be uncordoned and rebooted manually to recover.
6. Issue a `Reboot` with the selected reboot mode (default: kexec)
7. Wait 30 seconds for the node to stabilize
8. Uncordon the Kubernetes node

**Legacy flow** *(Talos < 1.13)*:

1. Issue `MachineService.Upgrade`, which installs the upgrade artifacts, cordons and drains the node, and reboots — all in a single server-side sequence. `--drain` and `--drain-timeout` are ignored (Talos drains and uncordons the node itself); `--force` skips etcd health checks. `--stage` is not supported on legacy nodes.
2. Wait 30 seconds for the node to stabilize

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

# Force upgrade on legacy (Talos < 1.13) nodes, skipping etcd health checks
topf upgrade --force

# Upgrade, falling back to pod deletion if graceful drain fails
topf upgrade --delete-if-eviction-fails

# Upgrade with a longer drain timeout (shared by graceful and fallback)
topf upgrade --delete-if-eviction-fails --drain-timeout=10m

# Stage an upgrade without rebooting (reboot manually later to complete it)
topf upgrade --stage

# Stage an upgrade and label the node so a controller can find it
topf upgrade --stage --stage-label topf.io/staged-upgrade=true

# Stage an upgrade and taint the node to discourage new pods from scheduling
topf upgrade --stage --stage-taint topf.postfinance.ch/staged-upgrade=true:PreferNoSchedule

# Stage an upgrade and annotate the node so a controller can find it
topf upgrade --stage --stage-annotation topf.postfinance.ch/staged-at=2026-08-07
```
