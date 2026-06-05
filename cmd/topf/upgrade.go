// Copyright 2026 PostFinance AG
// SPDX-License-Identifier: MIT

package main

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/postfinance/topf/internal/cmd/upgrade"
	"github.com/postfinance/topf/internal/nodepool"
	"github.com/postfinance/topf/internal/topf"
	"github.com/siderolabs/talos/pkg/machinery/api/machine"
	"github.com/urfave/cli/v3"
)

func newUpgradeCmd() *cli.Command {
	return &cli.Command{
		Name:        "upgrade",
		Usage:       "upgrades talos on each node to the desired version",
		Description: `Issues upgrade commands to each node to upgrade Talos to the desired version specified in the installer image.`,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "dry-run",
				Usage:   "only show what upgrades would be performed without actually upgrading",
				Value:   false,
				Sources: cli.EnvVars("TOPF_DRY_RUN"),
			},
			&cli.StringFlag{
				Name:    "batch-size",
				Value:   "1",
				Usage:   "number of worker nodes to upgrade concurrently, as an integer (e.g. \"5\") or a percentage of the total node count (e.g. \"25%\"); control-plane nodes are always upgraded one at a time",
				Sources: cli.EnvVars("TOPF_BATCH_SIZE"),
			},
			&cli.BoolFlag{
				Name:    "force",
				Usage:   "force the upgrade (skip checks on etcd health and members, might lead to data loss)",
				Value:   false,
				Sources: cli.EnvVars("TOPF_FORCE"),
			},
			&cli.StringFlag{
				Name:    "reboot-mode",
				Value:   "default",
				Usage:   "select the reboot mode during upgrade: \"default\" uses kexec, \"powercycle\" does a full reboot",
				Sources: cli.EnvVars("TOPF_REBOOT_MODE"),
			},
		},
		Before: noPositionalArgs,
		Action: func(ctx context.Context, c *cli.Command) error {
			t := MustGetRuntime(ctx)

			rebootMode, err := parseRebootMode(c.String("reboot-mode"))
			if err != nil {
				return err
			}

			batchSize, err := nodepool.ParseBatchSize(c.String("batch-size"))
			if err != nil {
				return err
			}

			err = upgrade.Execute(ctx, t, upgrade.Options{
				DryRun:     c.Bool("dry-run"),
				Force:      c.Bool("force"),
				RebootMode: rebootMode,
				BatchSize:  batchSize,
			})
			if errors.Is(err, topf.ErrDryRunChangesDetected) {
				return cli.Exit(err.Error(), 2)
			}

			return err
		},
	}
}

// rebootModes maps user-facing mode names to their protobuf values.
// https://github.com/siderolabs/talos/blob/main/cmd/talosctl/pkg/talos/helpers/mode.go
var rebootModes = map[string]machine.UpgradeRequest_RebootMode{ //nolint:gochecknoglobals // read-only lookup table
	"default":    machine.UpgradeRequest_DEFAULT,
	"powercycle": machine.UpgradeRequest_POWERCYCLE,
}

func validRebootModes() []string {
	modes := slices.Collect(maps.Keys(rebootModes))
	slices.SortFunc(modes, func(a, b string) int {
		return cmp.Compare(int32(rebootModes[a]), int32(rebootModes[b]))
	})

	return modes
}

func parseRebootMode(mode string) (machine.UpgradeRequest_RebootMode, error) {
	val, ok := rebootModes[mode]
	if !ok {
		return 0, fmt.Errorf("invalid reboot mode %q, valid values: %s", mode, strings.Join(validRebootModes(), ", "))
	}

	return val, nil
}
