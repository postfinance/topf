# Getting Started

## Supported Versions

| TOPF Version  | Talos Version |
| ------------- | ------------- |
| v0.x (latest) | v1.12.x       |

TOPF is built against the Talos v1.12 machinery libraries. It should work with older Talos versions, but only v1.12.x is officially supported and tested.

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
kubernetesVersion: 1.34.1
clusterEndpoint: https://192.168.1.100:6443
clusterName: mycluster

nodes:
  - host: node1
    ip: 172.20.10.2
    role: control-plane
```

Create a new patch to specify the install disk and desired talos version:

`all/01-installation.yaml`:

```yaml
machine:
  install:
    disk: /dev/vda
    image: factory.talos.dev/metal-installer/376567988ad370138ad8b2698212367b8edcb69b5fd68c80be1f2ec7d603b4ba:v1.12.3
```

!!! info "Obtaining the schematic ID"

    The long hash in the installer URL (`376567988a...`) is a **schematic ID** — a content-addressable
    hash of your extensions and overlay configuration. You can either:

    - Use the [Image Factory UI](https://factory.talos.dev) to select extensions and copy the ID, or
    - Reference a schematic file using the `@` prefix (see [Schematic reference resolution](configuration.md#schematic-reference-resolution)): `schematicId: @schematic.yaml`

    For details on how this image is used during upgrades, see [Installer Image](commands/upgrade.md#installer-image).

Then run `topf apply --auto-bootstrap` to provision the cluster.

Once finished use `topf kubeconfig` to create an admin kubeconfig for the cluster and use `topf talosconfig` to create a valid talosconfig.

## Next Steps

Learn how to structure and layer your cluster configuration in the [Configuration Model](configuration-model.md).
