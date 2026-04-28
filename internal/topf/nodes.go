// Copyright 2026 PostFinance AG
// SPDX-License-Identifier: MIT

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
	"github.com/siderolabs/talos/pkg/machinery/config/configpatcher"
	"github.com/siderolabs/talos/pkg/machinery/config/generate"
	"github.com/siderolabs/talos/pkg/machinery/resources/runtime"
	"go.yaml.in/yaml/v4"
)

func installerImagePatch(image string) (configpatcher.Patch, error) {
	patchBytes, err := yaml.Marshal(map[string]any{
		"machine": map[string]any{
			"install": map[string]any{
				"image": image,
			},
		},
	})
	if err != nil {
		return nil, err
	}

	return configpatcher.LoadPatch(patchBytes)
}

// collectNodeInfo queries a live node via COSI to populate MachineStatus, Schematic, and TalosVersion.
func (t *topf) collectNodeInfo(ctx context.Context, node *Node) error {
	nodeClient, err := node.Client(ctx)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	machineStatus, err := safe.StateGetResource(ctx, nodeClient.COSI, runtime.NewMachineStatus())
	if err != nil {
		return fmt.Errorf("unable to get machine status: %w", err)
	}

	node.MachineStatus = *machineStatus.TypedSpec()

	extensions, err := safe.StateListAll[*runtime.ExtensionStatus](ctx, nodeClient.COSI)
	if err != nil {
		return fmt.Errorf("couldn't list extensions: %w", err)
	}

	// it's possible that the schematic extension is not present
	// in which case we have to assume the default one
	node.Schematic = DefaultSchematic

	for extension := range extensions.All() {
		if extension.TypedSpec().Metadata.Name == "schematic" {
			node.Schematic = extension.TypedSpec().Metadata.Version
		}
	}

	// Collect Talos version via COSI client because it is also available on maintenance mode
	versions, err := safe.StateListAll[*runtime.Version](ctx, nodeClient.COSI)
	if err != nil {
		return fmt.Errorf("couldn't list versions: %w", err)
	}

	for v := range versions.All() {
		if v.Metadata().Type() == runtime.VersionType {
			node.TalosVersion = strings.TrimPrefix(v.TypedSpec().Version, "v")
		}
	}

	return nil
}

// generateNodeConfig loads patches and builds the machine config bundle for a node.
// node.TalosVersion must be set before calling this.
func (t *topf) generateNodeConfig(node *Node) error {
	t.Logger().With(node.Attrs()).Debug("generating configuration bundle")

	patchContext := &config.PatchContext{
		ClusterName:       t.Config().ClusterName,
		ClusterEndpoint:   t.Config().ClusterEndpoint.String(),
		KubernetesVersion: t.Config().KubernetesVersion,
		TalosVersion:      t.Config().TalosVersion,
		SchematicID:       t.Config().SchematicID,
		Node:              node.Node,
		Data:              t.Config().Data,
		ConfigDir:         t.configDir,
	}

	patches, patchSecrets, err := patchContext.Load()
	if err != nil {
		return fmt.Errorf("couldn't load patches: %w", err)
	}

	installPatch, err := installerImagePatch(t.Config().InstallerImage())
	if err != nil {
		return fmt.Errorf("failed to build installer image patch: %w", err)
	}

	patches = append([]configpatcher.Patch{installPatch}, patches...)

	t.addSecrets(patchSecrets)

	secretsBundle, err := t.Secrets()
	if err != nil {
		return fmt.Errorf("failed to get secrets bundle: %w", err)
	}

	versionContract, err := talosconfig.ParseContractFromVersion(node.TalosVersion)
	if err != nil {
		return err
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
		return err
	}

	node.ConfigBundle = configBundle

	return nil
}

// Nodes gathers information about each configured node.
// Errors during gathering information for individual nodes are recorded in the Node.Error field.
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

			t.Logger().With(node.Attrs()).Debug("collecting data")

			if err := t.collectNodeInfo(ctx, node); err != nil {
				node.Error = err
				return
			}

			if err := t.generateNodeConfig(node); err != nil {
				node.Error = err
			}
		}(node)
	}

	wg.Wait()

	return nodes, nil
}
