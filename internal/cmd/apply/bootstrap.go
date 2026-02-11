// Copyright 2026 PostFinance AG
// SPDX-License-Identifier: MIT

package apply

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/postfinance/topf/internal/topf"
	"github.com/postfinance/topf/pkg/config"
	"github.com/siderolabs/go-retry/retry"
	"github.com/siderolabs/talos/pkg/machinery/api/machine"
	"github.com/siderolabs/talos/pkg/machinery/client"
)

// bootstrap initiates the ETCD bootstrap process and waits for nodes to stabilize
func bootstrap(ctx context.Context, logger *slog.Logger, nodes []*topf.Node) error {
	if len(nodes) == 0 || nodes[0].Node.Role != config.RoleControlPlane {
		logger.Warn("bootstrap requires at least 1 control plane node, not sending bootstrap request")

		return nil
	}

	logger.Info("starting bootstrap process", "timeout", "10 minutes")

	alreadyBootstrapped := false

	err := retry.Constant(time.Minute*10,
		retry.WithUnits(time.Second*5),
		retry.WithAttemptTimeout(15*time.Second),
		retry.WithErrorLogging(logger.Enabled(ctx, slog.LevelDebug)),
	).RetryWithContext(ctx, func(ctx context.Context) error {
		bootstrapped, err := tryBootstrap(ctx, logger, nodes[0])
		if err != nil {
			return err
		}

		alreadyBootstrapped = bootstrapped

		return nil
	})
	if err != nil {
		return fmt.Errorf("bootstrap failed: %w", err)
	}

	if !alreadyBootstrapped {
		logger.Info("etcd bootstrap completed successfully")
	}

	return nil
}

// tryBootstrap attempts a single bootstrap operation.
// Returns (true, nil) if already bootstrapped, (false, nil) if bootstrap succeeded,
// or a retryable error if bootstrap should be retried.
func tryBootstrap(ctx context.Context, logger *slog.Logger, node *topf.Node) (alreadyBootstrapped bool, err error) {
	nodeClient, err := node.Client(ctx)
	if err != nil {
		return false, retry.ExpectedErrorf("couldn't get client for bootstrap: %w", err)
	}
	defer nodeClient.Close()

	etcdState, err := getEtcdState(ctx, nodeClient)
	if err != nil {
		return false, err
	}

	logger.Debug("etcd service state", "state", etcdState)

	switch etcdState {
	case "Preparing":
		logger.Info("etcd is in Preparing state, attempting bootstrap")

		if _, err = nodeClient.MachineClient.Bootstrap(ctx, &machine.BootstrapRequest{}); err != nil {
			return false, retry.ExpectedError(err)
		}

		return false, nil

	case "Running":
		if memberCount := getEtcdMemberCount(ctx, nodeClient); memberCount > 0 {
			logger.Info("etcd already bootstrapped", "member_count", memberCount)
			return true, nil
		}
	}

	return false, retry.ExpectedErrorf("etcd not ready for bootstrap, state: %s", etcdState)
}

func getEtcdState(ctx context.Context, c *client.Client) (string, error) {
	etcdSvc, err := c.ServiceInfo(ctx, "etcd")
	if err != nil {
		return "", retry.ExpectedErrorf("couldn't get etcd service info: %w", err)
	}

	if len(etcdSvc) > 0 && etcdSvc[0].Service != nil {
		return etcdSvc[0].Service.GetState(), nil
	}

	return "", nil
}

func getEtcdMemberCount(ctx context.Context, c *client.Client) int {
	resp, err := c.MachineClient.EtcdMemberList(ctx, &machine.EtcdMemberListRequest{})
	if err != nil {
		return 0
	}

	for _, msg := range resp.GetMessages() {
		if count := len(msg.GetMembers()); count > 0 {
			return count
		}
	}

	return 0
}
