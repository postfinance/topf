# Kubeconfig Command

The `kubeconfig` command generates a temporary admin kubeconfig for cluster access.

## Behavior

The generated kubeconfig:

- Is valid for **12 hours**
- Uses a client certificate with `system:masters` group (full admin access)
- Is signed by the cluster's Kubernetes CA
- Context name: `topf@<cluster-name>`

## Example Usage

```bash
# Output kubeconfig to stdout
topf kubeconfig

# Save and use immediately
topf kubeconfig > kubeconfig
export KUBECONFIG=$(pwd)/kubeconfig
kubectl get nodes
```
