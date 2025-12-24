package main

import (
	"context"
	"fmt"

	"github.com/postfinance/topf/internal/cmd/kubeconfig"
	"github.com/urfave/cli/v3"
	"k8s.io/client-go/tools/clientcmd"
)

func newKubeconfigCmd() *cli.Command {
	return &cli.Command{
		Name:  "kubeconfig",
		Usage: "generate a temporary admin kubeconfig",
		Action: func(ctx context.Context, _ *cli.Command) error {
			t := MustGetRuntime(ctx)

			kubeconfigStruct, err := kubeconfig.Generate(t)
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
