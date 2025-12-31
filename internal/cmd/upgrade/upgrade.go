// Package upgrade contains the logic to upgrade Talos OS on cluster nodes
package upgrade

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"time"

	"github.com/postfinance/topf/internal/interactive"
	"github.com/postfinance/topf/internal/topf"
	"github.com/siderolabs/talos/pkg/machinery/api/machine"
	"github.com/siderolabs/talos/pkg/machinery/resources/runtime"
)

// Options contains the options for the upgrade execution
type Options struct {
	Confirm bool

	// Talos upgrade options
	Force    bool
	Preserve bool
}

// Execute performs the Talos OS upgrades for all nodes in the cluster
func Execute(ctx context.Context, t topf.Topf, opts Options) error {
	logger := t.Logger().With("command", "upgrade")

	// Gather node information
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

		if !slices.Contains([]runtime.MachineStage{runtime.MachineStageRunning}, node.MachineStatus.Stage) {
			logger.Error("node must be 'running' for upgrade", "stage", node.MachineStatus.Stage.String())

			abort = true

			continue
		}
	}

	if abort {
		return errors.New("aborting due to errors with some nodes")
	}

	for _, node := range nodes {
		logger := logger.With(node.Attrs())

		installerImage := node.ConfigBundle.WorkerCfg.Machine().Install().Image()

		schematic, talosVersion, err := extractSchematicAndVersion(installerImage)
		if err != nil {
			return fmt.Errorf("couldn't extract schematic and version from installer image '%s': %w", installerImage, err)
		}

		upgradeRequired := node.TalosVersion != talosVersion || node.Schematic != schematic
		if !upgradeRequired {
			logger.Info("no upgrade required")
			continue
		}

		logger.Info("upgrade required",
			"schematic_actual", node.Schematic,
			"schematic_desired", schematic,
			"version_actual", node.TalosVersion,
			"version_desired", talosVersion,
			"installer", installerImage)

		// ask for user confirmation
		if opts.Confirm {
			if interactive.ConfirmPrompt(fmt.Sprintf("Do you want to upgrade node %s with installer %s? This will reboot the node.", node.Node.Host, installerImage)) == 'n' {
				logger.Info("skipping upgrade")
				continue
			}
		}

		nodeClient, err := node.Client(ctx)
		if err != nil {
			return err
		}
		defer nodeClient.Close()

		_, err = nodeClient.MachineClient.Upgrade(ctx, &machine.UpgradeRequest{
			Image:      installerImage,
			Preserve:   len(nodes) == 1,                   // TODO: make this a param
			Force:      opts.Force,                        // TODO: make this a param
			RebootMode: machine.UpgradeRequest_POWERCYCLE, // TODO: make this a param
		})
		if err != nil {
			return err
		}

		logger.Info("upgrade initiated")

		if err = node.Stabilize(ctx, logger, time.Second*30); err != nil {
			return fmt.Errorf("node didn't stabilize: %w", err)
		}
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
