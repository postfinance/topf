// Copyright 2026 PostFinance AG
// SPDX-License-Identifier: MIT

package main

import (
	"cmp"
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/postfinance/topf/internal/cmd/upgrade"
	"github.com/siderolabs/talos/pkg/machinery/api/machine"
	"github.com/urfave/cli/v3"
)

func newUpgradeCmd() *cli.Command {
	return &cli.Command{
		Name:        "upgrade",
		Usage:       "upgrades talos on each node to the desired version",
		Description: `Issues upgrade commands to each node to upgrade Talos to the desired version specified in the installer image.`,
		Flags: []cli.Flag{
			confirmFlag(),
			&cli.BoolFlag{
				Name:  "dry-run",
				Usage: "only show what upgrades would be performed without actually upgrading",
				Value: false,
			},
			&cli.BoolFlag{
				Name:  "force",
				Usage: "force the upgrade (skip checks on etcd health and members, might lead to data loss)",
				Value: false,
			},
			&cli.StringFlag{
				Name:  "reboot-mode",
				Value: "default",
				Usage: "select the reboot mode during upgrade: \"default\" uses kexec, \"powercycle\" does a full reboot",
			},
		},
		Before: noPositionalArgs,
		Action: func(ctx context.Context, c *cli.Command) error {
			t := MustGetRuntime(ctx)

			rebootMode, err := parseRebootMode(c.String("reboot-mode"))
			if err != nil {
				return err
			}

			return upgrade.Execute(ctx, t, upgrade.Options{
				Confirm:    c.Bool("confirm"),
				DryRun:     c.Bool("dry-run"),
				Force:      c.Bool("force"),
				RebootMode: rebootMode,
			})
		},
	}
}

// validRebootModes returns the valid reboot mode names derived from the protobuf enum,
// lowercased and sorted by their protobuf numeric value.
func validRebootModes() []string {
	modes := slices.Collect(maps.Keys(machine.UpgradeRequest_RebootMode_value))
	slices.SortFunc(modes, func(a, b string) int {
		return cmp.Compare(machine.UpgradeRequest_RebootMode_value[a], machine.UpgradeRequest_RebootMode_value[b])
	})

	for i, m := range modes {
		modes[i] = strings.ToLower(m)
	}

	return modes
}

func parseRebootMode(mode string) (machine.UpgradeRequest_RebootMode, error) {
	val, ok := machine.UpgradeRequest_RebootMode_value[strings.ToUpper(mode)]
	if !ok {
		return 0, fmt.Errorf("invalid reboot mode %q, valid values: %s", mode, strings.Join(validRebootModes(), ", "))
	}

	return machine.UpgradeRequest_RebootMode(val), nil
}
