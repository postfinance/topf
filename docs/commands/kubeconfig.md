# Kubeconfig Command

The `kubeconfig` command generates a temporary kubeconfig for cluster access, with full admin access by default.

## Behavior

The generated kubeconfig:

- Is valid for **12 hours** (configurable via `--validity`)
- Uses a client certificate for user `topf` (configurable via `--user`) in group `system:masters` (full admin access, configurable via `--groups`)
- Is signed by the cluster's Kubernetes CA
- Context name: `<user>@<cluster-name>`

## Example Usage

```bash
# Output kubeconfig to stdout
topf kubeconfig

# Save and use immediately
topf kubeconfig > kubeconfig
export KUBECONFIG=$(pwd)/kubeconfig
kubectl get nodes
```

## Limited-scope kubeconfigs

topf does not create any RBAC objects. To mint a kubeconfig with limited
access, bind a group (or user) to a role in the cluster, e.g. to the built-in
`view` ClusterRole:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: viewers
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: view
subjects:
  - apiGroup: rbac.authorization.k8s.io
    kind: Group
    name: viewers
```

Then generate a kubeconfig for that group:

```bash
topf kubeconfig --validity 1h --user alice --groups viewers > kubeconfig
```
