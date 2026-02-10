# Talosconfig Command

The `talosconfig` command generates a Talos client configuration from the cluster's secrets bundle.

## Behavior

The generated talosconfig is configured differently based on cluster size:

**Single-node cluster:**

- Endpoints: the single node
- Nodes: the single node

**Multi-node cluster:**

- Endpoints: all control-plane nodes
- Nodes: all nodes (control-plane and workers)

## Example Usage

```bash
# Output talosconfig to stdout
topf talosconfig

# Save and use with talosctl
topf talosconfig > talosconfig
export TALOSCONFIG=$(pwd)/talosconfig
talosctl version
```
