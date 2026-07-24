// Copyright 2026 PostFinance AG
// SPDX-License-Identifier: MIT

// Package upgrade contains the logic to upgrade Talos OS on cluster nodes
package upgrade

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"regexp"
	"slices"
	"time"

	"github.com/postfinance/topf/internal/interactive"
	"github.com/postfinance/topf/internal/nodepool"
	"github.com/postfinance/topf/internal/topf"
	taloskubeclient "github.com/siderolabs/talos/cmd/talosctl/pkg/talos/kubeclient"
	talosnodedrain "github.com/siderolabs/talos/cmd/talosctl/pkg/talos/nodedrain"
	"github.com/siderolabs/talos/pkg/machinery/api/common"
	"github.com/siderolabs/talos/pkg/machinery/api/machine"
	"github.com/siderolabs/talos/pkg/machinery/client"
	"github.com/siderolabs/talos/pkg/machinery/resources/runtime"
	"github.com/siderolabs/talos/pkg/reporter"
	"google.golang.org/grpc/codes"
	"k8s.io/client-go/kubernetes"
)

// Options contains the options for the upgrade execution
type Options struct {
	// Only show what upgrades would be performed without actually upgrading
	DryRun bool

	// RebootMode controls how the node is rebooted after the upgrade artifacts
	// are installed. It maps to machine.RebootRequest_Mode (used by the
	// Reboot RPC).
	RebootMode machine.RebootRequest_Mode

	// Drain controls whether the Kubernetes node is cordoned and drained
	// before the reboot and uncordoned after the node becomes Ready again.
	Drain bool

	// DrainTimeout is the maximum time to wait for pod evictions to complete.
	DrainTimeout time.Duration

	// MaxParallel controls how many worker nodes are upgraded concurrently.
	// Control-plane nodes are always upgraded one at a time.
	MaxParallel nodepool.MaxParallel
}

// Execute performs the Talos OS upgrades for all nodes in the cluster
func Execute(ctx context.Context, t topf.Topf, opts Options) error {
	logger := t.Logger().With("command", "upgrade")

	// Gather node information
	nodes, err := t.FilteredNodes(ctx)
	if err != nil {
		return err
	}

	if err := preChecks(logger, nodes); err != nil {
		return err
	}

	// Plan phase: determine which nodes require an upgrade. Interactive
	// confirmations happen here, sequentially, before any concurrent work.
	worklist, upgradeRequired, err := plan(t, logger, nodes, opts)
	if err != nil {
		return err
	}

	if opts.DryRun {
		if upgradeRequired {
			return topf.ErrDryRunChangesDetected
		}

		return nil
	}

	controlPlane, workers := nodepool.PartitionByRole(worklist)

	// Control-plane nodes are upgraded strictly one at a time to preserve etcd
	// quorum; this also satisfies "control-plane upgrades cannot be scheduled
	// concurrently".
	for _, node := range controlPlane {
		if err := upgradeNode(ctx, t, node, opts, logger.With(node.Attrs())); err != nil {
			return err
		}
	}

	// Worker nodes are upgraded using a rolling pool: up to n upgrades are kept
	// in flight, and as soon as one finishes the next node is started.
	if len(workers) > 0 {
		concurrency := opts.MaxParallel.Resolve(len(nodes))
		logger.Info("upgrading worker nodes", "count", len(workers), "concurrency", concurrency)

		return nodepool.RunConcurrent(ctx, workers, concurrency,
			func(ctx context.Context, node *topf.Node, logger *slog.Logger) error {
				return upgradeNode(ctx, t, node, opts, logger)
			}, logger)
	}

	return nil
}

// preChecks verifies that every node is reachable and running before any
// upgrade is attempted, reporting all problems at once.
func preChecks(logger *slog.Logger, nodes []*topf.Node) error {
	abort := false

	for _, node := range nodes {
		logger := logger.With(node.Attrs())

		if node.Error != nil {
			logger.Error("node pre-checks", "error", node.Error)

			abort = true

			continue
		}

		if !slices.Contains([]runtime.MachineStage{runtime.MachineStageRunning}, node.MachineStatus.Stage) {
			logger.Error("node must be 'running' for upgrade", "stage", node.MachineStatus.Stage.String())

			abort = true

			continue
		}
	}

	if abort {
		return errors.New("aborting due to errors with some nodes")
	}

	return nil
}

