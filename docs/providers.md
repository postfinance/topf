# Dynamic Providers

TOPF supports external provider binaries to dynamically supply nodes and secrets instead of relying solely on static configuration.

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

## Secrets Provider

The secrets provider manages the Talos secrets bundle (`secrets.yaml`) which contains cluster certificates, keys, and tokens.

### Default Behavior (no provider)

Without a `secretsProvider`, TOPF reads from and writes to a local `secrets.yaml` file. It automatically attempts SOPS decryption on read and SOPS encryption on write. If SOPS is not available, it falls back to plaintext.

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

**Secrets provider ideas:**

- Store and retrieve the secrets bundle from OpenBao
- Use a database or key-value store as backend
- Integrate with a corporate secrets management platform
