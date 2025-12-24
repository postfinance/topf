// Package talosconfig contains the logic to generate talosconfig file
package talosconfig

import (
	"fmt"

	"github.com/postfinance/topf/internal/topf"
	"github.com/siderolabs/talos/pkg/machinery/config/bundle"
	"github.com/siderolabs/talos/pkg/machinery/config/generate"
)

// Generate returns a talosconfig
func Generate(t topf.Topf) ([]byte, error) {
	secretsBundle, err := t.Secrets()
	if err != nil {
		return nil, fmt.Errorf("failed to load secrets bundle: %w", err)
	}

	configBundleOpts := []bundle.Option{
		bundle.WithVerbose(false), // prevent printing "generating PKI and tokens"
		bundle.WithInputOptions(
			&bundle.InputOptions{
				ClusterName: t.Config().ClusterName,
				GenOptions: []generate.Option{
					generate.WithSecretsBundle(secretsBundle),
				},
			},
		),
	}

	configBundle, err := bundle.NewBundle(configBundleOpts...)
	if err != nil {
		return nil, err
	}

	// setting nodes/endpoints directly is only safe in a single node cluster because we can't
	// anticipate what commands the user is going to make. even though some commands work
	// when multiple endpoints are specified, some may throw errors like "no request forwarding"
	if len(t.Config().Nodes) == 1 {
		endpoints := []string{t.Config().Nodes[0].Endpoint()}
		configBundle.TalosCfg.Contexts[t.Config().ClusterName].Nodes = endpoints
		configBundle.TalosCfg.Contexts[t.Config().ClusterName].Endpoints = endpoints
	}

	return configBundle.TalosCfg.Bytes()
}
