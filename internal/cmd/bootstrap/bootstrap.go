// Package bootstrap contains the logic to bootstrap a Talos cluster
package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/postfinance/topf/internal/topf"
	"github.com/postfinance/topf/pkg/config"
	"github.com/siderolabs/go-retry/retry"
	"github.com/siderolabs/talos/pkg/machinery/api/machine"
	"github.com/siderolabs/talos/pkg/machinery/resources/runtime"
	"golang.org/x/sync/errgroup"
)

// Options contains the options for the bootstrap execution
type Options struct {
	// Ask for user input before applying
	Confirm bool
}

// Execute bootstraps the Talos cluster by applying configurations to all nodes
func Execute(ctx context.Context, t topf.Topf, opts Options) error {
	logger := t.Logger().With("command", "bootstrap")

	if len(t.Config().Nodes) == 0 || t.Config().Nodes[0].Role != config.RoleControlPlane {
		return errors.New("bootstrap requires at least 1 control plane node")
	}

	// Gather node information
	nodes, err := t.Nodes(ctx)
	if err != nil {
		return err
	}

	// Pre Checks
	if ok := preChecks(logger, nodes); !ok {
		return errors.New("aborting due to errors with some nodes")
	}

	// apply configs
	for _, node := range nodes {
		_, err := node.Apply(ctx, logger, opts.Confirm)
		if err != nil {
			return fmt.Errorf("failed to apply config to node %v: %w", node.Node.Host, err)
		}
	}

	// bootstrap
	err = retry.Constant(time.Minute*10, retry.WithErrorLogging(logger.Enabled(ctx, slog.LevelDebug))).RetryWithContext(ctx, func(ctx context.Context) error {
		// bootstrap needs to be called on any CP node, we take the first one
		nodeClient, err := nodes[0].Client(ctx)
		if err != nil {
			return retry.ExpectedErrorf("couln't get client for bootstrap call: %w", err)
		}
		defer nodeClient.Close()

		_, err = nodeClient.MachineClient.Bootstrap(ctx, &machine.BootstrapRequest{})
		if err != nil {
			return retry.ExpectedError(err)
		}

		logger.Info("ETCD bootstrap initiated")

		return nil
	})
	if err != nil {
		return fmt.Errorf("bootstrap not successful: %w", err)
	}

	// wait for healthy nodes
	eg := errgroup.Group{}

	for _, node := range nodes {
		eg.Go(func() error {
			logger.Debug("waiting for node to be stable")
			return node.Stabilize(ctx, logger, time.Second*30)
		})
	}

	return eg.Wait()
}

func preChecks(logger *slog.Logger, nodes []*topf.Node) (ok bool) {
	ok = true

	for _, node := range nodes {
		logger := logger.With("node", node.Node.Host)

		if node.Error != nil {
			logger.Error("couldn't collect information for node", "error", node.Error)

			ok = false

			continue
		}

		if node.MachineStatus.Stage != runtime.MachineStageMaintenance {
			logger.Error("node not in maintenance mode", "stage", node.MachineStatus.Stage.String())

			ok = false

			continue
		}

		if !node.MachineStatus.Status.Ready {
			logger.Error("node not ready", "unmet conditions", node.MachineStatus.Status.UnmetConditions)

			ok = false

			continue
		}
	}

	return ok
}
