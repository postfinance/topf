// Copyright 2026 PostFinance AG
// SPDX-License-Identifier: MIT

// Package upgrade contains the logic to upgrade Talos OS on cluster nodes
package upgrade

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"slices"
	"time"

	"github.com/postfinance/topf/internal/interactive"
	"github.com/postfinance/topf/internal/nodepool"
	"github.com/postfinance/topf/internal/topf"
	"github.com/siderolabs/talos/pkg/machinery/api/machine"
	"github.com/siderolabs/talos/pkg/machinery/resources/runtime"
)

// Options contains the options for the upgrade execution
type Options struct {
	// Only show what upgrades would be performed without actually upgrading
	DryRun bool

	// Talos upgrade options
	Force      bool
	RebootMode machine.UpgradeRequest_RebootMode

	// MaxParallel controls how many worker nodes are upgraded concurrently.
	// Control-plane nodes are always upgraded one at a time.
	MaxParallel nodepool.MaxParallel
}

// Execute performs the Talos OS upgrades for all nodes in the cluster
func Execute(ctx context.Context, t topf.Topf, opts Options) error {
	logger := t.Logger().With("command", "upgrade")

	// Gather node information
	nodes, err := t.Nodes(ctx)
	if err != nil {
		return err
	}

	if err := preChecks(logger, nodes); err != nil {
		return err
	}

	// Plan phase: determine which nodes require an upgrade. Interactive
	// confirmations happen here, sequentially, before any concurrent work.
	worklist, upgradeRequired, err := plan(t, logger, nodes, opts)
	if err != nil {
		return err
	}

	if opts.DryRun {
		if upgradeRequired {
			return topf.ErrDryRunChangesDetected
		}

		return nil
	}

	controlPlane, workers := nodepool.PartitionByRole(worklist)

	// Control-plane nodes are upgraded strictly one at a time to preserve etcd
	// quorum; this also satisfies "control-plane upgrades cannot be scheduled
	// concurrently".
	for _, node := range controlPlane {
		if err := upgradeNode(ctx, node, opts, logger.With(node.Attrs())); err != nil {
			return err
		}
	}

	// Worker nodes are upgraded using a rolling pool: up to n upgrades are kept
	// in flight, and as soon as one finishes the next node is started.
	if len(workers) > 0 {
		concurrency := opts.MaxParallel.Resolve(len(nodes))
		logger.Info("upgrading worker nodes", "count", len(workers), "concurrency", concurrency)

		return nodepool.RunConcurrent(ctx, workers, concurrency,
			func(ctx context.Context, node *topf.Node, logger *slog.Logger) error {
				return upgradeNode(ctx, node, opts, logger)
			}, logger)
	}

	return nil
}

// preChecks verifies that every node is reachable and running before any
// upgrade is attempted, reporting all problems at once.
func preChecks(logger *slog.Logger, nodes []*topf.Node) error {
	abort := false

	for _, node := range nodes {
		logger := logger.With(node.Attrs())

		if node.Error != nil {
			logger.Error("node pre-checks", "error", node.Error)

			abort = true

			continue
		}

		if !slices.Contains([]runtime.MachineStage{runtime.MachineStageRunning}, node.MachineStatus.Stage) {
			logger.Error("node must be 'running' for upgrade", "stage", node.MachineStatus.Stage.String())

			abort = true

			continue
		}
	}

	if abort {
		return errors.New("aborting due to errors with some nodes")
	}

	return nil
}

// plan determines which nodes require an upgrade, performing interactive
// confirmations sequentially before any concurrent work.
func plan(t topf.Topf, logger *slog.Logger, nodes []*topf.Node, opts Options) (worklist []*topf.Node, upgradeRequired bool, err error) {
	for _, node := range nodes {
		logger := logger.With(node.Attrs())

		installerImage := node.ConfigProvider().Machine().Install().Image()

		schematic, talosVersion, err := extractSchematicAndVersion(installerImage)
		if err != nil {
			return nil, false, fmt.Errorf("couldn't extract schematic and version from installer image '%s': %w", installerImage, err)
		}

		nodeNeedsUpgrade := node.RunningVersion() != talosVersion || node.RunningSchematic() != schematic
		if !nodeNeedsUpgrade {
			logger.Info("no upgrade required")
			continue
		}

		logger.Info("upgrade required",
			"schematic_actual", node.RunningSchematic(),
			"schematic_desired", schematic,
			"version_actual", node.RunningVersion(),
			"version_desired", talosVersion,
			"installer", installerImage)

		upgradeRequired = true

		// in dry-run mode, skip the actual upgrade
		if opts.DryRun {
			continue
		}

		// ask for user confirmation
		if t.Confirm() {
			if interactive.ConfirmPrompt(fmt.Sprintf("Do you want to upgrade node %s with installer %s? This will reboot the node.", node.Node.Host, installerImage)) == 'n' {
				logger.Info("skipping upgrade")
				continue
			}
		}

		worklist = append(worklist, node)
	}

	return worklist, upgradeRequired, nil
}

// upgradeNode issues the upgrade RPC to a single node and waits for it to
// stabilize. The provided logger is expected to already carry the node's
// attributes.
func upgradeNode(ctx context.Context, node *topf.Node, opts Options, logger *slog.Logger) error {
	installerImage := node.ConfigProvider().Machine().Install().Image()

	nodeClient, err := node.Client(ctx)
	if err != nil {
		return err
	}
	defer nodeClient.Close()

	//nolint:staticcheck // the non-deprecated replacement (LifecycleClient.Upgrade) is a streaming RPC; migrating is tracked separately
	_, err = nodeClient.MachineClient.Upgrade(ctx, &machine.UpgradeRequest{
		Image:      installerImage,
		Preserve:   true, // talos default since v1.8+
		Force:      opts.Force,
		RebootMode: opts.RebootMode,
	})
	if err != nil {
		return err
	}

	logger.Info("upgrade initiated")

	if err = node.Stabilize(ctx, logger, time.Second*30); err != nil {
		return fmt.Errorf("node didn't stabilize: %w", err)
	}

	return nil
}

func extractSchematicAndVersion(input string) (schematic, version string, err error) {
	// Pattern matches: */<schematic>:v<version>
	pattern := `^.*/([a-zA-Z0-9]+):v?(.+)$`

	re := regexp.MustCompile(pattern)
	matches := re.FindStringSubmatch(input)

	if len(matches) != 3 {
		return "", "", errors.New("invalid format: expected */<schematic>:v?<version>")
	}

	schematic = matches[1]
	version = matches[2]

	return schematic, version, nil
}
