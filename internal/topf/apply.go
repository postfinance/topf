package topf

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/postfinance/topf/internal/interactive"
	"github.com/postfinance/topf/pkg/config"
	"github.com/siderolabs/talos/pkg/machinery/api/machine"
	talosconfig "github.com/siderolabs/talos/pkg/machinery/config"
	"github.com/siderolabs/talos/pkg/machinery/config/encoder"
)

// Apply applies the configuration bundle to the node.
func (n *Node) Apply(ctx context.Context, logger *slog.Logger, confirm bool) (bool, error) {
	logger = logger.With(n.Attrs())

	if n.ConfigBundle == nil {
		return false, errors.New("cannot apply config: config bundle is empty")
	}

	nodeClient, err := n.Client(ctx)
	if err != nil {
		return false, err
	}
	defer nodeClient.Close()

	var provider talosconfig.Provider
	if n.Node.Role == config.RoleControlPlane {
		provider = n.ConfigBundle.ControlPlaneCfg
	} else {
		provider = n.ConfigBundle.WorkerCfg
	}

	configBytes, err := provider.EncodeBytes(encoder.WithComments(encoder.CommentsDisabled), encoder.WithOmitEmpty(true))
	if err != nil {
		return false, err
	}

	logger.Info("dry-run apply")

	// first pass is a dry-run apply
	response, err := nodeClient.MachineClient.ApplyConfiguration(ctx, &machine.ApplyConfigurationRequest{
		Data:   configBytes,
		DryRun: true,
		Mode:   machine.ApplyConfigurationRequest_AUTO,
	})
	if err != nil {
		return false, fmt.Errorf("failed to apply machine config: %w", err)
	}

	applyResponse := response.GetMessages()[0]

	// no better way from API than matching on this until
	if strings.HasSuffix(applyResponse.GetModeDetails(), "\nNo changes.") {
		logger.Info("no changes to apply")
		return false, nil
	}

	if len(applyResponse.GetWarnings()) > 0 {
		logger.Warn("dry-run", "warnings", strings.Join(applyResponse.GetWarnings(), ", "))
	}

	// ask for user confirmation
	if confirm {
		fmt.Println("     " + strings.ReplaceAll(applyResponse.GetModeDetails(), "\n", "\n     "))

		if interactive.ConfirmPrompt(fmt.Sprintf("Do you want to apply the above changes to %s (Mode: %s)?", n.Node.Host, applyResponse.GetMode().String())) == 'n' {
			logger.Info("skipping")
			return false, nil
		}
	}

	// actually apply config
	response, err = nodeClient.MachineClient.ApplyConfiguration(ctx, &machine.ApplyConfigurationRequest{
		Data: configBytes,
		Mode: machine.ApplyConfigurationRequest_AUTO,
	})
	if err != nil {
		return false, fmt.Errorf("failed to apply machine config: %w", err)
	}

	applyResponse = response.GetMessages()[0]

	logger.Info("applied machine config", "mode", applyResponse.GetMode())

	return true, nil
}
