# Getting Started

## Supported Versions

| TOPF Version  | Talos Version |
| ------------- | ------------- |
| v0.x (latest) | v1.13.x       |

TOPF is built against the Talos v1.13 machinery libraries.

## Installation

### Homebrew

```bash
brew install postfinance/tap/topf
```

### Go Install

```bash
go install github.com/postfinance/topf/cmd/topf@latest
```

### Binary Download

Download the latest binary from [GitHub Releases](https://github.com/postfinance/topf/releases/latest).

### Container Image

```bash
docker pull ghcr.io/postfinance/topf
```

## Optional Dependencies

| Tool   | Purpose |
| ------ | ------- |
| [SOPS](https://github.com/getsops/sops) + [age](https://github.com/FiloSottile/age) | Decrypt SOPS-encrypted config and patch files |
| [vals](https://github.com/helmfile/vals) | Resolve vals references (`ref+<provider>://`) in config and patches |

These are only needed if your configuration uses SOPS encryption or vals references. Without them, plaintext configs work normally.

## Quickstart

Boot at least one Talos machine to maintenance mode.

Create a new folder for your cluster with a `topf.yaml` file:

```yaml
kubernetesVersion: 1.35.5
talosVersion: 1.13.4
clusterEndpoint: https://192.168.1.100:6443
clusterName: mycluster

nodes:
  - host: node1
    ip: 192.168.1.100
    role: control-plane
```

Create a new patch to specify the install disk:

`all/00-installation.yaml`:

```yaml
machine:
  install:
    disk: /dev/vda
```

### Set the node hostname

By default, Talos generates hostnames automatically (e.g. `talos-XXX-XXX`). The `host` value in `topf.yaml` is used by TOPF for display, logging, and node selection — it is **not** automatically applied as the Talos hostname.

To use the configured `host` value as the hostname, add a `HostnameConfig` patch. Create `all/01-hostname.yaml.tpl`:

```yaml
apiVersion: v1alpha1
kind: HostnameConfig
auto: "off"
hostname: {{ .Node.Host }}
```

This patch is applied to every node and sets the hostname from `nodes[].host`. For details on how patches are layered, see [Configuration Model](configuration-model.md).

Then run `topf apply --auto-bootstrap` to provision the cluster.

Once finished use `topf kubeconfig` to create an admin kubeconfig for the cluster and use `topf talosconfig` to create a valid talosconfig.

## Next Steps

Learn how to structure and layer your cluster configuration in the [Configuration Model](configuration-model.md).
