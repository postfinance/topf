# Render Command

The `render` command generates machine configuration files for all nodes **without connecting to a cluster**. It uses only local files (`topf.yaml` and patch files), making it useful for validating your patch templates before deploying.

## Flags

| Flag             | Default    | Description                                      |
| ---------------- | ---------- | ------------------------------------------------ |
| `--output`, `-o` | `./output` | Directory to write the generated config files    |
| [`--nodes-filter`](../configuration.md#filtering-nodes) | - | Regex pattern to filter which nodes to render (global flag) |

## Behavior

For each node, `render` assembles the full machine configuration using:

1. The `talosVersion` from `topf.yaml` (falls back to the bundled Talos version if not set)
2. The `schematicId` from `topf.yaml` (falls back to the default no-extensions schematic)
3. All applicable patches from `all/`, `<role>/`, and `node/<host>/`

Each node's config is written to `<output>/<hostname>.yaml`. If config generation fails for a node (e.g. a template error), the error is reported and the other nodes are still processed.

!!! tip
    Set `talosVersion` in `topf.yaml` to pin the Talos version used for rendering. This is especially useful for validating configs against a specific Talos release before upgrading.

## Example Usage

```bash
# Render all node configs to ./output
topf render

# Render to a custom directory
topf render -o /tmp/configs

# Render only control-plane nodes
# (this assumes /cp-.*/ matches exactly the control plane hostnames)
topf render --nodes-filter "cp-.*"
```

## Inspecting Generated Configs

`render` writes one `<hostname>.yaml` file per node so you can inspect the final merged configuration before applying it to the cluster. Errors from template rendering (e.g. missing variables, syntax errors) are reported per-node with the file path, making it easy to pinpoint issues.
