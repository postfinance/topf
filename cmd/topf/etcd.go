// Copyright 2026 PostFinance AG
// SPDX-License-Identifier: MIT

package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/postfinance/topf/internal/cmd/etcdbackup"
	"github.com/urfave/cli/v3"
	"go.yaml.in/yaml/v4"
)

const (
	outputFlag  = "output"
	outputTable = "table"
	outputYAML  = "yaml"
)

func newEtcdCmd() *cli.Command {
	return &cli.Command{
		Name:  "etcd",
		Usage: "manage the etcd cluster",
		Commands: []*cli.Command{
			newEtcdBackupCmd(),
		},
	}
}

func newEtcdBackupCmd() *cli.Command {
	return &cli.Command{
		Name:  "backup",
		Usage: "manage etcd snapshot backups",
		Commands: []*cli.Command{
			newEtcdBackupCreateCmd(),
			newEtcdBackupListCmd(),
		},
	}
}

// backupStore returns the backup store described by the backups block in topf.yaml
func backupStore(ctx context.Context) (etcdbackup.Store, error) {
	t := MustGetRuntime(ctx)
	cfg := t.Config()

	if cfg.Backups == nil || cfg.Backups.S3 == nil {
		return nil, errors.New("etcd backups require 'backups.s3' to be configured in topf.yaml")
	}

	if cfg.Backups.S3.SecretAccessKey != "" {
		t.AddSecretsToMask([]string{cfg.Backups.S3.SecretAccessKey})
	}

	return etcdbackup.NewS3Store(cfg.Backups.S3, cfg.ClusterName)
}

func newEtcdBackupCreateCmd() *cli.Command {
	return &cli.Command{
		Name:        "create",
		Usage:       "create a new etcd snapshot backup",
		Description: `Streams a consistent snapshot of the etcd database from the first reachable control-plane node, verifies its integrity offline and uploads it together with a manifest to the S3 storage configured.`,
		Before:      noPositionalArgs,
		Action: func(ctx context.Context, _ *cli.Command) error {
			t := MustGetRuntime(ctx)

			store, err := backupStore(ctx)
			if err != nil {
				return err
			}

			manifest, err := etcdbackup.Create(ctx, t, store)
			if err != nil {
				return err
			}

			t.Logger().Info("etcd backup created",
				"name", manifest.Name,
				"node", manifest.Node,
				"location", store.Location(),
				"size", manifest.Size,
				"etcdRevision", manifest.EtcdRevision,
				"etcdTotalKeys", manifest.EtcdTotalKeys,
			)

			return nil
		},
	}
}

func newEtcdBackupListCmd() *cli.Command {
	return &cli.Command{
		Name:   "list",
		Usage:  "list etcd snapshot backups",
		Before: noPositionalArgs,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    outputFlag,
				Aliases: []string{"o"},
				Usage:   "output format (table, yaml)",
				Value:   outputTable,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			store, err := backupStore(ctx)
			if err != nil {
				return err
			}

			manifests, err := store.List(ctx)
			if err != nil {
				return err
			}

			switch outputFormat := cmd.String(outputFlag); outputFormat {
			case outputTable:
				return renderBackupsTable(manifests)
			case outputYAML:
				return renderBackupsYAML(manifests)
			default:
				return fmt.Errorf("unsupported output format: %s (supported: table, yaml)", outputFormat)
			}
		},
	}
}

func renderBackupsTable(manifests []etcdbackup.Manifest) error {
	tw := table.NewWriter()
	tw.SetOutputMirror(os.Stdout)
	tw.AppendHeader(table.Row{"Name", "Created", "Node", "Talos", "Size", "Revision", "Keys"})

	for _, m := range manifests {
		tw.AppendRow(table.Row{
			m.Name,
			m.CreatedAt.UTC().Format("2006-01-02 15:04:05 UTC"),
			m.Node,
			m.TalosVersion,
			formatSize(m.Size),
			m.EtcdRevision,
			m.EtcdTotalKeys,
		})
	}

	tw.Render()

	return nil
}

func renderBackupsYAML(manifests []etcdbackup.Manifest) error {
	yamlBytes, err := yaml.Marshal(manifests)
	if err != nil {
		return fmt.Errorf("failed to marshal backups to yaml: %w", err)
	}

	fmt.Print(string(yamlBytes))

	return nil
}

// formatSize renders a byte count in a human readable binary unit
func formatSize(size int64) string {
	const unit = 1024

	if size < unit {
		return fmt.Sprintf("%d B", size)
	}

	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	return fmt.Sprintf("%.1f %ciB", float64(size)/float64(div), "KMGTPE"[exp])
}
