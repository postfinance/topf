package main

import (
	"context"

	"github.com/postfinance/topf/internal/cmd/apply"
	"github.com/urfave/cli/v3"
)

func newApplyCmd() *cli.Command {
	return &cli.Command{
		Name:        "apply",
		Usage:       "apply configuration changes to a running cluster",
		Description: `This command applies configuration changes to nodes in a running Talos cluster.`,
		Flags: []cli.Flag{
			confirmFlag(),
		},
		Before: noPositionalArgs,
		Action: func(ctx context.Context, c *cli.Command) error {
			t := MustGetRuntime(ctx)

			return apply.Execute(ctx, t, apply.Options{
				Confirm: c.Bool("confirm"),
			})
		},
	}
}
