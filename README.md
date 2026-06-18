# TOPF - Talos Orchestrator by PostFinance

<img src="docs/assets/topf.png" alt="TOPF logo" width="200">

[![Go Version](https://img.shields.io/github/go-mod/go-version/postfinance/topf)](https://go.dev/)
[![License](https://img.shields.io/github/license/postfinance/topf)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/postfinance/topf)](https://goreportcard.com/report/github.com/postfinance/topf)
[![Latest Release](https://img.shields.io/github/v/release/postfinance/topf)](https://github.com/postfinance/topf/releases/latest)

TOPF is managing [Talos](https://www.talos.dev/) based Kubernetes
clusters. It provides functionality for bootstrapping new clusters,
resetting existing ones, and applying configuration changes.

**[Full Documentation](https://postfinance.github.io/topf)**

[![demo](https://asciinema.org/a/yg1XKJYpwIJUdJZT.svg)](https://asciinema.org/a/yg1XKJYpwIJUdJZT)

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

## Quickstart

Boot at least one Talos machine to maintenance mode.

Create a new folder for your cluster with a `topf.yaml` file:

```yaml
kubernetesVersion: 1.35.3
talosVersion: 1.13.4
clusterEndpoint: https://192.168.1.100:6443
clusterName: mycluster

nodes:
- host: node1
  ip: 172.20.10.2
  role: control-plane
```

TOPF generates the installer image from `talosVersion` (and the optional `schematicId`) in `topf.yaml`. Create a patch to specify the install disk:

`all/00-installation.yaml`:

```yaml
machine:
  install:
    disk: /dev/sda
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

This patch is applied to every node and sets the hostname from `nodes[].host`.

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
