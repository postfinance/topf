# Nodes Command

The `nodes` command lists all nodes and their current state.

## Flags

| Flag                                                    | Default    | Description                                                     |
| ------------------------------------------------------- | ---------- | --------------------------------------------------------------- |
| `--output`, `-o`                                        | `table`    | Output format: `table` or `yaml`                                |
| `--machineconfig-output`, `-m`                          | `./output` | Write machine configs to this directory                         |
| [`--nodes-filter`](../configuration.md#filtering-nodes) | -          | Regex pattern to filter which nodes to operate on (global flag) |

## Table Output

The default table output displays the following columns:

| Column           | Description                     |
| ---------------- | ------------------------------- |
| Host             | Node hostname                   |
| IP               | Node IP address                 |
| Role             | `control-plane` or `worker`     |
| Stage            | Current machine stage           |
| Ready            | `✓` or `✗`                      |
| Unmet Conditions | Conditions preventing readiness |
| Schematic        | Installer schematic (truncated) |
| Talos            | Installed Talos version         |
| Error            | Any error encountered           |

## Machine Config Export

When using `--machineconfig-output`, TOPF writes each node's generated machine configuration to the specified directory as `<hostname>.yaml`.

## Example Usage

```bash
# List nodes in table format
topf nodes

# List nodes in YAML format
topf nodes -o yaml

# Export machine configs
topf nodes -m ./configs
```
