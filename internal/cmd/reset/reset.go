// Package reset contains the logic to reset a cluster back to maintenance mode
package reset

import (
	"context"

	"github.com/postfinance/topf/internal/topf"
	"github.com/siderolabs/talos/pkg/machinery/api/machine"
	"github.com/siderolabs/talos/pkg/machinery/resources/runtime"
)

// Options contains the options for the reset execution
type Options struct {
	// Whether to perform a full wipe of the installation disk. If false, only
	// STATE and EPHEMERAL partitions are wiped.
	Full     bool
	Graceful bool
	Shutdown bool
}

// Result contains the result of the reset operation
type Result struct {
	SuccessCount int
	SkipCount    int
	FailCount    int
}

// Execute performs the reset operation on all nodes in the cluster
func Execute(ctx context.Context, t topf.Topf, opts Options) error {
	logger := t.Logger().With("command", "reset")
	result := &Result{}

	// Gather node information
	nodes, err := t.Nodes(ctx)
	if err != nil {
		return err
	}

	if len(nodes) == 0 {
		logger.Info("no node to act upon")
		return nil
	}

	for _, n := range nodes {
		logger := logger.With(n.Attrs())

		if n.MachineStatus.Stage == runtime.MachineStageMaintenance {
			logger.Info("already in maintenance mode")

			result.SkipCount++

			continue
		}

		nodeClient, err := n.Client(ctx)
		if err != nil {
			logger.Info("couldn't get client", "error", err)

			result.SkipCount++

			continue
		}
		defer nodeClient.Close()

		partitions := []*machine.ResetPartitionSpec{
			{Label: "STATE", Wipe: true},
			{Label: "EPHEMERAL", Wipe: true},
		}

		// full wipe blindly wipes all partitions
		if opts.Full {
			partitions = nil
		}

		_, err = nodeClient.MachineClient.Reset(ctx, &machine.ResetRequest{
			SystemPartitionsToWipe: partitions,
			Graceful:               opts.Graceful,
			Reboot:                 !opts.Shutdown,
		})
		if err != nil {
			logger.Error("failed to initiate reset", "error", err)

			result.FailCount++

			continue
		}

		logger.Info("reset initiated")

		result.SuccessCount++
	}

	logger.Info("reset completed", "result", *result)

	return nil
}
