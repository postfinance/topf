package apply

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
)

// bootstrap initiates the ETCD bootstrap process and waits for nodes to stabilize
func bootstrap(ctx context.Context, logger *slog.Logger, nodes []*topf.Node) error {
	if len(nodes) == 0 || nodes[0].Node.Role != config.RoleControlPlane {
		return errors.New("bootstrap requires at least 1 control plane node")
	}

	// Initiate bootstrap
	err := retry.Constant(time.Minute*10, retry.WithErrorLogging(logger.Enabled(ctx, slog.LevelDebug))).RetryWithContext(ctx, func(ctx context.Context) error {
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

		logger.Info("etcd bootstrap initiated")

		return nil
	})
	if err != nil {
		return fmt.Errorf("bootstrap not successful: %w", err)
	}

	return nil
}
