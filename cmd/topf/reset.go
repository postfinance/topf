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

	"github.com/postfinance/topf/internal/cmd/reset"
	"github.com/siderolabs/talos/pkg/machinery/api/machine"
	"github.com/urfave/cli/v3"
)

func newResetCmd() *cli.Command {
	return &cli.Command{
		Name:        "reset",
		Usage:       "reset talos node(s)",
		Description: `This command resets system partitions and selected disks of a Talos node. Followed by a reboot or shutdown.`,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:        "wipe-state-and-ephemeral",
				Usage:       "if true, only STATE and EPHEMERAL partitions are wiped",
				Sources:     cli.EnvVars("TOPF_WIPE_STATE_EPHEMERAL"),
				DefaultText: defaultTextTrue,
			},
			&cli.BoolFlag{
				Name:        "graceful",
				Value:       true,
				Usage:       "if true, attempt to cordon/drain node and leave etcd (if applicable)",
				Sources:     cli.EnvVars("TOPF_GRACEFUL"),
				DefaultText: defaultTextTrue,
			},
			&cli.BoolFlag{
				Name:        "shutdown",
				Usage:       "if true, shut down machine after reset. otherwise, machine reboots.",
				Sources:     cli.EnvVars("TOPF_SHUTDOWN"),
				DefaultText: defaultTextFalse,
			},
			&cli.BoolFlag{
				Name:        "wait-for-maintenance",
				Usage:       "wait for all reset nodes to reach maintenance mode",
				Sources:     cli.EnvVars("TOPF_WAIT_FOR_MAINTENANCE"),
				DefaultText: defaultTextFalse,
			},
			&cli.StringFlag{
				Name:        "wipe-mode",
				Usage:       "disk reset mode: " + strings.Join(validWipeModes(), ", "),
				Sources:     cli.EnvVars("TOPF_WIPE_MODE"),
				DefaultText: "all",
			},
			&cli.StringSliceFlag{
				Name:    "system-labels-to-wipe",
				Usage:   "if set, just wipe selected system disk partitions by label but keep other partitions intact",
				Sources: cli.EnvVars("TOPF_WIPE_SYSTEM_LABELS"),
			},
			&cli.StringSliceFlag{
				Name:    "user-disks-to-wipe",
				Usage:   "if set, wipes defined devices in the list",
				Sources: cli.EnvVars("TOPF_WIPE_USER_DISKS"),
			},
		},
		Before: noPositionalArgs,
		Action: func(ctx context.Context, c *cli.Command) error {
			t := MustGetRuntime(ctx)

			mode, err := parseWipeMode(c.String("wipe-mode"))
			if err != nil {
				return err
			}

			opts := reset.Options{
				WipeStateAndEphemeral: c.Bool("wipe-state-and-ephemeral"),
				Graceful:              c.Bool("graceful"),
				Shutdown:              c.Bool("shutdown"),
				WaitForMaintenance:    c.Bool("wait-for-maintenance"),
				WipeMode:              mode,
				SystemLabels:          c.StringSlice("system-labels-to-wipe"),
				UserDisks:             c.StringSlice("user-disks-to-wipe"),
			}

			return reset.Execute(ctx, t, opts)
		},
	}
}

// wipeModes maps user-facing mode names to their protobuf values.
// https://github.com/siderolabs/talos/blob/main/cmd/talosctl/cmd/talos/reset.go
//
//nolint:gochecknoglobals // read-only lookup table
var wipeModes = map[string]machine.ResetRequest_WipeMode{
	"all":         machine.ResetRequest_ALL,
	"system-disk": machine.ResetRequest_SYSTEM_DISK,
	"user-disk":   machine.ResetRequest_USER_DISKS,
}

func validWipeModes() []string {
	modes := slices.Collect(maps.Keys(wipeModes))
	slices.SortFunc(modes, func(a, b string) int {
		return cmp.Compare(int32(wipeModes[a]), int32(wipeModes[b]))
	})

	return modes
}

func parseWipeMode(mode string) (machine.ResetRequest_WipeMode, error) {
	val, ok := wipeModes[mode]
	if !ok {
		return 0, fmt.Errorf("invalid wipe mode %q, valid values: %s", mode, strings.Join(validWipeModes(), ", "))
	}

	return val, nil
}
