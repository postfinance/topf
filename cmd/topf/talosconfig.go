// Copyright 2026 PostFinance AG
// SPDX-License-Identifier: MIT

package main

import (
	"context"
	"fmt"

	"github.com/postfinance/topf/internal/cmd/talosconfig"
	"github.com/urfave/cli/v3"
)

func newTalosconfigCmd() *cli.Command {
	return &cli.Command{
		Name:   "talosconfig",
		Usage:  "generate and save talosconfig from secrets bundle",
		Before: noPositionalArgs,
		Flags: []cli.Flag{
			&cli.DurationFlag{
				Name:  "validity",
				Usage: "validity duration of the client certificate",
				Value: talosconfig.DefaultValidity,
			},
			&cli.StringSliceFlag{
				Name:  "roles",
				Usage: "Talos roles of the client certificate (e.g. os:reader, os:operator)",
				Value: []string{string(talosconfig.DefaultRole)},
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			t := MustGetRuntime(ctx)

			talosCfg, err := talosconfig.Generate(t, cmd.StringSlice("roles"), cmd.Duration("validity"))
			if err != nil {
				return err
			}

			fmt.Println(string(talosCfg))

			return nil
		},
	}
}
