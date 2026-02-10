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
		Usage:  "generate a temporary admin kubeconfig",
		Before: noPositionalArgs,
		Flags: []cli.Flag{
			&cli.DurationFlag{
				Name:  "validity",
				Usage: "validity duration of the client certificate",
				Value: 12 * time.Hour,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			t := MustGetRuntime(ctx)

			kubeconfigStruct, err := kubeconfig.Generate(t, cmd.Duration("validity"))
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
