// Copyright 2026 PostFinance AG
// SPDX-License-Identifier: MIT

package topf

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/cosi-project/runtime/pkg/resource"
	"github.com/cosi-project/runtime/pkg/safe"
	"github.com/postfinance/topf/pkg/config"
	"github.com/siderolabs/talos/pkg/machinery/cel"
	"github.com/siderolabs/talos/pkg/machinery/cel/celenv"
	talosconfig "github.com/siderolabs/talos/pkg/machinery/config"
	"github.com/siderolabs/talos/pkg/machinery/config/bundle"
	"github.com/siderolabs/talos/pkg/machinery/config/configpatcher"
	"github.com/siderolabs/talos/pkg/machinery/config/generate"
	runtimetypes "github.com/siderolabs/talos/pkg/machinery/config/types/runtime"
	configresource "github.com/siderolabs/talos/pkg/machinery/resources/config"
	"github.com/siderolabs/talos/pkg/machinery/resources/runtime"
	"go.yaml.in/yaml/v4"
)

const defaultInstallDisk string = "/dev/sda" // as defined in talos/gen/config.go

func installerImagePatch(image string, versionContract *talosconfig.VersionContract) (configpatcher.Patch, error) {
	var patchBytes []byte

	var err error

	if versionContract.UnattendedInstallConfig() {
		// we reuse the upstream default: https://github.com/siderolabs/talos/blob/v1.14.0/pkg/machinery/config/generate/init.go#L299-L314
		unattended := runtimetypes.NewUnattendedInstallConfigV1Alpha1()
		unattended.Installer.Image = image
		unattended.ProvisioningSpec.Wipe = new(false)

		expr, exprErr := cel.ParseBooleanExpression(fmt.Sprintf("disk.dev_path == %q", defaultInstallDisk), celenv.DiskLocator())
		if exprErr != nil {
			return nil, fmt.Errorf("failed to build install disk selector: %w", exprErr)
		}

		unattended.ProvisioningSpec.DiskSelector.Match = expr
		patchBytes, err = yaml.Marshal(unattended)
	} else {
		patchBytes, err = yaml.Marshal(map[string]any{
			"machine": map[string]any{
				"install": map[string]any{
					"image": image,
				},
			},
		})
	}

	if err != nil {
		return nil, err
	}

	return configpatcher.LoadPatch(patchBytes)
}

// collectNodeInfo queries a live node via COSI to populate MachineStatus, Schematic, and TalosVersion.
func (n *Node) collectNodeInfo(ctx context.Context) error {
	nodeClient, err := n.Client(ctx)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	machineStatus, err := safe.StateGetResource(ctx, nodeClient.COSI, runtime.NewMachineStatus())
	if err != nil {
		return fmt.Errorf("unable to get machine status: %w", err)
	}

	n.MachineStatus = *machineStatus.TypedSpec()

	// Fetch the current machine config from the node and add its
	// sensitive values to the redaction pool. During a key rotation
	// the old certs still live on the node; without this they would
	// leak through the masked writer. Only available outside maintenance mode.
	if n.MachineStatus.Stage != runtime.MachineStageMaintenance {
		machineConfig, err := safe.StateGet[*configresource.MachineConfig](
			ctx, nodeClient.COSI,
			resource.NewMetadata(configresource.NamespaceName, configresource.MachineConfigType, configresource.ActiveID, resource.VersionUndefined),
		)
		if err != nil {
			return fmt.Errorf("couldn't get machine config: %w", err)
		}

		if machineConfig == nil {
			return errors.New("retrieved a 'nil' machine config")
		}

		n.t.AddSecretsToMask(collectCurrentConfigSecrets(machineConfig.Provider()))
	}

	extensions, err := safe.StateListAll[*runtime.ExtensionStatus](ctx, nodeClient.COSI)
	if err != nil {
		return fmt.Errorf("couldn't list extensions: %w", err)
	}

	// it's possible that the schematic extension is not present
	// in which case we have to assume the default one
	n.runningSchematic = DefaultSchematic

	for extension := range extensions.All() {
		if extension.TypedSpec().Metadata.Name == "schematic" {
			n.runningSchematic = extension.TypedSpec().Metadata.Version
		}
	}

	// Collect Talos version via COSI client because it is also available on maintenance mode
	versions, err := safe.StateListAll[*runtime.Version](ctx, nodeClient.COSI)
	if err != nil {
		return fmt.Errorf("couldn't list versions: %w", err)
	}

	for v := range versions.All() {
		if v.Metadata().Type() == runtime.VersionType {
			n.runningVersion = strings.TrimPrefix(v.TypedSpec().Version, "v")
		}
	}

	n.Node.RuntimeData = config.RuntimeData{
		TalosVersion: n.runningVersion,
		SchematicID:  n.runningSchematic,
		Stage:        n.MachineStatus.Stage.String(),
	}

	return nil
}

