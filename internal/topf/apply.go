// Copyright 2026 PostFinance AG
// SPDX-License-Identifier: MIT

package topf

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/postfinance/topf/internal/interactive"
	"github.com/siderolabs/talos/pkg/machinery/api/machine"
	"github.com/siderolabs/talos/pkg/machinery/config/encoder"
)

// ErrDryRunChangesDetected is returned by Apply when running in dry-run mode
// and changes were detected. Callers can use this to exit with a non-zero status.
var ErrDryRunChangesDetected = errors.New("dry-run: changes detected")

// Apply applies the configuration bundle to the node.
// If dryRun is true, only shows what changes would be applied without actually applying them.
func (n *Node) Apply(ctx context.Context, logger *slog.Logger, confirm, dryRun bool, mode machine.ApplyConfigurationRequest_Mode) (bool, error) {
	logger = logger.With(n.Attrs())

	if n.ConfigBundle == nil {
		return false, errors.New("cannot apply config: config bundle is empty")
	}

	nodeClient, err := n.Client(ctx)
	if err != nil {
		return false, err
	}
	defer nodeClient.Close()

	configBytes, err := n.ConfigProvider().EncodeBytes(encoder.WithComments(encoder.CommentsDisabled), encoder.WithOmitEmpty(true))
	if err != nil {
		return false, err
	}

	logger.Info("dry-run apply")

	// first pass is a dry-run apply
	response, err := nodeClient.MachineClient.ApplyConfiguration(ctx, &machine.ApplyConfigurationRequest{
		Data:   configBytes,
		DryRun: true,
		Mode:   mode,
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

	// in dry-run mode, print the changes and signal that changes were detected
	if dryRun {
		fmt.Fprintln(n.t.Writer(), "     "+strings.ReplaceAll(applyResponse.GetModeDetails(), "\n", "\n     "))

		return false, ErrDryRunChangesDetected
	}

	// ask for user confirmation
	if confirm {
		fmt.Fprintln(n.t.Writer(), "     "+strings.ReplaceAll(applyResponse.GetModeDetails(), "\n", "\n     "))

		if interactive.ConfirmPrompt(fmt.Sprintf("Do you want to apply the above changes to %s (Mode: %s)?", n.Node.Host, applyResponse.GetMode().String())) == 'n' {
			logger.Info("skipping")
			return false, nil
		}
	}

	// actually apply config
	response, err = nodeClient.MachineClient.ApplyConfiguration(ctx, &machine.ApplyConfigurationRequest{
		Data: configBytes,
		Mode: mode,
	})
	if err != nil {
		return false, fmt.Errorf("failed to apply machine config: %w", err)
	}

	applyResponse = response.GetMessages()[0]

	logger.Info("applied machine config", "mode", applyResponse.GetMode())

	return true, nil
}
