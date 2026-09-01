// Copyright 2026 PostFinance AG
// SPDX-License-Identifier: MIT

package main

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
	"time"

	"github.com/postfinance/topf/internal/cmd/upgrade"
	"github.com/postfinance/topf/internal/nodepool"
	"github.com/postfinance/topf/internal/topf"
	"github.com/siderolabs/talos/cmd/talosctl/pkg/talos/nodedrain"
	"github.com/siderolabs/talos/pkg/machinery/api/machine"
	"github.com/urfave/cli/v3"
)

func newUpgradeCmd() *cli.Command {
	return &cli.Command{
		Name:        "upgrade",
		Usage:       "upgrades talos on each node to the desired version",
		Description: `Issues upgrade commands to each node to upgrade Talos to the desired version specified in the installer image.`,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:        "dry-run",
				Usage:       "only show what upgrades would be performed without actually upgrading",
				Sources:     cli.EnvVars("TOPF_DRY_RUN"),
				DefaultText: defaultTextFalse,
			},
			&cli.StringFlag{
				Name:    "max-parallel",
				Value:   "1",
				Usage:   "number of worker nodes to upgrade concurrently, as an integer (e.g. \"5\") or a percentage of the total node count (e.g. \"25%\"); control-plane nodes are always upgraded one at a time",
				Sources: cli.EnvVars("TOPF_MAX_PARALLEL"),
			},
			&cli.StringFlag{
				Name:    "reboot-mode",
				Value:   "default",
				Usage:   "select the reboot mode during upgrade: \"default\" uses kexec, \"powercycle\" does a full reboot",
				Sources: cli.EnvVars("TOPF_REBOOT_MODE"),
			},
			&cli.BoolFlag{
				Name:        "drain",
				Usage:       "cordon and drain the Kubernetes node before rebooting, then uncordon after the node becomes Ready again",
				Value:       true,
				Sources:     cli.EnvVars("TOPF_DRAIN"),
				DefaultText: defaultTextTrue,
			},
			&cli.DurationFlag{
				Name:    "drain-timeout",
				Usage:   "maximum time to wait for pod evictions (and, with --delete-if-eviction-fails, deletions) to complete during drain",
				Value:   nodedrain.DefaultDrainTimeout,
				Sources: cli.EnvVars("TOPF_DRAIN_TIMEOUT"),
			},
			&cli.BoolFlag{
				Name:        "delete-if-eviction-fails",
				Usage:       "if graceful drain fails (e.g. a PodDisruptionBudget blocks eviction), retry by deleting pods directly (DELETE instead of EVICT, bypassing PDBs); uses --drain-timeout for the delete fallback",
				Sources:     cli.EnvVars("TOPF_DELETE_IF_EVICTION_FAILS"),
				DefaultText: defaultTextFalse,
			},
			&cli.DurationFlag{
				Name:    "stabilization-duration",
				Usage:   "how long a node must stay ready after rebooting before it is considered stable",
				Value:   time.Second * 30,
				Sources: cli.EnvVars("TOPF_STABILIZATION_DURATION"),
			},
			&cli.BoolFlag{
				Name:        "force",
				Usage:       "skip etcd health checks during upgrade; only applies to nodes running Talos < 1.13 (legacy MachineService.Upgrade RPC); has no effect on Talos >= 1.13, where the LifecycleService.Upgrade RPC validates etcd health server-side",
				Sources:     cli.EnvVars("TOPF_FORCE"),
				DefaultText: defaultTextFalse,
			},
			&cli.BoolFlag{
				Name:        "stage",
				Usage:       "install upgrade artifacts without rebooting; the node can be rebooted later to complete the upgrade (Talos >= 1.13 only)",
				Sources:     cli.EnvVars("TOPF_STAGE"),
				DefaultText: defaultTextFalse,
			},
			&cli.StringSliceFlag{
				Name:    "stage-label",
				Usage:   "Kubernetes node label to apply after staging an upgrade (key=value); can be repeated; requires --stage",
				Sources: cli.EnvVars("TOPF_STAGE_LABEL"),
			},
			&cli.StringSliceFlag{
				Name:    "stage-annotation",
				Usage:   "Kubernetes node annotation to apply after staging an upgrade (key=value); can be repeated; requires --stage",
				Sources: cli.EnvVars("TOPF_STAGE_ANNOTATION"),
			},
			&cli.StringSliceFlag{
				Name:    "stage-taint",
				Usage:   "Kubernetes node taint to apply after staging an upgrade (key[=value]:Effect); can be repeated; requires --stage",
				Sources: cli.EnvVars("TOPF_STAGE_TAINT"),
			},
		},
		Before: noPositionalArgs,
		Action: func(ctx context.Context, c *cli.Command) error {
			t := MustGetRuntime(ctx)

			rebootMode, err := parseRebootMode(c.String("reboot-mode"))
			if err != nil {
				return err
			}

			maxParallel, err := nodepool.ParseMaxParallel(c.String("max-parallel"))
			if err != nil {
				return err
			}

			err = upgrade.Execute(ctx, t, upgrade.Options{
				DryRun:                c.Bool("dry-run"),
				RebootMode:            rebootMode,
				Force:                 c.Bool("force"),
				Drain:                 c.Bool("drain"),
				DrainTimeout:          c.Duration("drain-timeout"),
				StabilizationDuration: c.Duration("stabilization-duration"),
				DeleteIfEvictionFails: c.Bool("delete-if-eviction-fails"),
				Stage:                 c.Bool("stage"),
				StageLabels:           c.StringSlice("stage-label"),
				StageAnnotations:      c.StringSlice("stage-annotation"),
				StageTaints:           c.StringSlice("stage-taint"),
				MaxParallel:           maxParallel,
			})
			if errors.Is(err, topf.ErrDryRunChangesDetected) {
				return cli.Exit(err.Error(), 2)
			}

			return err
		},
	}
}

// rebootModes maps user-facing mode names to their protobuf values, using the
// RebootRequest_Mode enum consumed by the Reboot RPC of the
// LifecycleService-based upgrade flow.
// https://github.com/siderolabs/talos/blob/main/cmd/talosctl/pkg/talos/helpers/mode.go
var rebootModes = map[string]machine.RebootRequest_Mode{ //nolint:gochecknoglobals // read-only lookup table
	"default":    machine.RebootRequest_DEFAULT,
	"powercycle": machine.RebootRequest_POWERCYCLE,
}

func validRebootModes() []string {
	modes := slices.Collect(maps.Keys(rebootModes))
	slices.SortFunc(modes, func(a, b string) int {
		return cmp.Compare(int32(rebootModes[a]), int32(rebootModes[b]))
	})

	return modes
}

func parseRebootMode(mode string) (machine.RebootRequest_Mode, error) {
	val, ok := rebootModes[mode]
	if !ok {
		return 0, fmt.Errorf("invalid reboot mode %q, valid values: %s", mode, strings.Join(validRebootModes(), ", "))
	}

	return val, nil
}
