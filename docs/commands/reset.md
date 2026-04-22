# Reset Command

The `reset` command resets Talos node(s) to their initial state, wiping system partitions and rebooting.

Nodes already in maintenance mode are automatically skipped.

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--confirm` | `true` | Ask for user confirmation before resetting |
| `--full` | `true` | Wipe the entire disk. If `false`, only STATE and EPHEMERAL partitions are wiped |
| `--graceful` | `false` | Attempt to cordon/drain the node and leave etcd before resetting |
| `--shutdown` | `false` | Shut down the machine after reset instead of rebooting |
| `--wait-for-maintenance` | `false` | Wait for all reset nodes to reach maintenance mode before returning |
| [`--nodes-filter`](../configuration.md#filtering-nodes) | - | Regex pattern to filter which nodes to operate on (global flag) |

## Example Usage

```bash
# Reset with full disk wipe (default)
topf reset

# Reset only STATE and EPHEMERAL partitions
topf reset --full=false

# Graceful reset (drain workloads first)
topf reset --graceful

# Reset and shut down instead of rebooting
topf reset --shutdown

# Reset specific nodes
topf reset --nodes-filter "node[1-2]"

# Reset and wait for nodes to enter maintenance mode (useful for chaining with apply)
topf reset --wait-for-maintenance
```
