package main

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"
	"go.yaml.in/yaml/v4"
)

func newSecretsCmd() *cli.Command {
	return &cli.Command{
		Name:  "secrets",
		Usage: "get or generate secrets.yaml for cluster",
		Action: func(ctx context.Context, _ *cli.Command) error {
			t := MustGetRuntime(ctx)

			secretsBundle, err := t.Secrets()
			if err != nil {
				return fmt.Errorf("failed to load secrets bundle: %w", err)
			}

			yamlBytes, err := yaml.Marshal(secretsBundle)
			if err != nil {
				return err
			}

			fmt.Println(string(yamlBytes))

			return nil
		},
	}
}
