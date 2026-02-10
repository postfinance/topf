# TOPF - Talos Orchestrator by PostFinance

[![Go Version](https://img.shields.io/github/go-mod/go-version/postfinance/topf)](https://go.dev/)
[![License](https://img.shields.io/github/license/postfinance/topf)](https://github.com/postfinance/topf/blob/main/LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/postfinance/topf)](https://goreportcard.com/report/github.com/postfinance/topf)
[![Latest Release](https://img.shields.io/github/v/release/postfinance/topf)](https://github.com/postfinance/topf/releases/latest)

TOPF is managing [Talos](https://www.talos.dev/) based Kubernetes clusters. It provides functionality for bootstrapping new clusters, resetting existing ones, and applying configuration changes.

[Get Started](getting-started.md){ .md-button .md-button--primary }

## What TOPF does

TOPF is a single binary that handles the full lifecycle of a Talos cluster:

- **Apply configurations** with pre-flight health checks, dry-run diffs, confirmation prompts, and post-apply stabilization — no need to juggle `talosctl` commands per node
- **Upgrade Talos** across all nodes with version comparison, so only nodes that actually need updating are touched
- **Bootstrap and reset** clusters with built-in safety checks
- **Generate kubeconfig and talosconfig** from the secrets bundle

Configuration is built from [layered patches](configuration-model.md) — small, composable YAML files organized by scope (all nodes, role, individual host). This makes cluster config easy to review, version, and share across environments.

TOPF uses the Talos Go libraries directly, so there are no intermediate config files to manage and no dependency on `talosctl` for day-to-day operations.

## Non-goals

TOPF is intentionally limited in scope:

- **Single cluster**: TOPF manages one cluster at a time. Multi-cluster orchestration is out of scope — for managing many clusters, run TOPF in a pipeline per cluster (see [Production Usage](production-usage.md)).
- **Not an operator**: TOPF is a static tool that runs when you invoke it. It performs a single reconciliation pass, not a continuous control loop. This is by design — you decide when changes are applied.
- **No Kubernetes upgrades**: TOPF does not orchestrate Kubernetes version upgrades with proper validation. Use `talosctl upgrade-k8s` for that (see [Kubernetes Upgrade](kubernetes-upgrade.md)).
