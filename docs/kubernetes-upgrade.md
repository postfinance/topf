# Kubernetes Upgrade

There are two ways to upgrade the Kubernetes version of your cluster.

## Option 1: `talosctl upgrade-k8s` (recommended)

The recommended upgrade path is to use `talosctl upgrade-k8s`, which performs a full orchestrated upgrade with compatibility checks, version skew validation, and component-by-component rollout:

```bash
talosctl upgrade-k8s --to 1.33.0
```

After the upgrade completes, update the `kubernetesVersion` field in `topf.yaml` to match:

```yaml
kubernetesVersion: 1.33.0
```

This ensures `topf apply` won't revert the version on the next run.

## Option 2: `topf apply`

Alternatively, you can update `kubernetesVersion` in `topf.yaml` and run `topf apply`. This will update the static pod manifests on each node to the specified version.

```bash
# Edit topf.yaml: kubernetesVersion: 1.33.0
topf apply
```

!!! warning
    This approach does **not** validate version compatibility, API version skew, or component readiness. It blindly applies whatever version you set. This may work well for patch-level updates (e.g. `1.33.0` → `1.33.1`), but should not be relied upon for minor or major version upgrades.
