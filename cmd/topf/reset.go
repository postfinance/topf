// Copyright 2026 PostFinance AG
// SPDX-License-Identifier: MIT

package main

import (
	"context"

	"github.com/postfinance/topf/internal/cmd/reset"
	"github.com/urfave/cli/v3"
)

func newResetCmd() *cli.Command {
	return &cli.Command{
		Name:  "reset",
		Usage: "reset talos node(s) to maintenance mode",
		Flags: []cli.Flag{
			confirmFlag(),
			&cli.BoolFlag{
				Name:    "full",
				Value:   true,
				Usage:   "if the entire disk should be wiped. if false, only STATE and EPHEMERAL are wiped from the install disk",
				Sources: cli.EnvVars("TOPF_FULL"),
			},
			&cli.BoolFlag{
				Name:    "graceful",
				Value:   false,
				Usage:   "if true, attempt to cordon/drain node and leave etcd (if applicable)",
				Sources: cli.EnvVars("TOPF_GRACEFUL"),
			},
			&cli.BoolFlag{
				Name:    "shutdown",
				Value:   false,
				Usage:   "if true, shut down machine after reset. otherwise, machine reboots.",
				Sources: cli.EnvVars("TOPF_SHUTDOWN"),
			},
			&cli.BoolFlag{
				Name:    "wait-for-maintenance",
				Value:   false,
				Usage:   "wait for all reset nodes to reach maintenance mode",
				Sources: cli.EnvVars("TOPF_WAIT_FOR_MAINTENANCE"),
			},
		},
		Description: `This command resets a Talos node to its initial state, wiping the state and ephemeral system partitions and rebooting the node.`,
		Before:      noPositionalArgs,
		Action: func(ctx context.Context, c *cli.Command) error {
			t := MustGetRuntime(ctx)

			opts := reset.Options{
				Confirm:            c.Bool("confirm"),
				Full:               c.Bool("full"),
				Graceful:           c.Bool("graceful"),
				Shutdown:           c.Bool("shutdown"),
				WaitForMaintenance: c.Bool("wait-for-maintenance"),
			}

			return reset.Execute(ctx, t, opts)
		},
	}
}
