package main

import (
	"context"

	"github.com/postfinance/topf/internal/cmd/bootstrap"
	"github.com/urfave/cli/v3"
)

func newBootstrapCmd() *cli.Command {
	return &cli.Command{
		Name:        "bootstrap",
		Usage:       "bootstrap a cluster",
		Description: `bootstrap a a fresh talos cluster`,
		Flags: []cli.Flag{
			confirmFlag(),
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			t := MustGetRuntime(ctx)

			return bootstrap.Execute(ctx, t, bootstrap.Options{
				Confirm: c.Bool("confirm"),
			})
		},
	}
}
