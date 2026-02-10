# Configuration

## topf.yaml

The `topf.yaml` file is the main configuration for your cluster. Here's a complete example with all available fields:

```yaml
# Required fields
clusterName: mycluster
clusterEndpoint: https://192.168.1.100:6443
kubernetesVersion: 1.34.1

# Optional: Directory containing patches (default: ".")
configDir: .

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
```

## Configuration Fields

| Field               | Required | Default | Description                                                                              |
| ------------------- | -------- | ------- | ---------------------------------------------------------------------------------------- |
| `clusterName`       | Yes      | -       | Name of the Kubernetes cluster                                                           |
| `clusterEndpoint`   | Yes      | -       | Kubernetes API endpoint URL                                                              |
| `kubernetesVersion` | Yes      | -       | Kubernetes version to install                                                            |
| `configDir`         | No       | `.`     | Directory containing patch files and node-specific configs                               |
| `secretsProvider`   | No       | -       | Path to binary that manages secrets.yaml                                                 |
| `nodesProvider`     | No       | -       | Path to binary that provides additional nodes                                            |
| `data`              | No       | -       | Arbitrary key-value data for use in [patch templates](configuration-model.md#templating) |
| `nodes`             | Yes      | -       | List of [nodes](#node-fields) in the cluster                                             |

### Node Fields

Each entry in the `nodes` list has the following fields:

| Field  | Required | Description                                                                                                     |
| ------ | -------- | --------------------------------------------------------------------------------------------------------------- |
| `host` | Yes      | Name of the node. Can be a FQDN or short name. Used for display, logging, and certificate validation            |
| `ip`   | No       | IP address used to connect to the node directly instead of resolving `host` via DNS                             |
| `role` | Yes      | Role of the node: `control-plane` or `worker`                                                                   |
| `data` | No       | Arbitrary key-value data for use in [patch templates](configuration-model.md#templating) via `.Node.Data.<key>` |

## Global Flags

TOPF supports the following global flags that can be used with any command:

| Flag             | Environment Variable | Default     | Description                                       |
| ---------------- | -------------------- | ----------- | ------------------------------------------------- |
| `--topfconfig`   | `TOPFCONFIG`         | `topf.yaml` | Path to the topf.yaml configuration file          |
| `--nodes-filter` | `TOPF_NODES_FILTER`  | -           | Regex pattern to filter which nodes to operate on |
| `--log-level`    | `LOG_LEVEL`          | `info`      | Logging level (debug, info, warn, error)          |

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
