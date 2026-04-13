// Copyright 2026 PostFinance AG
// SPDX-License-Identifier: MIT

// Package reset contains the logic to reset a cluster back to maintenance mode
package reset

import (
	"context"
	"fmt"
	"sync"

	"github.com/postfinance/topf/internal/interactive"
	"github.com/postfinance/topf/internal/topf"
	"github.com/siderolabs/talos/pkg/machinery/api/machine"
	"github.com/siderolabs/talos/pkg/machinery/resources/runtime"
)

// Options contains the options for the reset execution
type Options struct {
	// Ask for user input before resetting
	Confirm bool
	// Whether to perform a full wipe of the installation disk. If false, only
	// STATE and EPHEMERAL partitions are wiped.
	Full               bool
	Graceful           bool
	Shutdown           bool
	WaitForMaintenance bool
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

	var resetNodes []*topf.Node

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

		// ask for user confirmation
		if opts.Confirm {
			message := fmt.Sprintf("Do you want to reset %s ?", n.Node.Host)
			if interactive.ConfirmPrompt(message) == 'n' {
				logger.Info("skipping")

				result.SkipCount++

				continue
			}
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

		resetNodes = append(resetNodes, n)
	}

	logger.Info("reset completed", "result", *result)

	if opts.WaitForMaintenance && len(resetNodes) > 0 {
		logger.Info("waiting for nodes to reach maintenance mode", "count", len(resetNodes))

		var wg sync.WaitGroup

		for _, n := range resetNodes {
			wg.Add(1)

			go func(n *topf.Node) {
				defer wg.Done()

				logger := logger.With(n.Attrs())

				if err := n.WaitForMaintenance(ctx, logger); err != nil {
					logger.Error("failed waiting for maintenance mode", "error", err)
					return
				}

				logger.Info("node is in maintenance mode")
			}(n)
		}

		wg.Wait()
	}

	return nil
}
