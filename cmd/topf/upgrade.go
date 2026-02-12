// Copyright 2026 PostFinance AG
// SPDX-License-Identifier: MIT

package main

import (
	"context"
	"fmt"
	"slices"

	"github.com/postfinance/topf/internal/cmd/upgrade"
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
				Validator: func(mode string) error {
					modes := []string{"default", "powercycle"}
					if !slices.Contains(modes, mode) {
						return fmt.Errorf("%s is not a valid reboot mode", mode)
					}
					return nil
				},
			},
		},
		Before: noPositionalArgs,
		Action: func(ctx context.Context, c *cli.Command) error {
			t := MustGetRuntime(ctx)

			return upgrade.Execute(ctx, t, upgrade.Options{
				Confirm:    c.Bool("confirm"),
				DryRun:     c.Bool("dry-run"),
				Force:      c.Bool("force"),
				RebootMode: c.String("reboot-mode"),
			})
		},
	}
}
