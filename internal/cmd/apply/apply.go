// Copyright 2026 PostFinance AG
// SPDX-License-Identifier: MIT

// Package apply contains the logic to apply Talos configurations to cluster nodes
package apply

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"time"

	"github.com/postfinance/topf/internal/nodepool"
	"github.com/postfinance/topf/internal/topf"
	"github.com/siderolabs/talos/pkg/machinery/api/machine"
	"github.com/siderolabs/talos/pkg/machinery/resources/runtime"
)

// Options contains the options for the apply execution
type Options struct {
	// Only show what changes would be applied without actually applying them
	DryRun bool
	// Automatically bootstrap etcd
	AutoBootstrap bool
	// Skip nodes with pre-flight errors and continue with healthy nodes
	SkipProblematicNodes bool
	// Skip post-apply stabilization and health checks
	SkipPostApplyChecks bool
	// Allow applying to nodes that are not ready (have unmet conditions)
	AllowNotReady bool
	// Apply mode passed to Talos (auto, reboot, no-reboot, staged, try)
	Mode machine.ApplyConfigurationRequest_Mode
	// BatchSize controls how many worker nodes are applied to concurrently.
	// Control-plane nodes are always applied to one at a time.
	BatchSize nodepool.BatchSize
}

// Execute applies the Talos configurations to all nodes in the cluster
func Execute(ctx context.Context, t topf.Topf, opts Options) error {
	logger := t.Logger().With("command", "apply")

	nodes, err := t.Nodes(ctx)
	if err != nil {
		return err
	} else if len(nodes) == 0 {
		logger.Warn("no nodes to process. exiting")
		return nil
	}

	// Pre-flight checks
	filteredNodes, err := runPreflightChecks(logger, nodes, &opts)
	if err != nil {
		return err
	}

	// Apply configs
	if err := applyConfigs(ctx, logger, filteredNodes, opts); err != nil {
		return err
	}

	// Bootstrap if requested (skip in dry-run mode)
	if opts.AutoBootstrap && !opts.DryRun {
		return bootstrap(ctx, logger, filteredNodes)
	}

	return nil
}

// runPreflightChecks filters nodes based on pre-flight checks and returns healthy nodes
func runPreflightChecks(logger *slog.Logger, nodes []*topf.Node, opts *Options) ([]*topf.Node, error) {
	maintenanceNodesCnt := 0

	filteredNodes := slices.DeleteFunc(nodes, func(node *topf.Node) bool {
		logger := logger.With(node.Attrs())

		if node.Error != nil {
			logger.Error("node pre-checks", "error", node.Error)
			return true
		}

		// when AllowNotReady is true, we skip the readiness check
		if !opts.AllowNotReady && !node.MachineStatus.Status.Ready {
			logger.Error("node not ready", "unmet conditions", node.MachineStatus.Status.UnmetConditions)
			return true
		}

		st := node.MachineStatus.Stage
		if !slices.Contains([]runtime.MachineStage{runtime.MachineStageRunning, runtime.MachineStageMaintenance, runtime.MachineStageBooting}, st) {
			logger.Error("node in unprocessable stage", "stage", st.String())
			return true
		}

		if st == runtime.MachineStageMaintenance {
			maintenanceNodesCnt++
		}

		return false
	})

	if len(filteredNodes) == 0 {
		return nil, errors.New("no healthy nodes available to process")
	}

	if len(filteredNodes) != len(nodes) {
		if opts.SkipProblematicNodes {
			logger.Warn("pre-flight checks failed for some nodes. continuing with healthy nodes only", "healthy_nodes", len(filteredNodes), "total_nodes", len(nodes))
		} else {
			return nil, errors.New("aborting due to errors with some nodes")
		}
	}

	if maintenanceNodesCnt == len(filteredNodes) {
		logger.Info("all nodes are in maintenance stage. ignoring the post-apply checks")

		opts.SkipPostApplyChecks = true
	}

	return filteredNodes, nil
}

// applyConfigs applies configuration to all filtered nodes. In dry-run mode all
// nodes are processed sequentially. Otherwise control-plane nodes are applied to
// one at a time (to preserve etcd quorum and keep the bootstrap node first), and
// worker nodes are applied to using a rolling pool of at most BatchSize nodes.
func applyConfigs(ctx context.Context, logger *slog.Logger, nodes []*topf.Node, opts Options) error {
	if opts.DryRun {
		return applyDryRun(ctx, logger, nodes, opts)
	}

	controlPlane, workers := nodepool.PartitionByRole(nodes)

	for _, node := range controlPlane {
		if err := applyNode(ctx, node, opts, logger.With(node.Attrs())); err != nil {
			return err
		}
	}

	if len(workers) > 0 {
		concurrency := opts.BatchSize.Resolve(len(nodes))
		logger.Info("applying to worker nodes", "count", len(workers), "concurrency", concurrency)

		return nodepool.RunConcurrent(ctx, workers, concurrency,
			func(ctx context.Context, node *topf.Node, logger *slog.Logger) error {
				return applyNode(ctx, node, opts, logger)
			}, logger)
	}

	return nil
}

// applyDryRun applies the configuration to all nodes sequentially in dry-run
// mode, aggregating whether any changes were detected.
func applyDryRun(ctx context.Context, logger *slog.Logger, nodes []*topf.Node, opts Options) error {
	var changesDetected bool

	for _, node := range nodes {
		logger := logger.With(node.Attrs())

		err := applyNode(ctx, node, opts, logger)
		if errors.Is(err, topf.ErrDryRunChangesDetected) {
			changesDetected = true
			continue
		}

		if err != nil {
			return err
		}
	}

	if changesDetected {
		return topf.ErrDryRunChangesDetected
	}

	return nil
}

// applyNode applies the configuration to a single node and, unless skipped,
// waits for it to stabilize. The provided logger is expected to already carry
// the node's attributes. In dry-run mode it returns ErrDryRunChangesDetected
// when changes are detected.
func applyNode(ctx context.Context, node *topf.Node, opts Options, logger *slog.Logger) error {
	applied, err := node.Apply(ctx, logger, opts.DryRun, opts.Mode)
	if errors.Is(err, topf.ErrDryRunChangesDetected) {
		return err
	}

	if err != nil {
		return fmt.Errorf("failed to apply config to node %v: %w", node.Node.Host, err)
	}

	// if nothing was applied or dry-run mode, skip healthchecks
	if !applied || opts.DryRun || opts.SkipPostApplyChecks {
		return nil
	}

	if err = node.Stabilize(ctx, logger, time.Second*30); err != nil {
		return fmt.Errorf("node didn't stabilize: %w", err)
	}

	return nil
}
