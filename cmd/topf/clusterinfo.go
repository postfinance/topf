package main

import (
	"context"
	"fmt"

	"github.com/postfinance/topf/internal/cmd/clusterinfo"
	"github.com/urfave/cli/v3"
	"go.yaml.in/yaml/v4"
)

func newClusterInfoCmd() *cli.Command {
	return &cli.Command{
		Name:  "clusterinfo",
		Usage: "output non-sensitive cluster information",
		Action: func(ctx context.Context, _ *cli.Command) error {
			t := MustGetRuntime(ctx)

			clusterInfo, err := clusterinfo.Get(t)
			if err != nil {
				return err
			}

			yamlBytes, err := yaml.Marshal(clusterInfo)
			if err != nil {
				return err
			}

			fmt.Println(string(yamlBytes))

			return nil
		},
	}
}
