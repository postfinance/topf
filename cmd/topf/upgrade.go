package main

import (
	"context"

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
		},
		Before: noPositionalArgs,
		Action: func(ctx context.Context, c *cli.Command) error {
			t := MustGetRuntime(ctx)

			return upgrade.Execute(ctx, t, upgrade.Options{
				Confirm: c.Bool("confirm"),
				DryRun:  c.Bool("dry-run"),
				Force:   c.Bool("force"),
			})
		},
	}
}
