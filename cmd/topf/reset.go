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
			&cli.BoolFlag{
				Name:  "full",
				Value: true,
				Usage: "if the entire disk should be wiped. if false, only STATE and EPHEMERAL are wiped from the install disk",
			},
			&cli.BoolFlag{
				Name:  "graceful",
				Value: false,
				Usage: "if true, attempt to cordon/drain node and leave etcd (if applicable)",
			},
			&cli.BoolFlag{
				Name:  "shutdown",
				Value: false,
				Usage: "if true, shut down machine after reset. otherwise, machine reboots.",
			},
		},
		Description: `This command resets a Talos node to its initial state, wiping the state and ephemeral system partitions and rebooting the node.`,
		Action: func(ctx context.Context, c *cli.Command) error {
			t := MustGetRuntime(ctx)
			logger := t.Logger().With("command", "reset")

			opts := reset.Options{
				Full:     c.Bool("full"),
				Graceful: c.Bool("graceful"),
				Shutdown: c.Bool("shutdown"),
			}

			_, err := reset.Execute(ctx, t, opts)
			if err != nil {
				return err
			}

			logger.Info("reset completed")

			return nil
		},
	}
}
