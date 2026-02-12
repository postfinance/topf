# Apply Command

The `apply` command is the primary way to apply configuration changes to a running Talos cluster. It handles the full lifecycle of updating nodes, from pre-flight checks to post-apply validation.

## Flow

1. **Gather Nodes**: Read `topf.yaml` and generate configurations for all nodes

2. **Pre-flight Checks**: Validate each node's health
   - Nodes with errors → unhealthy
   - Nodes not ready (unmet conditions) → unhealthy (unless `--allow-not-ready` is set)
   - Nodes not in Running/Maintenance/Booting stage → unhealthy
   - **If any unhealthy nodes found**:
     - Without `--skip-problematic-nodes`: **ABORT**
     - With `--skip-problematic-nodes`: Continue with healthy nodes only (warn and filter)

3. **Determine Post-Apply Behavior**: If all remaining nodes are in maintenance mode, automatically enable `--skip-post-apply-checks`

4. **Apply Configurations** (for each healthy node):
   - Dry-run apply to check for changes
   - If changes detected:
     - Show diff (if `--confirm` enabled)
     - Ask for confirmation (if `--confirm` enabled)
     - Apply configuration
   - If config applied AND not `--skip-post-apply-checks`: Stabilize (wait 30s for node to be ready)

5. **Bootstrap** (if `--auto-bootstrap` enabled):
   - Select first control plane node
   - Call ETCD bootstrap API
   - Retry for up to 10 minutes

## Flags

| Flag                       | Default | Description                                                        |
| -------------------------- | ------- | ------------------------------------------------------------------ |
| `--confirm`                | `true`  | Ask for confirmation before applying changes to each node          |
| `--dry-run`                | `false` | Only show changes without actually applying them                   |
| `--auto-bootstrap`         | `false` | Automatically bootstrap ETCD after applying configurations         |
| `--skip-problematic-nodes` | `false` | Continue with healthy nodes if some fail pre-flight checks         |
| `--skip-post-apply-checks` | `false` | Skip the 30-second stabilization check after applying configs      |
| `--allow-not-ready`        | `false` | Allow applying to nodes that are not ready (have unmet conditions) |
| [`--nodes-filter`](../configuration.md#filtering-nodes) | -       | Regex pattern to filter which nodes to operate on (global flag)    |

## Example Usage

```bash
# Apply with confirmation (default)
topf apply

# Apply without confirmation
topf apply --confirm=false

# Preview what would be changed
topf apply --dry-run

# Apply and bootstrap a new cluster
topf apply --auto-bootstrap

# Apply only to healthy nodes, skip problematic ones
topf apply --skip-problematic-nodes

# Apply without waiting for nodes to stabilize
topf apply --skip-post-apply-checks

# Apply to nodes even if they have unmet conditions
topf apply --allow-not-ready
```

## Pre-flight Checks

The apply command validates each node before attempting to apply configuration:

- **Node errors**: Skip nodes that failed to initialize or communicate
- **Ready status**: Skip nodes with unmet conditions (e.g., missing network, disk issues) unless `--allow-not-ready` is set
- **Machine stage**: Only process nodes in Running, Maintenance, or Booting stages

## Post-apply Stabilization

After applying configuration to a node, the command waits up to 30 seconds for the node to:

- Report as ready
- Have no unmet conditions
- Reach a stable state

This check is automatically skipped if:

- All nodes are in maintenance mode (fresh install)
- The `--skip-post-apply-checks` flag is set
- No configuration changes were applied

## Bootstrap

When `--auto-bootstrap` is enabled, after all configurations are applied, the command will:

1. Select the first control plane node
2. Call the ETCD bootstrap API
3. Retry for up to 10 minutes if the call fails

This is typically used when bringing up a new cluster for the first time, and has no influence if the cluster is already bootstrapped.
