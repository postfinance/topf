// Copyright 2026 PostFinance AG
// SPDX-License-Identifier: MIT

// Package reset contains the logic to reset a cluster back to maintenance mode
package reset

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"

	"github.com/postfinance/topf/internal/interactive"
	"github.com/postfinance/topf/internal/topf"
	"github.com/siderolabs/talos/pkg/machinery/api/machine"
	"github.com/siderolabs/talos/pkg/machinery/resources/runtime"
)

// Options contains the options for the reset execution
type Options struct {
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
	nodes, err := t.FilteredNodes(ctx)
	if err != nil {
		return err
	}

	if len(nodes) == 0 {
		logger.Info("no node to act upon")
		return nil
	}

	var resetNodes []*topf.Node

	for _, n := range nodes {
		switch resetNode(ctx, t, logger.With(n.Attrs()), n, opts) {
		case outcomeSkipped:
			result.SkipCount++
		case outcomeFailed:
			result.FailCount++
		case outcomeSuccess:
			result.SuccessCount++

			resetNodes = append(resetNodes, n)
		}
	}

	logger.Info("reset completed", "result", *result)

	var failErr error
	if result.FailCount > 0 {
		failErr = fmt.Errorf("reset failed on %d node(s)", result.FailCount)
	}

	if opts.WaitForMaintenance && len(resetNodes) > 0 {
		if err := waitForMaintenance(ctx, logger, resetNodes); err != nil {
			return errors.Join(failErr, err)
		}
	}

	return failErr
}

type outcome int

const (
	outcomeSuccess outcome = iota
	outcomeSkipped
	outcomeFailed
)

// resetNode initiates the reset of a single node. The node client is closed
// before returning.
func resetNode(ctx context.Context, t topf.Topf, logger *slog.Logger, n *topf.Node, opts Options) outcome {
	if n.MachineStatus.Stage == runtime.MachineStageMaintenance {
		logger.Info("already in maintenance mode")

		return outcomeSkipped
	}

	nodeClient, err := n.Client(ctx)
	if err != nil {
		logger.Error("couldn't get client", "error", err)

		return outcomeFailed
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
	if t.Confirm() {
		message := fmt.Sprintf("Do you want to reset %s ?", n.Node.Host)
		if interactive.ConfirmPrompt(message) == 'n' {
			logger.Info("skipping")

			return outcomeSkipped
		}
	}

	_, err = nodeClient.MachineClient.Reset(ctx, &machine.ResetRequest{
		SystemPartitionsToWipe: partitions,
		Graceful:               opts.Graceful,
		Reboot:                 !opts.Shutdown,
	})
	if err != nil {
		logger.Error("failed to initiate reset", "error", err)

		return outcomeFailed
	}

	logger.Info("reset initiated")

	return outcomeSuccess
}

// waitForMaintenance waits in parallel for all nodes to reach maintenance mode.
func waitForMaintenance(ctx context.Context, logger *slog.Logger, nodes []*topf.Node) error {
	logger.Info("waiting for nodes to reach maintenance mode", "count", len(nodes))

	var wg sync.WaitGroup

	errs := make(chan error, len(nodes))

	for _, n := range nodes {
		wg.Add(1)

		go func(n *topf.Node) {
			defer wg.Done()

			logger := logger.With(n.Attrs())

			if err := n.WaitForMaintenance(ctx, logger); err != nil {
				logger.Error("failed waiting for maintenance mode", "error", err)

				errs <- err

				return
			}

			logger.Info("node is in maintenance mode")
		}(n)
	}

	wg.Wait()
	close(errs)

	var waitErrs []error
	for err := range errs {
		waitErrs = append(waitErrs, err)
	}

	return errors.Join(waitErrs...)
}
