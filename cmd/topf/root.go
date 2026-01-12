// Package main is the entrypoint for the Topf CLI
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/postfinance/topf/internal/topf"
	"github.com/urfave/cli/v3"
)

type ContextKey string

const (
	topfRuntimeCtxKey ContextKey = "topf"
)

var version = "dev"

func main() {
	app := &cli.Command{
		Name:        "topf",
		Usage:       "Talos Orchestrator by PostFinance",
		Description: "Topf is a CLI for managing Talos clusters.",
		Version:     version,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "topfconfig",
				Value:   "topf.yaml",
				Usage:   "the topf config file to use",
				Sources: cli.EnvVars("TOPFCONFIG"),
			},
			&cli.StringFlag{
				Name:    "config-dir",
				Value:   ".",
				Usage:   "directory from which to read the configuration (patches)",
				Sources: cli.EnvVars("TOPF_CONFIG_DIR"),
			},
			&cli.StringFlag{
				Name:    "nodes",
				Value:   "",
				Usage:   "use a regex expression to select a subset of nodes to work upon",
				Sources: cli.EnvVars("TOPF_NODES"),
			},
			&cli.StringFlag{
				Name:    "log-level",
				Value:   "info",
				Usage:   "set the logging level (debug, info, warn, error)",
				Sources: cli.EnvVars("LOG_LEVEL"),
			},
		},
		Before: func(ctx context.Context, c *cli.Command) (context.Context, error) {
			// Validate config-dir exists
			configDir := c.String("config-dir")
			if stat, err := os.Stat(configDir); err != nil {
				if os.IsNotExist(err) {
					return ctx, fmt.Errorf("config directory does not exist: %s", configDir)
				}

				return ctx, fmt.Errorf("failed to access config directory: %w", err)
			} else if !stat.IsDir() {
				return ctx, fmt.Errorf("config path is not a directory: %s", configDir)
			}

			// passing down the Topf runtime to all commands via context
			topf, err := topf.NewTopfRuntime(
				c.String("topfconfig"),
				configDir,
				c.String("nodes"),
				c.String("log-level"),
			)
			if err != nil {
				return ctx, err
			}

			return context.WithValue(ctx, topfRuntimeCtxKey, topf), nil
		},
		Commands: []*cli.Command{
			newApplyCmd(),
			newUpgradeCmd(),
			newResetCmd(),
			newClusterInfoCmd(),
			newNodesCmd(),
			newSecretsCmd(),
			newKubeconfigCmd(),
			newTalosconfigCmd(),
		},
	}

	if err := app.Run(context.Background(), os.Args); err != nil {
		slog.Error("error", "error", err.Error())
		os.Exit(1)
	}
}

// MustGetRuntime returns the topf runtime from the context or panics
func MustGetRuntime(ctx context.Context) topf.Topf {
	t, ok := ctx.Value(topfRuntimeCtxKey).(topf.Topf)
	if !ok {
		panic("TopfRuntimeCtxKey not found in context")
	}

	return t
}

func confirmFlag() *cli.BoolFlag {
	return &cli.BoolFlag{
		Name:  "confirm",
		Usage: "confirm any changes before applying them",
		Value: true,
	}
}

// noPositionalArgs is a Before hook that rejects any positional arguments.
// Use this for commands that only accept flags.
func noPositionalArgs(ctx context.Context, c *cli.Command) (context.Context, error) {
	if c.Args().Len() > 0 {
		return ctx, fmt.Errorf("unexpected argument(s): %v. Did you mean to use flags? (e.g., --flag=value instead of flag=value)", c.Args().Slice())
	}

	return ctx, nil
}
