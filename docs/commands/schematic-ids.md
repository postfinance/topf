# Schematic IDs Command

The `schematic-ids` command resolves and prints the schematic IDs for all configured nodes. This is useful for external tooling that needs the resolved hashes — e.g. downloading installer ISOs from the image factory or embedding IDs in CI artifacts.

## Output

One schematic ID per line, deduplicated and sorted. Plain stdout, no logging noise, so it composes cleanly in shell pipelines:

```bash
$ topf schematic-ids
376567988ad370138ad8b2698212367b8edcb69b5fd68c80be1f2ec7d603b4ba
ce4c980550dd2ab1b17bbf2b08801c7eb59418eafe8f279833297925d67c7515
```

## Flags

| Flag                                                    | Default | Description                                                     |
| ------------------------------------------------------- | ------- | --------------------------------------------------------------- |
| [`--nodes-filter`](../configuration.md#filtering-nodes) | -       | Regex pattern to filter which nodes to resolve schematics for (global flag) |

The [`--submit-to-factory`](../configuration.md#global-flags) global flag is also honored — when set, new schematics are submitted to the image factory API.

## Example Usage

```bash
# Print all resolved schematic IDs
topf schematic-ids

# Filter to a specific node
topf schematic-ids --nodes-filter "node1"

# Use in a script to download the installer ISO
ID=$(topf schematic-ids --nodes-filter "^node1$")
curl -Lo metal-amd64.iso "https://factory.talos.dev/image/${ID}/v1.12.7/metal-amd64.iso"
```
