// Package apply contains the logic to apply Talos configurations to cluster nodes
package apply

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"time"

	"github.com/postfinance/topf/internal/topf"
	"github.com/siderolabs/talos/pkg/machinery/resources/runtime"
)

// Options contains the options for the apply execution
type Options struct {
	// Ask for user input before applying
	Confirm bool
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

// applyConfigs applies configuration to all filtered nodes
func applyConfigs(ctx context.Context, logger *slog.Logger, nodes []*topf.Node, opts Options) error {
	for _, node := range nodes {
		logger := logger.With(node.Attrs())

		applied, err := node.Apply(ctx, logger, opts.Confirm, opts.DryRun)
		if err != nil {
			return fmt.Errorf("failed to apply config to node %v: %w", node.Node.Host, err)
		}

		// if nothing was applied or dry-run mode, skip healthchecks
		if !applied || opts.DryRun || opts.SkipPostApplyChecks {
			continue
		}

		if err = node.Stabilize(ctx, logger, time.Second*30); err != nil {
			return fmt.Errorf("node didn't stabilize: %w", err)
		}
	}

	return nil
}
