# Upgrade Command

The `upgrade` command upgrades Talos OS on each node to the desired version specified in the installer image.

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--confirm` | `true` | Ask for user confirmation before upgrading |
| `--dry-run` | `false` | Only show what upgrades would be performed without actually upgrading |
| `--force` | `false` | Force the upgrade (skip checks on etcd health and members, might lead to data loss) |
| [`--nodes-filter`](../configuration.md#filtering-nodes) | - | Regex pattern to filter which nodes to operate on (global flag) |

## Behavior

1. **Pre-flight checks**: Ensures all nodes are in the `Running` stage
2. **Version comparison**: Extracts schematic and version from the installer image and only upgrades nodes where either differs from the current state
3. **Per-node confirmation**: Before each upgrade (unless `--confirm=false`)
4. **Upgrade**: Issues the upgrade command with `POWERCYCLE` reboot mode
5. **Stabilization**: Waits 30 seconds after upgrade for the node to stabilize

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
