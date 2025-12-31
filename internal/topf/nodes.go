package topf

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/cosi-project/runtime/pkg/safe"
	"github.com/postfinance/topf/pkg/config"
	talosconfig "github.com/siderolabs/talos/pkg/machinery/config"
	"github.com/siderolabs/talos/pkg/machinery/config/bundle"
	"github.com/siderolabs/talos/pkg/machinery/config/generate"
	"github.com/siderolabs/talos/pkg/machinery/resources/runtime"
)

// Nodes gathers information about each configured node
// Errors during gathering information for individual nodes are recorded in the Node.Error field
//
//nolint:funlen,gocognit // TODO: refactor and split into parts
func (t *topf) Nodes(ctx context.Context) ([]*Node, error) {
	cfg := t.Config()

	nodes := make([]*Node, 0, len(cfg.Nodes))

	for _, node := range cfg.Nodes {
		nodes = append(nodes, &Node{Node: &node, t: t})
	}

	var wg sync.WaitGroup

	for _, node := range nodes {
		wg.Add(1)

		go func(node *Node) {
			defer wg.Done()

			logger := t.Logger().With(node.Attrs())

			logger.Debug("collecting data")

			// Create client
			nodeClient, err := node.Client(ctx)
			if err != nil {
				node.Error = fmt.Errorf("failed to create client: %w", err)
				return
			}

			// Get machine status
			machineStatus, err := safe.StateGetResource(ctx, nodeClient.COSI, runtime.NewMachineStatus())
			if err != nil {
				node.Error = fmt.Errorf("unable to get machine status: %w", err)
				return
			}

			node.MachineStatus = *machineStatus.TypedSpec()

			// Collect schematic
			extensions, err := safe.StateListAll[*runtime.ExtensionStatus](ctx, nodeClient.COSI)
			if err != nil {
				node.Error = fmt.Errorf("couldn't list extensions: %w", err)
				return
			}

			// it's possible that the schematic extension is not present
			// in which case we have to assume the default one
			node.Schematic = DefaultSchematic

			for extension := range extensions.All() {
				if extension.TypedSpec().Metadata.Name == "schematic" {
					schematic := extension.TypedSpec()
					node.Schematic = schematic.Metadata.Version
				}
			}

			// Collect Talos version vis COSI client because it is also avaialable
			// on maintenance mode
			versions, err := safe.StateListAll[*runtime.Version](ctx, nodeClient.COSI)
			if err != nil {
				node.Error = fmt.Errorf("couldn't list versions: %w", err)
				return
			}

			for v := range versions.All() {
				if v.Metadata().Type() == runtime.VersionType {
					node.TalosVersion = strings.TrimPrefix(v.TypedSpec().Version, "v")
				}
			}

			// Generating Config bundle
			logger.Debug("generating configuration bundle")

			patchContext := &config.PatchContext{
				ClusterName:       t.Config().ClusterName,
				ClusterEndpoint:   t.Config().ClusterEndpoint.String(),
				KubernetesVersion: t.Config().KubernetesVersion,
				Node:              node.Node,
				Data:              t.Config().Data,
			}

			patches, err := patchContext.Load()
			if err != nil {
				node.Error = fmt.Errorf("couldn't load patch: %w", err)
				return
			}

			secretsBundle, err := t.Secrets()
			if err != nil {
				node.Error = fmt.Errorf("failed to get secrets bundle: %w", err)
				return
			}

			versionContract, err := talosconfig.ParseContractFromVersion(node.TalosVersion)
			if err != nil {
				node.Error = err
				return
			}

			configBundleOpts := []bundle.Option{
				bundle.WithInputOptions(
					&bundle.InputOptions{
						ClusterName: t.Config().ClusterName,
						Endpoint:    t.Config().ClusterEndpoint.String(),
						KubeVersion: strings.TrimPrefix(t.Config().KubernetesVersion, "v"),
						GenOptions: []generate.Option{
							generate.WithSecretsBundle(secretsBundle),
							generate.WithVersionContract(versionContract),
						},
					},
				),
				bundle.WithVerbose(false), // prevent printing "generating PKI and tokens"
			}

			switch node.Node.Role {
			case config.RoleControlPlane:
				configBundleOpts = append(configBundleOpts, bundle.WithPatchControlPlane(patches))
			case config.RoleWorker:
				configBundleOpts = append(configBundleOpts, bundle.WithPatchWorker(patches))
			}

			configBundle, err := bundle.NewBundle(configBundleOpts...)
			if err != nil {
				node.Error = err
				return
			}

			node.ConfigBundle = configBundle

			logger.Debug("node information collected")
		}(node)
	}

	wg.Wait()

	return nodes, nil
}
