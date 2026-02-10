# TOPF - Talos Orchestrator by PostFinance

[![Go Version](https://img.shields.io/github/go-mod/go-version/postfinance/topf)](https://go.dev/)
[![License](https://img.shields.io/github/license/postfinance/topf)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/postfinance/topf)](https://goreportcard.com/report/github.com/postfinance/topf)
[![Latest Release](https://img.shields.io/github/v/release/postfinance/topf)](https://github.com/postfinance/topf/releases/latest)

TOPF is managing [Talos](https://www.talos.dev/) based Kubernetes clusters. It
provides functionality for bootstrapping new clusters, resetting existing ones,
and applying configuration changes.

**[Full Documentation](https://postfinance.github.io/topf)**

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
    image: factory.talos.dev/metal-installer/376567988ad370138ad8b2698212367b8edcb69b5fd68c80be1f2ec7d603b4ba:v1.12.0
```

Then run `topf apply --auto-bootstrap` to provision the cluster.

Once finished use `topf kubeconfig` to create an admin kubeconfig for the
cluster and use `topf talosconfig` to create a valid talosconfig.

For detailed documentation on configuration, commands, patches, and more, visit
the **[full documentation site](https://postfinance.github.io/topf)**.

## Alternatives

- **[talosctl](https://www.talos.dev/)** — the official Talos CLI; fully
featured but lower-level, requires managing configs and node operations
manually
- **[talhelper](https://github.com/budimanjojo/talhelper)** — popular community
tool for generating Talos machine configs from a declarative YAML definition
- **[Omni](https://www.siderolabs.com/omni/)** — SideroLabs' management
platform for Talos clusters with a UI, multi-environment support, and GitOps
workflows; available as SaaS or self-hosted (requires a commercial license for
production use)
