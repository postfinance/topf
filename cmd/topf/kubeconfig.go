// Copyright 2026 PostFinance AG
// SPDX-License-Identifier: MIT

package main

import (
	"context"
	"fmt"
	"time"

	"github.com/postfinance/topf/internal/cmd/kubeconfig"
	"github.com/urfave/cli/v3"
	"k8s.io/client-go/tools/clientcmd"
)

func newKubeconfigCmd() *cli.Command {
	return &cli.Command{
		Name:   "kubeconfig",
		Usage:  "generate a temporary kubeconfig",
		Before: noPositionalArgs,
		Flags: []cli.Flag{
			&cli.DurationFlag{
				Name:  "validity",
				Usage: "validity duration of the client certificate",
				Value: 12 * time.Hour,
			},
			&cli.StringFlag{
				Name:  "user",
				Usage: "user (CN) of the client certificate",
				Value: kubeconfig.DefaultUser,
			},
			&cli.StringSliceFlag{
				Name:  "groups",
				Usage: "groups (O) of the client certificate (system:masters grants full admin)",
				Value: []string{kubeconfig.DefaultGroup},
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			t := MustGetRuntime(ctx)

			kubeconfigStruct, err := kubeconfig.Generate(t, cmd.String("user"), cmd.StringSlice("groups"), cmd.Duration("validity"))
			if err != nil {
				return fmt.Errorf("couldn't generate kubeconfig: %w", err)
			}

			kubeconfigBytes, err := clientcmd.Write(*kubeconfigStruct)
			if err != nil {
				return fmt.Errorf("failed to marshal kubeconfig: %w", err)
			}

			fmt.Println(string(kubeconfigBytes))

			return nil
		},
	}
}
