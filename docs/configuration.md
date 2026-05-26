# Configuration

## topf.yaml

The `topf.yaml` file is the main configuration for your cluster. Here's a complete example with all available fields:

```yaml
# Required fields
clusterName: mycluster
clusterEndpoint: https://192.168.1.100:6443
kubernetesVersion: 1.34.1

# Optional: Talos version and schematic for installer image generation
talosVersion: 1.12.7
schematicId: 376567988ad370138ad8b2698212367b8edcb69b5fd68c80be1f2ec7d603b4ba

# Optional: Custom image factory and platform (defaults: factory.talos.dev, metal)
# factory: factory.talos.dev
# platform: metal
# secureboot: false

# Optional: Directory containing patches (default: same directory as topf.yaml)
# patchesDir: .

# Optional: Path to secrets.yaml (default: <dir of topf.yaml>/secrets.yaml)
# Relative paths are resolved against the directory containing topf.yaml.
# secretsPath: secrets.yaml

# Optional: Provider binaries for dynamic configuration
secretsProvider: /path/to/secrets-provider
nodesProvider: /path/to/nodes-provider

# Optional: Arbitrary data for use in patch templates
data:
  region: us-west-2
  environment: production

# Node definitions
nodes:
  - host: node1
    ip: 172.20.10.2
    role: control-plane
    data:
      uuid: "550e8400-e29b-41d4-a716-446655440000"
  - host: node2
    ip: 172.20.10.3
    role: worker
  - host: cloud-node1
    ip: 10.0.1.5
    role: worker
    platform: aws  # per-node platform override
```

## Configuration Fields

