# Clusterinfo Command

The `clusterinfo` command outputs non-sensitive cluster information in YAML format.

## Output

The output includes:

- **clusterName**: Name of the cluster
- **clusterEndpoint**: Kubernetes API endpoint
- **kubernetesVersion**: Configured Kubernetes version
- **clusterCA**: Base64-encoded Kubernetes CA certificate
- **etcdCA**: Base64-encoded etcd CA certificate
- **talosCA**: Base64-encoded Talos OS CA certificate
- **nodes**: Node configuration

No sensitive data (private keys, tokens) is included.

## Example Usage

```bash
topf clusterinfo
```
