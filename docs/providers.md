# Dynamic Providers

TOPF supports external provider binaries to dynamically supply nodes, patches and secrets instead of relying solely on static configuration.

## Nodes Provider

By default, nodes are defined statically in `topf.yaml`. When `nodesProvider` is set, TOPF executes the binary to fetch additional nodes which are **merged** with the static ones.

### Binary Contract

The binary is invoked as:

```bash
<binary> nodes <clusterName>
```

It must output a YAML list of nodes to **stdout**:

```yaml
- host: dynamic-node-01
  ip: 192.168.1.10
  role: worker
  data:
    uuid: "550e8400-e29b-41d4-a716-446655440000"
    rack: "A3"
- host: dynamic-node-02
  ip: 192.168.1.11
  role: worker
  data:
    uuid: "6ba7b810-9dad-11d1-80b4-00c04fd430c8"
    rack: "B1"
```

- Exit code `0` on success, non-zero on failure
- **stderr** is passed through to the user for diagnostics
- The `data` field is optional and can carry arbitrary key-value pairs for use in [patch templates](configuration-model.md#templating) via `.Node.Data.<key>`

### Merging

Nodes from the binary are appended to nodes defined in `topf.yaml`. All nodes (static + dynamic) are then subject to `--nodes-filter`.

### Configuration

```yaml
# topf.yaml
nodesProvider: /path/to/nodes-provider
nodes:
  - host: static-node-01
    ip: 192.168.1.1
    role: control-plane
```

## Patches Provider

By default, patches are read from the [patches directory](configuration-model.md). When `patchesProvider` is set, TOPF also executes the binary **once per node** to generate patches for that node. The patches directory then becomes optional: if it does not exist, only the provider patches are applied.

### Binary Contract

The binary is invoked as:

```bash
<binary> patches
```

The patch context of the node is passed as JSON via **stdin**. It holds the same data available to [patch templates](configuration-model.md#templating), and keys match the template field names, so `.Node.Data.uuid` in a template is `.Node.Data.uuid` in the JSON:

```json
{
  "ClusterName": "mycluster",
  "ClusterEndpoint": "https://192.168.1.100:6443",
  "KubernetesVersion": "1.34.1",
  "TalosVersion": "1.12.7",
  "SchematicID": "376567988ad370138ad8b2698212367b8edcb69b5fd68c80be1f2ec7d603b4ba",
  "Data": {
    "region": "us-west-2"
  },
  "Node": {
    "Host": "node1",
    "IP": "172.20.10.2",
    "Role": "control-plane",
    "Data": {
      "uuid": "550e8400-e29b-41d4-a716-446655440000"
    },
    "TalosVersion": "",
    "SchematicID": "",
    "Factory": "",
    "Platform": "",
    "SecureBoot": false,
    "RuntimeData": {
      "TalosVersion": "1.12.7",
      "SchematicID": "376567988ad370138ad8b2698212367b8edcb69b5fd68c80be1f2ec7d603b4ba",
      "Stage": "running"
    }
  },
  "PatchesDir": "/path/to/cluster"
}
```

All keys are always present: unset values are empty strings, `false` or `null`. `SchematicID` holds the resolved ID (after [`@` references](configuration.md#schematic-reference-resolution) are resolved), and `RuntimeData` is only filled when TOPF contacts the live node (see [templating](configuration-model.md#templating)).

The binary must output the patches for the node to **stdout**, in the same format as a [patch file](configuration-model.md#patch-formats): one or more YAML documents containing strategic merge patches.

```yaml
apiVersion: v1alpha1
kind: HostnameConfig
auto: "off"
hostname: node1
```

- Exit code `0` on success, non-zero on failure. A failure aborts config generation for that node
- **stderr** is passed through to the user for diagnostics, without [redaction](configuration.md#redacting-sensitive-output)
- Empty output means there are no additional patches for the node
- The binary is executed concurrently for all processed nodes

A minimal provider written in bash with [jq](https://jqlang.org/):

```bash
#!/usr/bin/env bash
set -euo pipefail

host=$(jq -r '.Node.Host')

cat <<EOF
apiVersion: v1alpha1
kind: HostnameConfig
auto: "off"
hostname: ${host}
EOF
```

Like the output of a [template](configuration-model.md#templating), the output is used as-is and doesn't go through [secret resolution](configuration-model.md#secret-resolution). Secrets copied from `data` stay [redacted](configuration.md#redacting-sensitive-output), but any other secret the provider outputs is not.

### Merge Order

Patches from the provider are applied **before** the patches from the patches directory (`all/`, `<role>/` and `node/<host>/`), so local patches can override fields set by the provider. See [merge order](configuration-model.md#merge-order).

### Configuration

```yaml
# topf.yaml
patchesProvider: /path/to/patches-provider
```

## Secrets Provider

The secrets provider manages the Talos secrets bundle (`secrets.yaml`) which contains cluster certificates, keys, and tokens.

### Default Behavior (no provider)

Without a `secretsProvider`, TOPF reads from and writes to a local `secrets.yaml` file. The location defaults to `secrets.yaml` next to the config file and can be overridden with `secretsPath`. It automatically attempts SOPS decryption on read and SOPS encryption on write. If SOPS is not available, it falls back to plaintext. [vals](https://github.com/helmfile/vals) references in the secrets file are also resolved if present.

### Binary Contract

When `secretsProvider` is set, the binary is invoked with two subcommands:

**Get secrets:**

```bash
<binary> secrets get <clusterName>
```

- Output the secrets bundle as YAML to **stdout**
- Empty output means secrets don't exist yet (not an error)

**Put secrets:**

```bash
<binary> secrets put <clusterName>
```

- The secrets bundle is passed as YAML via **stdin**
- Called when TOPF generates a new secrets bundle

For both commands:

- Exit code `0` on success, non-zero on failure
- **stderr** is passed through to the user

### Configuration

```yaml
# topf.yaml
secretsProvider: /path/to/secrets-provider
```

## Use Cases

The provider interface is intentionally simple (a binary with arguments and stdin/stdout) so it can be implemented in any language or as a wrapper around existing tools.

**Nodes provider ideas:**

- Query a cloud API (AWS, GCP, Azure) to discover instances by tag
- Read Terraform state to extract node IPs and metadata
- Query a CMDB or inventory system

**Patches provider ideas:**

- Generate per-node network configuration (addresses, routes, VLANs) from an IPAM system
- Fetch hardware-specific settings (install disk, kernel arguments) from an inventory system
- Generate patches with a configuration language (CUE, Jsonnet, KCL) instead of Go templates

**Secrets provider ideas:**

- Store and retrieve the secrets bundle from OpenBao
- Use a database or key-value store as backend
- Integrate with a corporate secrets management platform