| Field               | Required | Default | Description                                                                              |
| ------------------- | -------- | ------- | ---------------------------------------------------------------------------------------- |
| `clusterName`       | Yes      | -       | Name of the Kubernetes cluster                                                           |
| `clusterEndpoint`   | Yes      | -       | Kubernetes API endpoint URL                                                              |
| `kubernetesVersion` | Yes      | -       | Kubernetes version to install                                                            |
| `talosVersion`      | No       | bundled Talos version | Talos version used to generate the installer image. Can be overridden per node |
| `schematicId`       | No       | default (no extensions) | Talos image factory schematic ID. Can be a hash string or an `@`-prefixed path to a schematic file (see [Schematic reference resolution](#schematic-reference-resolution)). Can be overridden per node |
| `factory`           | No       | `factory.talos.dev` | Talos image factory address. Can be overridden per node |
| `platform`          | No       | `metal` | Talos platform identifier (e.g. `metal`, `aws`, `gcp`). Can be overridden per node |
| `secureboot`        | No       | `false` | Use the secure boot installer variant (`<platform>-installer-secureboot`). Can be overridden per node |
| `patchesDir`        | No       | directory of topf.yaml | Directory containing patch files and node-specific configurations. Relative paths are resolved against the directory containing topf.yaml |
| `secretsPath`       | No       | `<dir of topf.yaml>/secrets.yaml` | Path to secrets.yaml. Relative paths are resolved against the directory containing topf.yaml |
| `secretsProvider`   | No       | -       | Path to binary that manages secrets.yaml                                                 |
| `nodesProvider`     | No       | -       | Path to binary that provides additional nodes                                            |
| `data`              | No       | -       | Arbitrary key-value data for use in [patch templates](configuration-model.md#templating) |
| `nodes`             | Yes      | -       | List of [nodes](#node-fields) in the cluster                                             |

### Node Fields

Each entry in the `nodes` list has the following fields:

| Field          | Required | Description                                                                                                     |
| -------------- | -------- | --------------------------------------------------------------------------------------------------------------- |
| `host`         | Yes      | Name of the node. Can be a FQDN or short name. Used for display, logging, and certificate validation            |
| `ip`           | No       | IP address used to connect to the node directly instead of resolving `host` via DNS                             |
| `role`         | Yes      | Role of the node: `control-plane` or `worker`                                                                   |
| `talosVersion` | No       | Overrides the cluster-level `talosVersion` for this node                                                         |
| `schematicId`  | No       | Overrides the cluster-level `schematicId` for this node. Supports `@`-prefixed paths                         |
| `factory`      | No       | Overrides the cluster-level `factory` for this node                                                             |
| `platform`     | No       | Overrides the cluster-level `platform` for this node                                                             |
| `secureboot`   | No       | Overrides the cluster-level `secureboot` for this node                                                          |
| `data`         | No       | Arbitrary key-value data for use in [patch templates](configuration-model.md#templating) via `.Node.Data.<key>` |

## Schematic Reference Resolution

Instead of hard-coding a schematic ID hash, you can reference a schematic definition file using the `@` prefix:

```yaml
schematicId: @schematic.yaml
```

Topf will:

1. Read the file (relative to the directory containing `topf.yaml`)
2. If the path ends in `.tpl`, render it through Go templates with the same data available as [patch templates](configuration-model.md#templating)
3. Compute the schematic ID locally from the canonical YAML representation
4. Use the schematic ID for config generation and installer image URLs

This allows you to define your extensions declaratively:

`topf.yaml`:

```yaml
schematicId: @schematic.yaml
```

`schematic.yaml`:

```yaml
customization:
  systemExtensions:
    officialExtensions:
      - siderolabs/qemu-guest-agent
      - siderolabs/nvidia-container-toolkit
```

For templated schematics (`.yaml.tpl`), the full [patch context](configuration-model.md#templating) is available:

`schematic.yaml.tpl`:

```yaml
customization:
  systemExtensions:
    officialExtensions:
      - siderolabs/qemu-guest-agent
      {{- if eq .Node.Role "worker" }}
      - siderolabs/nvidia-container-toolkit
      {{- end }}
```

By default, schematic IDs are computed locally without network calls. This works for any schematic that has already been registered with the image factory, since the ID is a deterministic hash of the canonical YAML.

For **new** schematics that the factory has never seen, you must use `--submit-to-factory` to register them — the factory cannot build images for an ID it doesn't know about. After the initial submission, local computation works for subsequent runs.

!!! tip
    You can check whether a schematic is already known to the factory:

    ```bash
    curl https://factory.talos.dev/schematics/<id>
    ```

    Returns the schematic YAML if found, or `schematic not found` if unknown.

| Flag                  | Environment Variable      | Default | Description                                                          |
| --------------------- | ------------------------- | ------- | -------------------------------------------------------------------- |
| `--submit-to-factory` | `TOPF_SUBMIT_TO_FACTORY`  | `false` | Submit schematics to the image factory API (default: compute IDs locally) |

## Global Flags

TOPF supports the following global flags that can be used with any command:

| Flag                   | Environment Variable     | Default     | Description                                               |
| ---------------------- | ------------------------ | ----------- | --------------------------------------------------------- |
| `--topfconfig`         | `TOPFCONFIG`             | `topf.yaml` | Path to the topf.yaml configuration file                  |
| `--nodes-filter`       | `TOPF_NODES_FILTER`      | -           | Regex pattern to filter which nodes to operate on        |
| `--log-level`          | `LOG_LEVEL`              | `info`      | Logging level (debug, info, warn, error)                  |
| `--confirm`            | `TOPF_CONFIRM`           | `true`      | Confirm any changes before applying them                  |
| `--redact`             | `TOPF_REDACT`            | `true`      | Redact secrets and certificates from output               |
| `--submit-to-factory`  | `TOPF_SUBMIT_TO_FACTORY` | `false`     | Submit schematics to the image factory API (default: compute IDs locally) |

### Filtering Nodes

The `--nodes-filter` flag accepts a **Go regular expression** and applies to all commands. Only nodes whose `host` matches the pattern will be targeted. This is useful for operating on a subset of your cluster.

```bash
# Target a single node
topf apply --nodes-filter "node1"

# Target nodes 1 through 3
topf apply --nodes-filter "node[1-3]"

# Target all control plane nodes by naming convention
topf reset --nodes-filter "cp-.*"

# Target a specific FQDN
topf upgrade --nodes-filter "node1.company.tld"
```

The filter can also be set via the `TOPF_NODES_FILTER` environment variable:

```bash
export TOPF_NODES_FILTER="node[1-3]"
topf apply
```

### Redacting Sensitive Output

When `--redact` is enabled (the default), topf replaces secrets and certificate data with `*** redacted ***` in any command output. The following values are redacted:

- **Talos secrets bundle**: private keys, CA certificates, bootstrap tokens, encryption secrets, and trustd tokens from `secrets.yaml`
- **SOPS-encrypted values**: any value that was encrypted with SOPS in `topf.yaml` or in patch files is decrypted internally and its plaintext is redacted from output
- **vals-resolved values**: any value that was resolved from a [vals](https://github.com/helmfile/vals) reference (e.g. `ref+vault://`, `ref+file://`) has its plaintext redacted from output

Disable it only when you need to inspect the raw diff for debugging:

```bash
topf apply --dry-run --redact=false
```
