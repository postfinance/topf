# Nodes Command

The `nodes` command lists all nodes and their current state.

## Flags

| Flag                                                    | Default | Description                                                     |
| ------------------------------------------------------- | ------- | --------------------------------------------------------------- |
| `--output`, `-o`                                        | `table` | Output format: `table` or `yaml`                                |
| [`--nodes-filter`](../configuration.md#filtering-nodes) | -       | Regex pattern to filter which nodes to operate on (global flag) |

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

## Example Usage

```bash
# List nodes in table format
topf nodes

# List nodes in YAML format
topf nodes -o yaml
```

!!! tip
    To generate machine configs without a running cluster, use the [`render`](render.md) command. Use `render --online` to generate configs using the actual running Talos version.
