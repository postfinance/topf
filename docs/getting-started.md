# Getting Started

## Supported Versions

| TOPF Version  | Talos Version |
| ------------- | ------------- |
| v0.x (latest) | v1.12.x       |

TOPF is built against the Talos v1.12 machinery libraries. It should work with older Talos versions, but only v1.12.x is officially supported and tested.

## Installation

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

## Quickstart

Boot at least one Talos machine to maintenance mode.

Create a new folder for you cluster with a `topf.yaml` file:

```yaml
kubernetesVersion: 1.34.1
clusterEndpoint: https://192.168.1.100:6443
clusterName: mycluster

nodes:
  - host: node1
    ip: 172.20.10.2
    role: control-plane
```

Create a new patch to specify the install disk and desired talos version:

`patches/01-installation.yaml`:

```yaml
machine:
  install:
    disk: /dev/vda
    image: factory.talos.dev/metal-installer/376567988ad370138ad8b2698212367b8edcb69b5fd68c80be1f2ec7d603b4ba:v1.11.5
```

Then run `topf apply --auto-bootstrap` to provision the cluster.

Once finished use `topf kubeconfig` to create an admin kubeconfig for the cluster and use `topf talosconfig` to create a valid talosconfig.

## Next Steps

Learn how to structure and layer your cluster configuration in the [Configuration Model](configuration-model.md).