// plan determines which nodes require an upgrade, performing interactive
// confirmations sequentially before any concurrent work.
func plan(t topf.Topf, logger *slog.Logger, nodes []*topf.Node, opts Options) (worklist []*topf.Node, upgradeRequired bool, err error) {
	for _, node := range nodes {
		logger := logger.With(node.Attrs())

		installerImage := node.ConfigProvider().Machine().Install().Image()

		schematic, talosVersion, err := extractSchematicAndVersion(installerImage)
		if err != nil {
			return nil, false, fmt.Errorf("couldn't extract schematic and version from installer image '%s': %w", installerImage, err)
		}

		nodeNeedsUpgrade := node.RunningVersion() != talosVersion || node.RunningSchematic() != schematic
		if !nodeNeedsUpgrade {
			logger.Info("no upgrade required")
			continue
		}

		logger.Info("upgrade required",
			"schematic_actual", node.RunningSchematic(),
			"schematic_desired", schematic,
			"version_actual", node.RunningVersion(),
			"version_desired", talosVersion,
			"installer", installerImage)

		upgradeRequired = true

		// in dry-run mode, skip the actual upgrade
		if opts.DryRun {
			continue
		}

		// ask for user confirmation
		if t.Confirm() {
			if interactive.ConfirmPrompt(fmt.Sprintf("Do you want to upgrade node %s with installer %s? This will reboot the node.", node.Node.Host, installerImage)) == 'n' {
				logger.Info("skipping upgrade")
				continue
			}
		}

		worklist = append(worklist, node)
	}

	return worklist, upgradeRequired, nil
}

// upgradeNode performs a Talos OS upgrade on a single node using the
// LifecycleService.Upgrade streaming RPC. The flow is:
//
//  1. Resolve the Kubernetes node name (if draining is enabled), so that
//     a drain failure can't be caused by a reason that was knowable before
//     the upgrade mutated the node.
//  2. Pre-pull the installer image via ImageService.Pull.
//  3. Stream LifecycleService.Upgrade, draining progress messages and
//     verifying the final exit code is zero.
//  4. Drain the Kubernetes node (cordon + evict pods) if draining is enabled.
//  5. Issue a Reboot with the configured reboot mode. A gRPC Unavailable or
//     Canceled error from Reboot is treated as success: the node begins
//     shutting down before the RPC response reaches us.
//  6. Close the per-node Talos client (the node is rebooting; the connection
//     is dead anyway).
//  7. Wait for the node to stabilize.
//  8. Re-fetch a Kubernetes clientset and uncordon the node. The clientset
//     is re-fetched (rather than reused from step 4) because the reboot may
//     rotate credentials or invalidate the in-memory kubeconfig, and the
//     control-plane node we proxy through may itself have been upgraded.
//
// The kubeconfig for drain/uncordon operations is fetched via a control-plane
// node client (t.ControlPlaneClient), because the Kubeconfig RPC is only
// available on control-plane nodes. GetKubernetesNodeName uses the node's own
// client, as the Nodename resource is available on all nodes.
//
// The provided logger is expected to already carry the node's attributes.
func upgradeNode(ctx context.Context, t topf.Topf, node *topf.Node, opts Options, logger *slog.Logger) error {
	installerImage := node.ConfigProvider().Machine().Install().Image()

	nodeClient, err := node.Client(ctx)
	if err != nil {
		return err
	}
	defer nodeClient.Close()

	// Resolve the Kubernetes node name up front, so that a drain failure
	// can't be caused by a reason that was knowable before the upgrade.
	k8sNodeName := ""

	if opts.Drain {
		k8sNodeName, err = talosnodedrain.GetKubernetesNodeName(ctx, nodeClient)
		if err != nil {
			return fmt.Errorf("resolving kubernetes node name: %w", err)
		}
	}

	containerdInstance := systemContainerdInstance()

	if err := pullInstallerImage(ctx, nodeClient, containerdInstance, installerImage, logger); err != nil {
		return fmt.Errorf("pulling installer image: %w", err)
	}

	if err := runUpgrade(ctx, nodeClient, containerdInstance, installerImage, logger); err != nil {
		return fmt.Errorf("upgrade: %w", err)
	}

	// Drain the node after the upgrade artifacts are installed but before
	// the reboot, so pods are evicted gracefully.
	if opts.Drain {
		drainClientset, err := newK8sClientset(ctx, t, logger)
		if err != nil {
			return fmt.Errorf("creating kubernetes client for drain: %w", err)
		}

		drainReport := func(u reporter.Update) {
			logger.Info("drain", "k8s_node", k8sNodeName, "message", u.Message)
		}

		logger.Info("draining kubernetes node", "k8s_node", k8sNodeName)

		if err := talosnodedrain.CordonAndDrain(ctx, drainClientset, k8sNodeName, talosnodedrain.Options{
			DrainTimeout: opts.DrainTimeout,
		}, drainReport); err != nil {
			return fmt.Errorf("draining node: %w", err)
		}

		logger.Info("kubernetes node drained", "k8s_node", k8sNodeName)
	}

	logger.Info("upgrade artifacts installed, rebooting node", "reboot_mode", opts.RebootMode.String())

	// Reboot returns once the reboot is initiated. If the node starts
	// shutting down before the gRPC response arrives, the stream is torn
	// down and we get Unavailable or Canceled. Both are expected here.
	if err := nodeClient.Reboot(ctx, client.WithRebootMode(opts.RebootMode)); err != nil &&
		client.StatusCode(err) != codes.Unavailable &&
		!errors.Is(err, context.Canceled) {
		return fmt.Errorf("reboot: %w", err)
	}

	// The node is rebooting; the per-node client is dead. Close it now
	// rather than holding it open across the stabilization window.
	if cerr := nodeClient.Close(); cerr != nil {
		logger.Debug("closing per-node client after reboot", "error", cerr)
	}

	logger.Info("reboot initiated")

	if err = node.Stabilize(ctx, logger, time.Second*30); err != nil {
		return fmt.Errorf("node didn't stabilize: %w", err)
	}

	// After the node has stabilized, uncordon it so that the Kubernetes
	// scheduler can place pods on it again. The clientset is re-fetched
	// because the reboot may have rotated credentials or invalidated the
	// in-memory kubeconfig obtained before the reboot.
	if opts.Drain {
		uncordonClientset, err := newK8sClientset(ctx, t, logger)
		if err != nil {
			return fmt.Errorf("creating kubernetes client for uncordon: %w", err)
		}

		uncordonReport := func(u reporter.Update) {
			logger.Info("uncordon", "k8s_node", k8sNodeName, "message", u.Message)
		}

		if err := talosnodedrain.Uncordon(ctx, uncordonClientset, k8sNodeName, uncordonReport); err != nil {
			return fmt.Errorf("uncordoning node: %w", err)
		}

		logger.Info("kubernetes node uncordoned", "k8s_node", k8sNodeName)
	}

	return nil
}

