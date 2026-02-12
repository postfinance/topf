# Migration from talhelper

This guide walks through migrating an existing talhelper-managed cluster to TOPF.

## Overview

The main differences:

| Aspect       | talhelper                                    | TOPF                                                     |
| ------------ | -------------------------------------------- | -------------------------------------------------------- |
| Config file  | `talconfig.yaml`                             | `topf.yaml`                                              |
| Patches      | Inline in config or separate files           | Separate files in `patches/`, `<role>/`, `nodes/<host>/` |
| Patch format | Strategic merge + JSON patches (RFC 6902)    | Strategic merge only (with `$patch: delete` support)     |
| Secrets      | `talenv.sops.yaml` + envsubst                | SOPS-encrypted fields directly in `topf.yaml`            |
| Templating   | envsubst / talhelper variables               | Go templates (`.Data`, `.Node.Data`)                     |
| Workflow     | Generate configs, then apply with `talosctl` | Direct apply with `topf apply`                           |

## Step 1: Create `topf.yaml`

Translate your `talconfig.yaml` node definitions into `topf.yaml`:

**Before** (`talconfig.yaml`):

```yaml
clusterName: mycluster
endpoint: https://192.168.1.100:6443
kubernetesVersion: v1.32.8
nodes:
  - hostname: node-01
    ipAddress: 192.168.1.1
    installDisk: /dev/nvme0n1
    controlPlane: true
    networkInterfaces:
      - interface: eno1
        dhcp: true
        vip:
          ip: 192.168.1.100
  - hostname: node-02
    ipAddress: 192.168.1.2
    controlPlane: false
```

**After** (`topf.yaml`):

```yaml
clusterName: mycluster
clusterEndpoint: https://192.168.1.100:6443
kubernetesVersion: v1.32.8
nodes:
  - host: node-01
    ip: 192.168.1.1
    role: control-plane
  - host: node-02
    ip: 192.168.1.2
    role: worker
```

Note that node-specific config like `installDisk`, `networkInterfaces`, and `vip` moves into patch files (see below).

## Step 2: Extract patches into files

Inline patches from `talconfig.yaml` become separate files.

**Before** (inline in `talconfig.yaml`):

```yaml
patches:
  - |-
    cluster:
      proxy:
        disabled: true
      discovery:
        enabled: true
controlPlane:
  patches:
    - |-
      ...
```

**After** (separate files):

```text
patches/01-cluster-config.yaml      # global patches
control-plane/01-vip.yaml           # control-plane patches
worker/10-kubelet.yaml              # worker patches
nodes/node-01/01-specific.yaml      # node-specific patches
```

Each file is a standalone YAML document:

```yaml
# patches/01-cluster-config.yaml
cluster:
  proxy:
    disabled: true
  discovery:
    enabled: true
```

## Step 3: Replace JSON patches with strategic merge

talhelper often uses JSON patches (RFC 6902) for removing or adding fields. TOPF uses strategic merge patches instead.

**Before** (JSON patch to remove a label):

```yaml
- op: remove
  path: /machine/nodeLabels/node.kubernetes.io~1exclude-from-external-load-balancers
```

**After** (strategic merge with `$patch: delete`):

```yaml
# control-plane/04-remove-external-lb-exclusion.yaml
machine:
  nodeLabels:
    node.kubernetes.io/exclude-from-external-load-balancers:
      $patch: delete
```

This is cleaner and avoids the need for JSON pointer escaping (`~1` for `/`).

## Step 4: Migrate envs and templating

**Before** (talhelper uses `talenv.sops.yaml` + envsubst):

```yaml
# talenv.sops.yaml
CONTROLPLANE_ENDPOINT: ENC[AES256_GCM,data:...,type:str]

# talconfig.yaml
additionalApiServerCertSans:
  - "${CONTROLPLANE_ENDPOINT}"
```

**After** (TOPF uses SOPS-encrypted `data` fields in `topf.yaml` + Go templates):

```yaml
# topf.yaml (SOPS-encrypted)
data:
  additionalControlPlaneEndpoint: ENC[AES256_GCM,data:...,type:str]
```

```yaml
# control-plane/01-extra-SANs.yaml.tpl
cluster:
  apiServer:
    certSANs:
      - { { .Data.additionalControlPlaneEndpoint } }
```

No separate secrets file needed — sensitive values live directly in `topf.yaml` under `data`, encrypted with SOPS.

## Step 5: Migrate secrets

Move your existing Talos secrets bundle `talsecret.sops.yaml` to `secrets.yaml`. It is compatible with TOPF. Simply keep it in the same directory.

## Step 6: Apply

Instead of generating configs and applying with `talosctl`:

```bash
# Before (talhelper)
talhelper genconfig
talosctl apply-config --nodes 192.168.1.1 --file clusterconfig/mycluster-node-01.yaml

# After (TOPF)
topf apply
```

TOPF handles config generation and apply in a single step.

## Real-world Example

See [this commit](https://github.com/clementnuss/k8s-gitops/commit/a63a51278a4ac0f18995cbcbfd628ef83cb51fc5) for a complete migration from talhelper to TOPF on a homelab cluster.
