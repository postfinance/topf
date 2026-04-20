// Copyright 2026 PostFinance AG
// SPDX-License-Identifier: MIT

package main

import (
	"cmp"
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/postfinance/topf/internal/cmd/apply"
	"github.com/siderolabs/talos/pkg/machinery/api/machine"
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
				Name:  "dry-run",
				Usage: "only show what changes would be applied without actually applying them",
				Value: false,
			},
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
			&cli.BoolFlag{
				Name:  "allow-not-ready",
				Usage: "allow applying to nodes that are not ready (have unmet conditions)",
				Value: false,
			},
			&cli.StringFlag{
				Name:  "mode",
				Value: "auto",
				Usage: "apply mode: " + strings.Join(validApplyModes(), ", "),
			},
		},
		Before: noPositionalArgs,
		Action: func(ctx context.Context, c *cli.Command) error {
			t := MustGetRuntime(ctx)

			mode, err := parseApplyMode(c.String("mode"))
			if err != nil {
				return err
			}

			return apply.Execute(ctx, t, apply.Options{
				Confirm:              c.Bool("confirm"),
				DryRun:               c.Bool("dry-run"),
				AutoBootstrap:        c.Bool("auto-bootstrap"),
				SkipProblematicNodes: c.Bool("skip-problematic-nodes"),
				SkipPostApplyChecks:  c.Bool("skip-post-apply-checks"),
				AllowNotReady:        c.Bool("allow-not-ready"),
				Mode:                 mode,
			})
		},
	}
}

// applyModes maps user-facing mode names to their protobuf values.
// https://github.com/siderolabs/talos/blob/main/cmd/talosctl/pkg/talos/helpers/mode.go
var applyModes = map[string]machine.ApplyConfigurationRequest_Mode{ //nolint:gochecknoglobals // read-only lookup table
	"reboot":    machine.ApplyConfigurationRequest_REBOOT,
	"auto":      machine.ApplyConfigurationRequest_AUTO,
	"no-reboot": machine.ApplyConfigurationRequest_NO_REBOOT,
	"staged":    machine.ApplyConfigurationRequest_STAGED,
	"try":       machine.ApplyConfigurationRequest_TRY,
}

func validApplyModes() []string {
	modes := slices.Collect(maps.Keys(applyModes))
	slices.SortFunc(modes, func(a, b string) int {
		return cmp.Compare(int32(applyModes[a]), int32(applyModes[b]))
	})

	return modes
}

func parseApplyMode(mode string) (machine.ApplyConfigurationRequest_Mode, error) {
	val, ok := applyModes[mode]
	if !ok {
		return 0, fmt.Errorf("invalid apply mode %q, valid values: %s", mode, strings.Join(validApplyModes(), ", "))
	}

	return val, nil
}
