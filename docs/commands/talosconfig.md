# Talosconfig Command

The `talosconfig` command generates a Talos client configuration file (talosconfig) from the cluster's secrets bundle.

The endpoints will be set to the control-plane nodes. Duplicates are removed.

The nodes list contains all nodes.

## Example Usage

```bash
# Output talosconfig to stdout
topf talosconfig

# Save and use with talosctl
topf talosconfig > talosconfig
export TALOSCONFIG=$(pwd)/talosconfig
talosctl version
```
