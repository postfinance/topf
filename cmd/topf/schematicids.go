// Copyright 2026 PostFinance AG
// SPDX-License-Identifier: MIT

package main

import (
	"context"
	"os"

	"github.com/postfinance/topf/internal/cmd/schematicids"
	"github.com/urfave/cli/v3"
)

func newSchematicIDsCmd() *cli.Command {
	return &cli.Command{
		Name:        "schematic-ids",
		Usage:       "output resolved schematic IDs for all nodes",
		Description: `Resolves all schematic IDs (including @-prefixed file references) and prints them to stdout, one per line.`,
		Before:      noPositionalArgs,
		Action: func(ctx context.Context, _ *cli.Command) error {
			t := MustGetRuntime(ctx)

			return schematicids.Execute(ctx, t, os.Stdout)
		},
	}
}
