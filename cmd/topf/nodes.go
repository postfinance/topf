package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/postfinance/topf/internal/topf"
	"github.com/postfinance/topf/pkg/config"
	"github.com/siderolabs/talos/pkg/machinery/config/encoder"
	"github.com/urfave/cli/v3"
	"go.yaml.in/yaml/v4"
)

func newNodesCmd() *cli.Command {
	return &cli.Command{
		Name:   "nodes",
		Usage:  "list all nodes and their current state",
		Before: noPositionalArgs,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "output",
				Aliases: []string{"o"},
				Usage:   "output format (table, yaml)",
				Value:   "table",
			},
			&cli.StringFlag{
				Name:        "machineconfig-output",
				Aliases:     []string{"m"},
				Usage:       "write machineconfig in <dir>",
				Value:       "./output",
				DefaultText: "./output",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			t := MustGetRuntime(ctx)

			nodes, err := t.Nodes(ctx)
			if err != nil {
				return err
			}

			// Write machine configs if the flag is set
			if cmd.IsSet("machineconfig-output") {
				outputDir := cmd.String("machineconfig-output")
				if err := writeMachineConfigs(nodes, outputDir); err != nil {
					return fmt.Errorf("failed to write machine configs: %w", err)
				}
			}

			outputFormat := cmd.String("output")

			switch outputFormat {
			case "table":
				return renderNodesTable(nodes)
			case "yaml":
				return renderNodesYAML(nodes)
			default:
				return fmt.Errorf("unsupported output format: %s (supported: table, yaml)", outputFormat)
			}
		},
	}
}

func renderNodesTable(nodes []*topf.Node) error {
	elipsis := func(s string) string {
		if len(s) > 8 {
			return s[:8] + "..."
		}

		return s
	}

	tw := table.NewWriter()
	tw.SetOutputMirror(os.Stdout)
	tw.AppendHeader(table.Row{"Host", "IP", "Role", "Stage", "Ready", "Unmet Conditions", "Schematic", "Talos", "Error"})
	tw.SetColumnConfigs([]table.ColumnConfig{
		{Name: "Unmet Conditions", WidthMax: 30},
		{Name: "Error", WidthMax: 30},
	})

	for _, node := range nodes {
		host := node.Node.Host
		ip := node.Node.IP
		role := string(node.Node.Role)

		var ready string

		unmetConditions := ""

		schematic := elipsis(node.Schematic)

		talosVersion := node.TalosVersion
		errorMsg := ""

		stage := node.MachineStatus.Stage.String()
		if node.MachineStatus.Status.Ready {
			ready = "✓"
		} else {
			ready = "✗"
		}

		if len(node.MachineStatus.Status.UnmetConditions) > 0 {
			var conditions []string
			for _, cond := range node.MachineStatus.Status.UnmetConditions {
				conditions = append(conditions, cond.Name+": "+cond.Reason)
			}

			unmetConditions = strings.Join(conditions, "\n")
		}

		if node.Error != nil {
			errorMsg = node.Error.Error()
		}

		tw.AppendRow(table.Row{host, ip, role, stage, ready, unmetConditions, schematic, talosVersion, errorMsg})
	}

	tw.Render()

	return nil
}

func renderNodesYAML(nodes []*topf.Node) error {
	yamlBytes, err := yaml.Marshal(nodes)
	if err != nil {
		return fmt.Errorf("failed to marshal nodes to YAML: %w", err)
	}

	fmt.Println(string(yamlBytes))

	return nil
}

func writeMachineConfigs(nodes []*topf.Node, outputDir string) error {
	// Create the output directory if it doesn't exist
	if err := os.MkdirAll(outputDir, 0o750); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	for _, node := range nodes {
		if node.ConfigBundle == nil {
			fmt.Fprintf(os.Stderr, "Warning: skipping %s - no config bundle available\n", node.Node.Host)
			continue
		}

		// Determine which config to use based on the node's role
		var provider interface {
			EncodeBytes(...encoder.Option) ([]byte, error)
		}

		if node.Node.Role == config.RoleControlPlane {
			provider = node.ConfigBundle.ControlPlaneCfg
		} else {
			provider = node.ConfigBundle.WorkerCfg
		}

		// Encode the config to YAML
		configBytes, err := provider.EncodeBytes(
			encoder.WithComments(encoder.CommentsDisabled),
			encoder.WithOmitEmpty(true),
		)
		if err != nil {
			return fmt.Errorf("failed to encode config for %s: %w", node.Node.Host, err)
		}

		// Write to file named <hostname>.yaml
		outputPath := filepath.Join(outputDir, node.Node.Host+".yaml")
		if err := os.WriteFile(outputPath, configBytes, 0o600); err != nil {
			return fmt.Errorf("failed to write config for %s: %w", node.Node.Host, err)
		}

		fmt.Fprintf(os.Stdout, "Wrote machine config for %s to %s\n", node.Node.Host, outputPath)
	}

	return nil
}
