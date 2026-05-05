// Copyright 2026 PostFinance AG
// SPDX-License-Identifier: MIT

package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/postfinance/topf/internal/topf"
	"github.com/siderolabs/talos/pkg/machinery/config/encoder"
	"github.com/urfave/cli/v3"
)

func newRenderCmd() *cli.Command {
	return &cli.Command{
		Name:        "render",
		Usage:       "render machine configs without applying them",
		Description: `Generates machine config files for all nodes using local files (topf.yaml and patches). With --online, queries live nodes for their actual running Talos version.`,
		Before:      noPositionalArgs,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "output",
				Aliases:     []string{"o"},
				Usage:       "directory to write machine configs",
				Value:       "./output",
				DefaultText: "./output",
			},
			&cli.BoolFlag{
				Name:  "online",
				Usage: "query live nodes for their running Talos version",
			},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			t := MustGetRuntime(ctx)

			nodes, err := t.Render(ctx, c.Bool("online"))
			if err != nil {
				return err
			}

			return writeMachineConfigs(nodes, c.String("output"))
		},
	}
}

func writeMachineConfigs(nodes []*topf.Node, outputDir string) error {
	if err := os.MkdirAll(outputDir, 0o750); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	var errs []error

	for _, node := range nodes {
		if node.ConfigBundle == nil {
			err := node.Error
			if err == nil {
				err = errors.New("no config bundle available")
			}

			errs = append(errs, fmt.Errorf("%s: %w", node.Node.Host, err))

			continue
		}

		configBytes, err := node.ConfigProvider().EncodeBytes(
			encoder.WithComments(encoder.CommentsDisabled),
			encoder.WithOmitEmpty(true),
		)
		if err != nil {
			return fmt.Errorf("failed to encode config for %s: %w", node.Node.Host, err)
		}

		outputPath := filepath.Join(outputDir, node.Node.Host+".yaml")
		if err := os.WriteFile(outputPath, configBytes, 0o600); err != nil {
			return fmt.Errorf("failed to write config for %s: %w", node.Node.Host, err)
		}

		fmt.Fprintf(os.Stdout, "Wrote machine config for %s to %s\n", node.Node.Host, outputPath)
	}

	return errors.Join(errs...)
}
