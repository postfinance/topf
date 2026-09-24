# Reset Command

The `reset` command resets system partitions and selected disks of a Talos node. Followed by a reboot or shutdown.

Nodes already in maintenance mode are automatically skipped.

## Flags

All flags can also be set via environment variables using the `TOPF_` prefix and uppercasing the flag name (e.g. `--wait-for-maintenance` → `TOPF_WAIT_FOR_MAINTENANCE`).

| Flag | Default | Description |
|------|---------|-------------|
| `--wipe-state-and-ephemeral` | `false` | if true, only STATE and EPHEMERAL partitions are wiped |
| `--graceful` | `true` | Attempt to cordon/drain the node and leave etcd before resetting |
| `--shutdown` | `false` | Shut down the machine after reset instead of rebooting |
| `--wait-for-maintenance` | `false` | Wait for all reset nodes to reach maintenance mode before returning |
| `--wipe-mode` | `all` | Disk reset mode |
| `--system-labels-to-wipe` | - | If set, just wipe selected system disk partitions by label but keep other partitions intact |
| `--user-disks-to-wipe` | - | If set, wipes defined devices in the list |
| [`--nodes-filter`](../configuration.md#filtering-nodes) | - | Regex pattern to filter which nodes to operate on (global flag) |

## Example Usage

```bash
# Reset all (incl. boot) system partitions (default)
topf reset

# Graceful reset (drain workloads and leave etcd first)
topf reset --graceful

# Reset and shut down instead of rebooting
topf reset --shutdown

# Reset only STATE and EPHEMERAL partitions
topf reset --wipe-state-and-ephemeral

# Reset only selected partitions
topf reset --system-labels-to-wipe STATE --system-labels-to-wipe EPHEMERAL

# Wipe user disks
topf reset --user-disks-to-wipe /dev/sdb --user-disks-to-wipe /dev/sdc

# Reset specific nodes
topf reset --nodes-filter "node[1-2]"

# Reset and wait for nodes to enter maintenance mode (useful for chaining with apply)
topf reset --wait-for-maintenance
```
