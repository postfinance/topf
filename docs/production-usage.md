# Production Usage

For larger environments, the configuration directory should be versioned and TOPF should be run from a CI/CD pipeline rather than locally.

## Versioned Configuration

TOPF does not manage configuration fetching itself, but since the `data` field in `topf.yaml` accepts arbitrary key-value pairs, you can use it to store metadata about where your configuration lives. A CI script can then read those values and pull the configuration before running `topf apply`.

For example, patches and configuration can be maintained in a dedicated Git repository, separate from the per-cluster `topf.yaml`. This allows you to:

- Share a common configuration baseline across multiple clusters
- Pin clusters to a specific config version
- Roll out configuration changes progressively (update `ref` per cluster)

Here's one way to set this up:

```yaml
# topf.yaml
clusterName: prod-cluster-01
clusterEndpoint: https://10.0.0.100:6443
kubernetesVersion: 1.32.8

nodesProvider: custom-binary
secretsProvider: custom-binary

data:
  configSource:
    repo: git.company.tld/kubernetes/talos/config
    ref: main

nodes:
  - host: cp-01
    ip: 10.0.0.1
    role: control-plane
```

A CI step clones the referenced config repo before `topf apply` runs:

```bash
# Pull versioned configuration
git clone --branch "${CONFIG_REF}" "https://${CONFIG_REPO}" .

# Apply
topf apply --confirm=false
```

This way, each cluster's `topf.yaml` is minimal — it defines the cluster identity, nodes, and a pointer to the shared configuration. The actual patches live in the config repository and can be updated independently.

## Example: PostFinance

At PostFinance, TOPF runs in a GitLab CI pipeline with:

- A **nodes provider** binary that fetches the node list from an internal inventory system
- A **secrets provider** binary that retrieves the secrets bundle from the corporate secrets management tool
- A **shared config repository** referenced via `data.configSource`, pulled by the pipeline before apply
- Per-cluster `topf.yaml` files that only contain cluster identity, node definitions, and the config version to use