// newK8sClientset obtains an in-memory Kubernetes clientset by fetching the
// admin kubeconfig through a control-plane node's Talos API. The control-plane
// client is closed after the kubeconfig is retrieved; the returned clientset
// is independent of it.
func newK8sClientset(ctx context.Context, t topf.Topf, logger *slog.Logger) (kubernetes.Interface, error) {
	cpClient, err := t.ControlPlaneClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("creating control-plane client: %w", err)
	}

	clientset, err := taloskubeclient.FromTalosClient(ctx, cpClient)

	if cerr := cpClient.Close(); cerr != nil {
		logger.Debug("closing control-plane client", "error", cerr)
	}

	if err != nil {
		return nil, fmt.Errorf("creating kubernetes client: %w", err)
	}

	return clientset, nil
}

// systemContainerdInstance returns the containerd instance used for pulling
// and running installer artifacts: the CRI driver against the system
// namespace. This matches talosctl's default for upgrades ("system").
func systemContainerdInstance() *common.ContainerdInstance {
	return &common.ContainerdInstance{
		Driver:    common.ContainerDriver_CRI,
		Namespace: common.ContainerdNamespace_NS_SYSTEM,
	}
}

// pullInstallerImage pre-pulls the installer image via the
// ImageService.Pull streaming RPC. Progress messages are logged at debug
// level; the stream completes when the server sends the image name.
func pullInstallerImage(ctx context.Context, c *client.Client, containerdInstance *common.ContainerdInstance, imageRef string, logger *slog.Logger) error {
	stream, err := c.ImageClient.Pull(ctx, &machine.ImageServicePullRequest{
		Containerd: containerdInstance,
		ImageRef:   imageRef,
	})
	if err != nil {
		return err
	}

	for {
		resp, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			return errors.New("image pull stream ended without confirmation")
		}

		if err != nil {
			return err
		}

		switch payload := resp.GetResponse().(type) {
		case *machine.ImageServicePullResponse_PullProgress:
			logger.Debug("image pull progress",
				"layer", payload.PullProgress.GetLayerId(),
				"progress", payload.PullProgress.GetProgress())
		case *machine.ImageServicePullResponse_Name:
			// final message: server reports the resolved image name
			logger.Info("installer image pulled", "image", payload.Name)

			return nil
		}
	}
}

// runUpgrade invokes the LifecycleService.Upgrade streaming RPC and
// drains progress messages until the server sends a terminal exit code. A
// non-zero exit code is surfaced as an error.
func runUpgrade(ctx context.Context, c *client.Client, containerdInstance *common.ContainerdInstance, imageRef string, logger *slog.Logger) error {
	stream, err := c.LifecycleClient.Upgrade(ctx, &machine.LifecycleServiceUpgradeRequest{
		Containerd: containerdInstance,
		Source: &machine.InstallArtifactsSource{
			ImageName: imageRef,
		},
	})
	if err != nil {
		return err
	}

	for {
		resp, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			return errors.New("upgrade stream ended without exit code")
		}

		if err != nil {
			return err
		}

		progress := resp.GetProgress()
		if progress == nil {
			continue
		}

		switch p := progress.GetResponse().(type) {
		case *machine.LifecycleServiceInstallProgress_Message:
			logger.Info("upgrade progress", "message", p.Message)
		case *machine.LifecycleServiceInstallProgress_ExitCode:
			if p.ExitCode != 0 {
				return fmt.Errorf("upgrade failed with exit code %d", p.ExitCode)
			}

			logger.Info("upgrade artifacts installed", "exit_code", p.ExitCode)

			return nil
		}
	}
}

func extractSchematicAndVersion(input string) (schematic, version string, err error) {
	// Pattern matches: */<schematic>:v<version>
	pattern := `^.*/([a-zA-Z0-9]+):v?(.+)$`

	re := regexp.MustCompile(pattern)
	matches := re.FindStringSubmatch(input)

	if len(matches) != 3 {
		return "", "", errors.New("invalid format: expected */<schematic>:v?<version>")
	}

	schematic = matches[1]
	version = matches[2]

	return schematic, version, nil
}
