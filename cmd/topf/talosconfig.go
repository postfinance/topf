package main

import (
	"context"
	"fmt"

	"github.com/postfinance/topf/internal/cmd/talosconfig"
	"github.com/urfave/cli/v3"
)

func newTalosconfigCmd() *cli.Command {
	return &cli.Command{
		Name:  "talosconfig",
		Usage: "generate and save talosconfig from secrets bundle",
		Action: func(ctx context.Context, _ *cli.Command) error {
			t := MustGetRuntime(ctx)

			talosCfg, err := talosconfig.Generate(t)
			if err != nil {
				return err
			}

			fmt.Println(string(talosCfg))

			return nil
		},
	}
}
