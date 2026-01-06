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
			&cli.BoolFlag{
				Name:  "auto-bootstrap",
				Usage: "automatically run the bootstrap command after an apply",
				Value: false,
			},
			&cli.BoolFlag{
				Name:  "skip-problematic-nodes",
				Usage: "skip nodes with pre-flight errors and continue with healthy nodes",
				Value: false,
			},
			&cli.BoolFlag{
				Name:  "skip-post-apply-checks",
				Usage: "skip post-apply stabilization and health checks",
				Value: false,
			},
		},
		Before: noPositionalArgs,
		Action: func(ctx context.Context, c *cli.Command) error {
			t := MustGetRuntime(ctx)

			return apply.Execute(ctx, t, apply.Options{
				Confirm:              c.Bool("confirm"),
				AutoBootstrap:        c.Bool("auto-bootstrap"),
				SkipProblematicNodes: c.Bool("skip-problematic-nodes"),
				SkipPostApplyChecks:  c.Bool("skip-post-apply-checks"),
			})
		},
	}
}