func (t *topf) generateNodeConfig(ctx context.Context, node *Node) error {
	t.Logger().With(node.Attrs()).Debug("generating configuration bundle")

	cfg := t.Config()

	patchContext := &config.PatchContext{
		ClusterName:       cfg.ClusterName,
		ClusterEndpoint:   cfg.ClusterEndpoint.String(),
		KubernetesVersion: cfg.KubernetesVersion,
		TalosVersion:      cfg.TalosVersion,
		SchematicID:       cmp.Or(node.Node.SchematicID, cfg.SchematicID, DefaultSchematic),
		Node:              node.Node,
		Data:              cfg.Data,
		PatchesDir:        t.patchesDir,
		DecryptCache:      t.decryptCache,
	}

	resolvedSchematic, err := t.resolver.Resolve(ctx, cmp.Or(node.Node.Factory, cfg.Factory, DefaultFactory), patchContext.SchematicID, patchContext)
	if err != nil {
		return fmt.Errorf("failed to resolve schematic ID: %w", err)
	}

	node.resolvedSchematic = resolvedSchematic
	patchContext.SchematicID = resolvedSchematic

	patches, patchSecrets, err := patchContext.Load()
	if err != nil {
		return fmt.Errorf("couldn't load patches: %w", err)
	}

	versionContract, err := talosconfig.ParseContractFromVersion(node.TalosVersion())
	if err != nil {
		return err
	}

	installPatch, err := installerImagePatch(node.InstallerImage(), versionContract)
	if err != nil {
		return fmt.Errorf("failed to build installer image patch: %w", err)
	}

	patches = append([]configpatcher.Patch{installPatch}, patches...)

	t.AddSecretsToMask(patchSecrets)

	secretsBundle, err := t.Secrets()
	if err != nil {
		return fmt.Errorf("failed to get secrets bundle: %w", err)
	}

	configBundleOpts := []bundle.Option{
		bundle.WithInputOptions(
			&bundle.InputOptions{
				ClusterName: t.Config().ClusterName,
				Endpoint:    t.Config().ClusterEndpoint.String(),
				KubeVersion: node.KubernetesVersion(),
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

// filterNodes returns topf.Node wrappers for all configured nodes matching
// the nodes-filter regex.
func (t *topf) filterNodes() []*Node {
	cfg := t.Config()

	nodes := make([]*Node, 0, len(cfg.Nodes))

	for i := range cfg.Nodes {
		nodeCfg := &cfg.Nodes[i]

		if !t.nodesFilter.MatchString(nodeCfg.Host) {
			continue
		}

		nodes = append(nodes, &Node{Node: nodeCfg, t: t})
	}

	return nodes
}

// FilteredNodes gathers information about each configured node matching the
// nodes-filter regex. Per-node errors are recorded in Node.Error.
func (t *topf) FilteredNodes(ctx context.Context) ([]*Node, error) {
	nodes := t.filterNodes()

	var wg sync.WaitGroup

	for _, node := range nodes {
		wg.Add(1)

		go func(node *Node) {
			defer wg.Done()

			t.Logger().With(node.Attrs()).Debug("collecting data")

			if err := node.collectNodeInfo(ctx); err != nil {
				node.Error = err
				return
			}

			if err := t.generateNodeConfig(ctx, node); err != nil {
				node.Error = err
			}
		}(node)
	}

	wg.Wait()

	return nodes, nil
}
