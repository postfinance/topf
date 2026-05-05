# Render Command

The `render` command generates machine configuration files for all nodes using local files (`topf.yaml` and patch files). With `--online`, it queries live nodes for their actual running Talos version before generating configs.

## Flags

| Flag             | Default    | Description                                      |
| ---------------- | ---------- | ------------------------------------------------ |
| `--output`, `-o` | `./output` | Directory to write the generated config files    |
| `--online`       | `false`    | Query live nodes for their running Talos version |
| [`--nodes-filter`](../configuration.md#filtering-nodes) | - | Regex pattern to filter which nodes to render (global flag) |

## Behavior

For each node, `render` assembles the full machine configuration using:

1. The Talos version for config generation (fallback chain: running version from `--online` → `talosVersion` from `topf.yaml` → bundled Talos version)
2. The `schematicId` from `topf.yaml` (falls back to the default no-extensions schematic)
3. All applicable patches from `all/`, `<role>/`, and `node/<host>/`

Each node's config is written to `<output>/<hostname>.yaml`. If config generation fails for a node (e.g. a template error), the error is reported and the other nodes are still processed.

### Online mode

With `--online`, topf connects to each node and retrieves the actually running Talos version. This ensures the generated config uses the correct version contract for the node's current state — useful when nodes may be at different versions.

Without `--online`, the Talos version is resolved from `topf.yaml` (or the bundled version), which is sufficient when all nodes are at the same known version.

## Example Usage

```bash
# Render all node configs to ./output
topf render

# Render using the actual running Talos versions
topf render --online

# Render to a custom directory
topf render -o /tmp/configs

# Render only control-plane nodes
topf render --nodes-filter "cp-.*"
```

## Inspecting Generated Configs

`render` writes one `<hostname>.yaml` file per node so you can inspect the final merged configuration before applying it to the cluster. Errors from template rendering (e.g. missing variables, syntax errors) are reported per-node, making it easy to pinpoint issues.
