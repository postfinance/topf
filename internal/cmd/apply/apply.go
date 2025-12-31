// Package apply contains the logic to apply Talos configurations to cluster nodes
package apply

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/postfinance/topf/internal/topf"
	"github.com/siderolabs/talos/pkg/machinery/resources/runtime"
)

// Options contains the options for the apply execution
type Options struct {
	// Ask for user input before applying
	Confirm bool
}

// Execute applies the Talos configurations to all nodes in the cluster
func Execute(ctx context.Context, t topf.Topf, opts Options) error {
	logger := t.Logger().With("command", "apply")

	nodes, err := t.Nodes(ctx)
	if err != nil {
		return err
	}

	// Pre Checks
	// collect all errors to report them all at once
	abort := false

	for _, node := range nodes {
		logger := logger.With(node.Attrs())

		if node.Error != nil {
			logger.Error("node pre-checks", "error", node.Error)

			abort = true

			continue
		}

		if !slices.Contains([]runtime.MachineStage{runtime.MachineStageRunning, runtime.MachineStageMaintenance, runtime.MachineStageBooting}, node.MachineStatus.Stage) {
			logger.Error("node in unprocessable stage", "stage", node.MachineStatus.Stage.String())

			abort = true

			continue
		}
	}

	if abort {
		return errors.New("aborting due to errors with some nodes")
	}

	// apply configs
	for _, node := range nodes {
		logger := logger.With(node.Attrs())

		applied, err := node.Apply(ctx, logger, opts.Confirm)
		if err != nil {
			return fmt.Errorf("failed to apply config to node %v: %w", node.Node.Host, err)
		}

		// if nothing was applied, skip healthchecks
		if !applied {
			continue
		}

		if err = node.Stabilize(ctx, logger, time.Second*30); err != nil {
			return fmt.Errorf("node didn't stabilize: %w", err)
		}
	}

	return nil
}
