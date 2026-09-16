// Copyright 2026 PostFinance AG
// SPDX-License-Identifier: MIT

// Package main is the entrypoint for the Topf CLI
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/postfinance/topf/internal/topf"
	talosversion "github.com/siderolabs/talos/pkg/machinery/version"
	"github.com/urfave/cli/v3"
)

type ContextKey string

const (
	topfRuntimeCtxKey ContextKey = "topf"
)

const (
	defaultTextTrue  = "true"
	defaultTextFalse = "false"
)

var version = "dev"

func main() {
	app := &cli.Command{
		Name:                  "topf",
		Usage:                 "Talos Orchestrator by PostFinance",
		Description:           "Topf is a CLI for managing Talos clusters.",
		Version:               fmt.Sprintf("%s (Talos %s)", version, talosversion.Tag),
		EnableShellCompletion: true,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "topfconfig",
				Value:   "topf.yaml",
				Usage:   "the topf config file to use",
				Sources: cli.EnvVars("TOPFCONFIG"),
			},
			&cli.StringFlag{
				Name:    "nodes-filter",
				Value:   "",
				Usage:   "use a regex expression to select a subset of nodes to work upon",
				Sources: cli.EnvVars("TOPF_NODES_FILTER"),
			},
			&cli.StringFlag{
				Name:    "log-level",
				Value:   "info",
				Usage:   "set the logging level (debug, info, warn, error)",
				Sources: cli.EnvVars("LOG_LEVEL"),
			},
			&cli.BoolFlag{
				Name:    "json-log",
				Value:   false,
				Usage:   "emit json logs instead of human readable text",
				Sources: cli.EnvVars("TOPF_JSON_LOG"),
			},
			&cli.BoolFlag{
				Name:        "redact",
				Value:       true,
				Usage:       "redact sensitive values (secrets, private keys) from output",
				Sources:     cli.EnvVars("TOPF_REDACT"),
				DefaultText: defaultTextTrue,
			},
			&cli.BoolFlag{
				Name:        "confirm",
				Usage:       "confirm any changes before applying them",
				Value:       true,
				Sources:     cli.EnvVars("TOPF_CONFIRM"),
				DefaultText: defaultTextTrue,
			},
			&cli.BoolFlag{
				Name:        "submit-to-factory",
				Usage:       "submit schematics to the image factory API instead of computing IDs locally",
				Sources:     cli.EnvVars("TOPF_SUBMIT_TO_FACTORY"),
				DefaultText: defaultTextFalse,
			},
		},
		Before: func(ctx context.Context, c *cli.Command) (context.Context, error) {
			if isHelpOrCompletionInvocation() {
				return ctx, nil
			}

			logger, err := topf.NewLogger(c.String("log-level"), c.Bool("json-log"))
			if err != nil {
				return ctx, err
			}

			slog.SetDefault(logger)

			topf, err := topf.NewTopfRuntime(topf.RuntimeConfig{
				ConfigPath:       c.String("topfconfig"),
				NodesRegexFilter: c.String("nodes-filter"),
				Logger:           logger,
				Redact:           c.Bool("redact"),
				Confirm:          c.Bool("confirm"),
				SubmitToFactory:  c.Bool("submit-to-factory"),
				TopfVersion:      version,
			})
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
			newSchematicIDsCmd(),
			newRenderCmd(),
			newSecretsCmd(),
			newKubeconfigCmd(),
			newTalosconfigCmd(),
		},
	}

	for _, c := range app.Commands {
		c.ShellComplete = completeWithRootFlags
	}

	if err := app.Run(context.Background(), os.Args); err != nil {
		slog.Error("failed to run command", "error", err)
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

// noPositionalArgs is a Before hook that rejects any positional arguments.
// Use this for commands that only accept flags.
func noPositionalArgs(ctx context.Context, c *cli.Command) (context.Context, error) {
	if isHelpOrCompletionInvocation() {
		return ctx, nil
	}

	if c.Args().Len() > 0 {
		return ctx, fmt.Errorf("unexpected argument(s): %v. Did you mean to use flags? (e.g., --flag=value instead of flag=value)", c.Args().Slice())
	}

	return ctx, nil
}

// isHelpOrCompletionInvocation reports whether topf was invoked to show help
// or generate shell completions, where the runtime is not required.
func isHelpOrCompletionInvocation() bool {
	if len(os.Args) == 1 || os.Args[1] == "completion" {
		return true
	}

	for _, arg := range os.Args[1:] {
		if arg == "--generate-shell-completion" {
			return true
		}
	}

	return false
}

// completeWithRootFlags additionally suggests root flags on subcommands,
// which the default completion does not do.
func completeWithRootFlags(ctx context.Context, c *cli.Command) {
	cli.DefaultCompleteWithFlags(ctx, c)

	if c.Root() == c {
		return
	}

	args := os.Args
	if len(args) > 2 && strings.HasPrefix(args[len(args)-2], "-") {
		completeRootFlags(ctx, c.Root())
	}
}

// completeRootFlags suggests root flags, except help and version which the
// subcommand pass already suggests.
func completeRootFlags(ctx context.Context, root *cli.Command) {
	for _, f := range root.Flags {
		name := strings.TrimSpace(f.Names()[0])
		if name == "help" || name == "version" {
			continue
		}

		cli.DefaultCompleteWithFlags(ctx, &cli.Command{Flags: []cli.Flag{f}, Writer: root.Writer})
	}
}
