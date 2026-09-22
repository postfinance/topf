// Copyright 2026 PostFinance AG
// SPDX-License-Identifier: MIT

// Package upgrade contains the logic to upgrade Talos OS on cluster nodes
package upgrade

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/postfinance/topf/internal/interactive"
	"github.com/postfinance/topf/internal/nodepool"
	"github.com/postfinance/topf/internal/progress"
	"github.com/postfinance/topf/internal/topf"
	taloskubeclient "github.com/siderolabs/talos/cmd/talosctl/pkg/talos/kubeclient"
	talosnodedrain "github.com/siderolabs/talos/cmd/talosctl/pkg/talos/nodedrain"
	"github.com/siderolabs/talos/pkg/machinery/api/common"
	"github.com/siderolabs/talos/pkg/machinery/api/machine"
	"github.com/siderolabs/talos/pkg/machinery/client"
	"github.com/siderolabs/talos/pkg/machinery/resources/runtime"
	"github.com/siderolabs/talos/pkg/reporter"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/kubectl/pkg/drain"

	"github.com/blang/semver/v4"
)

// Options contains the options for the upgrade execution
type Options struct {
	// Only show what upgrades would be performed without actually upgrading
	DryRun bool

	// RebootMode controls how the node is rebooted after the upgrade.
	RebootMode machine.RebootRequest_Mode

	// Force skips etcd health checks on the legacy MachineService.Upgrade
	// RPC (Talos < 1.13). Has no effect on nodes running Talos >= 1.13,
	// where the LifecycleService.Upgrade RPC validates etcd health
	// server-side and has no force knob.
	Force bool

	// AllowNotReady allows upgrading nodes that are not ready (have unmet conditions).
	AllowNotReady bool

	// Drain controls whether the Kubernetes node is cordoned and drained
	// before the reboot and uncordoned after the node becomes Ready again.
	Drain bool

	// DrainTimeout is the maximum time to wait for pod evictions to complete.
	DrainTimeout time.Duration

	// StabilizationDuration is how long a node must stay ready after
	// rebooting before it is considered stable.
	StabilizationDuration time.Duration

	// DeleteIfEvictionFails retries the drain with direct pod deletion
	// (DELETE instead of EVICT, bypassing PDBs) if the graceful drain fails.
	DeleteIfEvictionFails bool

	// Stage installs upgrade artifacts without rebooting. The node is
	// labeled/tainted (if StageLabels/StageTaints are set) so an external
	// controller or human can reboot it later.
	Stage bool

	// StageLabels are Kubernetes node labels (key=value) applied after
	// staging. Requires Stage.
	StageLabels []string

	// StageAnnotations are Kubernetes node annotations (key=value) applied
	// after staging. Requires Stage.
	StageAnnotations []string

	// StageTaints are Kubernetes node taints (key=value:Effect) applied
	// after staging. Requires Stage.
	StageTaints []string

	// stagePatch is the pre-parsed node patch (labels + annotations + taints), built
	// once in validateOptions. Zero-valued if no labels/annotations/taints are set.
	stagePatch corev1.Node

	// MaxParallel controls how many worker nodes are upgraded concurrently.
	// Control-plane nodes are always upgraded one at a time.
	MaxParallel nodepool.MaxParallel

	// ReporterMode controls per-node progress rendering (reporter vs slog).
	ReporterMode reporter.OutputMode
}

// Execute performs the Talos OS upgrades for all nodes in the cluster
func Execute(ctx context.Context, t topf.Topf, opts Options) error {
	logger := t.Logger().With("command", "upgrade")

	if err := validateOptions(&opts); err != nil {
		return err
	}

	nodes, err := t.FilteredNodes(ctx)
	if err != nil {
		return err
	}

	if t.Confirm() && opts.MaxParallel.Resolve(len(nodes)) > 1 {
		return errors.New("--max-parallel > 1 requires --confirm=false: interactive prompts cannot be answered during concurrent upgrades")
	}

	if err := preChecks(logger, nodes, opts); err != nil {
		return err
	}

	worklist, err := plan(logger, nodes, opts)
	if err != nil {
		return err
	}

	if opts.DryRun {
		if len(worklist) > 0 {
			return topf.ErrDryRunChangesDetected
		}

		return nil
	}

	controlPlane, workers := nodepool.PartitionByRole(worklist)

	concurrency := opts.MaxParallel.Resolve(len(workers))
	rep := progress.NewReporter(opts.ReporterMode, concurrency)

	// control-plane nodes are upgraded one at a time to preserve etcd quorum
	for _, node := range controlPlane {
		if err := upgradeNode(ctx, t, node, opts, progress.New(logger.With(node.Attrs()), rep, node.Node.Host)); err != nil {
			return err
		}
	}

	if len(workers) > 0 {
		logger.Info("upgrading worker nodes", "count", len(workers), "concurrency", concurrency)

		return nodepool.RunConcurrent(ctx, workers, concurrency,
			func(ctx context.Context, node *topf.Node, logger *slog.Logger) error {
				return upgradeNode(ctx, t, node, opts, progress.New(logger, rep, node.Node.Host))
			}, logger)
	}

	return nil
}

func validateOptions(opts *Options) error {
	if !opts.Stage {
		if len(opts.StageLabels) > 0 || len(opts.StageAnnotations) > 0 || len(opts.StageTaints) > 0 {
			return errors.New("--stage-label, --stage-annotation, and --stage-taint require --stage")
		}

		return nil
	}

	if len(opts.StageLabels) == 0 && len(opts.StageAnnotations) == 0 && len(opts.StageTaints) == 0 {
		return nil
	}

	stagePatch := corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Labels:      map[string]string{},
			Annotations: map[string]string{},
		},
	}

	for _, l := range opts.StageLabels {
		k, v, ok := strings.Cut(l, "=")
		if !ok || k == "" {
			return fmt.Errorf("invalid label %q: expected key=value", l)
		}

		stagePatch.Labels[k] = v
	}

	for _, a := range opts.StageAnnotations {
		k, v, ok := strings.Cut(a, "=")
		if !ok || k == "" {
			return fmt.Errorf("invalid annotation %q: expected key=value", a)
		}

		stagePatch.Annotations[k] = v
	}

	for _, tw := range opts.StageTaints {
		kv, effect, ok := strings.Cut(tw, ":")
		if !ok || effect == "" {
			return fmt.Errorf("invalid taint %q: expected key[=value]:Effect", tw)
		}

		k, v := kv, ""
		if before, after, ok := strings.Cut(kv, "="); ok {
			k, v = before, after
		}

		if k == "" {
			return fmt.Errorf("invalid taint %q: expected key[=value]:Effect", tw)
		}

		stagePatch.Spec.Taints = append(stagePatch.Spec.Taints, corev1.Taint{
			Key:    k,
			Value:  v,
			Effect: corev1.TaintEffect(effect),
		})
	}

	opts.stagePatch = stagePatch

	return nil
}

// preChecks verifies that every node is reachable, in a processable stage
// (maintenance is not processable for upgrades), and ready, reporting all
// problems at once. The readiness check is skipped when opts.AllowNotReady.
func preChecks(logger *slog.Logger, nodes []*topf.Node, opts Options) error {
	abort := false

	for _, node := range nodes {
		logger := logger.With(node.Attrs())

		if node.Error != nil {
			logger.Error("node pre-checks", "error", node.Error)

			abort = true

			continue
		}

		if !opts.AllowNotReady && !node.MachineStatus.Status.Ready {
			logger.Error("node not ready", "unmet conditions", node.MachineStatus.Status.UnmetConditions)

			abort = true
		}

		st := node.MachineStatus.Stage
		switch {
		case st == runtime.MachineStageMaintenance:
			logger.Error("node is in maintenance mode; apply its configuration first (topf apply), then upgrade")

			abort = true
		case !slices.Contains([]runtime.MachineStage{runtime.MachineStageRunning, runtime.MachineStageBooting}, st):
			logger.Error("node in unprocessable stage", "stage", st.String())

			abort = true
		}
	}

	if abort {
		return errors.New("aborting due to errors with some nodes")
	}

	return nil
}

// plan determines which nodes require an upgrade
func plan(logger *slog.Logger, nodes []*topf.Node, opts Options) (worklist []*topf.Node, err error) {
	for _, node := range nodes {
		logger := logger.With(node.Attrs())

		installerImage := node.InstallerImageRef()

		schematic, talosVersion, err := extractSchematicAndVersion(installerImage)
		if err != nil {
			return nil, fmt.Errorf("couldn't extract schematic and version from installer image '%s': %w", installerImage, err)
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

		if opts.Stage && !supportsLifecycleUpgrade(node.RunningVersion()) {
			return nil, fmt.Errorf("node %s runs Talos %s: --stage requires Talos >= 1.13", node.Node.Host, node.RunningVersion())
		}

		worklist = append(worklist, node)
	}

	return worklist, nil
}

func upgradeNode(ctx context.Context, t topf.Topf, node *topf.Node, opts Options, p progress.Progress) error {
	if t.Confirm() {
		prompt := fmt.Sprintf("Do you want to upgrade node %s with installer %s?", node.Node.Host, node.InstallerImageRef())

		if interactive.ConfirmPrompt(prompt) == 'n' {
			p.Info("skipping upgrade")
			return nil
		}
	}

	if supportsLifecycleUpgrade(node.RunningVersion()) {
		return upgradeNodeLifecycle(ctx, t, node, opts, p)
	}

	p.Info("node running Talos < 1.13, using legacy upgrade RPC", "running_version", node.RunningVersion())

	return upgradeNodeLegacy(ctx, node, opts, p)
}

func supportsLifecycleUpgrade(runningVersion string) bool {
	if runningVersion == "" {
		return true
	}

	v, err := semver.ParseTolerant(runningVersion)
	if err != nil {
		return true
	}

	return v.GTE(semver.MustParse("1.13.0"))
}

func upgradeNodeLifecycle(ctx context.Context, t topf.Topf, node *topf.Node, opts Options, p progress.Progress) error {
	installerImage := node.InstallerImageRef()

	nodeClient, err := node.Client(ctx)
	if err != nil {
		return err
	}
	defer nodeClient.Close()

	k8sNodeName := ""

	if opts.Drain || (opts.Stage && hasStagePatch(opts.stagePatch)) {
		k8sNodeName, err = talosnodedrain.GetKubernetesNodeName(ctx, nodeClient)
		if err != nil {
			return fmt.Errorf("resolving kubernetes node name: %w", err)
		}
	}

	containerdInstance := systemContainerdInstance()

	if err := pullInstallerImage(ctx, nodeClient, containerdInstance, installerImage, p); err != nil {
		p.Fail("pulling installer image failed")

		return fmt.Errorf("pulling installer image: %w", err)
	}

	if err := runUpgrade(ctx, nodeClient, containerdInstance, installerImage, p); err != nil {
		p.Fail("upgrade failed")

		return fmt.Errorf("upgrade: %w", err)
	}

	if opts.Stage {
		if hasStagePatch(opts.stagePatch) {
			if err := applyStagePatch(ctx, t, k8sNodeName, opts.stagePatch, p); err != nil {
				return err
			}
		}

		p.Done("upgrade staged; node not rebooted — reboot manually to complete the upgrade")

		return nil
	}

	if opts.Drain {
		if err := drainNode(ctx, t, opts, k8sNodeName, p); err != nil {
			p.Fail("draining node failed")

			return err
		}
	}

	p.Running(fmt.Sprintf("rebooting node (mode %s)", opts.RebootMode.String()))

	if err := nodeClient.Reboot(ctx, client.WithRebootMode(opts.RebootMode)); err != nil {
		p.Fail("reboot failed")

		return fmt.Errorf("reboot: %w", err)
	}

	if cerr := nodeClient.Close(); cerr != nil {
		p.Debug("closing per-node client after reboot", "error", cerr)
	}

	p.Running("reboot initiated")

	if err = node.Stabilize(ctx, p, opts.StabilizationDuration); err != nil {
		p.Fail("node didn't stabilize")

		return fmt.Errorf("node didn't stabilize: %w", err)
	}

	p.Done("node stable")

	if opts.Drain {
		uncordonClientset, err := newK8sClientset(ctx, t, p)
		if err != nil {
			return fmt.Errorf("creating kubernetes client for uncordon: %w", err)
		}

		uncordonReport := func(u reporter.Update) {
			p.Running("uncordon: " + u.Message)
		}

		if err := talosnodedrain.Uncordon(ctx, uncordonClientset, k8sNodeName, uncordonReport); err != nil {
			p.Fail("uncordoning node failed")

			return fmt.Errorf("uncordoning node: %w", err)
		}

		p.Done("kubernetes node uncordoned")
	}

	return nil
}

//nolint:staticcheck // the non-deprecated replacement (LifecycleClient.Upgrade) requires Talos >= 1.13
func upgradeNodeLegacy(ctx context.Context, node *topf.Node, opts Options, p progress.Progress) error {
	installerImage := node.InstallerImageRef()

	nodeClient, err := node.Client(ctx)
	if err != nil {
		return err
	}
	defer nodeClient.Close()

	p.Info("issuing legacy upgrade", "installer", installerImage, "force", opts.Force)

	_, err = nodeClient.MachineClient.Upgrade(ctx, &machine.UpgradeRequest{
		Image:      installerImage,
		Preserve:   true, // talos default since v1.8+
		Force:      opts.Force,
		RebootMode: toLegacyRebootMode(opts.RebootMode),
	})
	if err != nil {
		p.Fail("legacy upgrade failed")

		return fmt.Errorf("legacy upgrade: %w", err)
	}

	p.Running("upgrade initiated")

	if err = node.Stabilize(ctx, p, opts.StabilizationDuration); err != nil {
		p.Fail("node didn't stabilize")

		return fmt.Errorf("node didn't stabilize: %w", err)
	}

	p.Done("node stable")

	return nil
}

func toLegacyRebootMode(mode machine.RebootRequest_Mode) machine.UpgradeRequest_RebootMode {
	switch mode {
	case machine.RebootRequest_POWERCYCLE:
		return machine.UpgradeRequest_POWERCYCLE
	default:
		return machine.UpgradeRequest_DEFAULT
	}
}

func newK8sClientset(ctx context.Context, t topf.Topf, p progress.Progress) (kubernetes.Interface, error) {
	cpClient, err := t.ControlPlaneClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("creating control-plane client: %w", err)
	}

	clientset, err := taloskubeclient.FromTalosClient(ctx, cpClient)

	if cerr := cpClient.Close(); cerr != nil {
		p.Debug("closing control-plane client", "error", cerr)
	}

	if err != nil {
		return nil, fmt.Errorf("creating kubernetes client: %w", err)
	}

	return clientset, nil
}

func drainNode(ctx context.Context, t topf.Topf, opts Options, k8sNodeName string, p progress.Progress) error {
	clientset, err := newK8sClientset(ctx, t, p)
	if err != nil {
		return fmt.Errorf("creating kubernetes client for drain: %w", err)
	}

	p.Running("draining kubernetes node")

	drainCtx, cancel := context.WithTimeout(ctx, opts.DrainTimeout)
	defer cancel()

	if err := cordonNode(drainCtx, clientset, k8sNodeName, p); err != nil {
		return err
	}

	gracefulErr := drainPods(drainCtx, clientset, k8sNodeName, opts.DrainTimeout, false, p)
	if gracefulErr == nil {
		p.Done("kubernetes node drained")

		return nil
	}

	if !opts.DeleteIfEvictionFails {
		return fmt.Errorf("draining node: %w", gracefulErr)
	}

	p.Warn("graceful drain failed, retrying with forced pod deletion", "k8s_node", k8sNodeName, "error", gracefulErr)

	forcedErr := drainPods(ctx, clientset, k8sNodeName, opts.DrainTimeout, true, p)
	if forcedErr != nil {
		return fmt.Errorf("draining node: %w", errors.Join(gracefulErr, forcedErr))
	}

	p.Done("kubernetes node drained (forced)")

	return nil
}

func cordonNode(ctx context.Context, clientset kubernetes.Interface, nodeName string, p progress.Progress) error {
	node, err := clientset.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("getting node %q: %w", nodeName, err)
	}

	drainer := &drain.Helper{
		Ctx:    ctx,
		Client: clientset,
		Out:    io.Discard,
		ErrOut: io.Discard,
	}

	p.Running("cordoning kubernetes node")

	if err := drain.RunCordonOrUncordon(drainer, node, true); err != nil {
		return fmt.Errorf("cordoning node %q: %w", nodeName, err)
	}

	p.Running("kubernetes node cordoned")

	return nil
}

func drainPods(ctx context.Context, clientset kubernetes.Interface, nodeName string, timeout time.Duration, deleteIfEvictionFails bool, p progress.Progress) error {
	drainCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	action := "evicting"
	if deleteIfEvictionFails {
		action = "deleting"
	}

	var announced sync.Map

	drainer := &drain.Helper{
		Ctx:                 drainCtx,
		Client:              clientset,
		Force:               true,
		GracePeriodSeconds:  -1,
		IgnoreAllDaemonSets: true,
		DeleteEmptyDirData:  true,
		DisableEviction:     deleteIfEvictionFails,
		Timeout:             timeout,
		Out:                 io.Discard,
		ErrOut:              io.Discard,
		OnPodDeletionOrEvictionStarted: func(pod *corev1.Pod, usingEviction bool) {
			key := pod.Namespace + "/" + pod.Name
			if _, dup := announced.LoadOrStore(key, struct{}{}); dup {
				return
			}

			verb := "deleting"
			if usingEviction {
				verb = "evicting"
			}

			p.Running(fmt.Sprintf("%s pod %s/%s", verb, pod.Namespace, pod.Name))
		},
		OnPodDeletionOrEvictionFinished: func(pod *corev1.Pod, _ bool, err error) {
			announced.Delete(pod.Namespace + "/" + pod.Name)

			if err != nil {
				p.Warn("drain", "k8s_node", nodeName, "message", fmt.Sprintf("failed to %s pod %s/%s: %v", action, pod.Namespace, pod.Name, err))

				return
			}

			p.Running(fmt.Sprintf("%s pod %s/%s", action, pod.Namespace, pod.Name))
		},
	}

	if err := drain.RunNodeDrain(drainer, nodeName); err != nil {
		return fmt.Errorf("draining node %q: %w", nodeName, err)
	}

	return nil
}

func hasStagePatch(p corev1.Node) bool {
	return len(p.Labels) > 0 || len(p.Annotations) > 0 || len(p.Spec.Taints) > 0
}

func applyStagePatch(ctx context.Context, t topf.Topf, nodeName string, patch corev1.Node, p progress.Progress) error {
	clientset, err := newK8sClientset(ctx, t, p)
	if err != nil {
		return fmt.Errorf("creating kubernetes client for staging: %w", err)
	}

	patchData, err := json.Marshal(patch)
	if err != nil {
		return fmt.Errorf("marshaling node patch: %w", err)
	}

	p.Running("applying node patch")

	if _, err := clientset.CoreV1().Nodes().Patch(ctx, nodeName, types.StrategicMergePatchType, patchData, metav1.PatchOptions{}); err != nil {
		return fmt.Errorf("patching node %q: %w", nodeName, err)
	}

	return nil
}

func systemContainerdInstance() *common.ContainerdInstance {
	return &common.ContainerdInstance{
		Driver:    common.ContainerDriver_CRI,
		Namespace: common.ContainerdNamespace_NS_SYSTEM,
	}
}

func pullInstallerImage(ctx context.Context, c *client.Client, containerdInstance *common.ContainerdInstance, imageRef string, p progress.Progress) error {
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
			msg := payload.PullProgress.GetProgress().Fmt()
			if msg == "" {
				continue
			}

			p.Running(fmt.Sprintf("layer %s: %s", payload.PullProgress.GetLayerId(), msg))
		case *machine.ImageServicePullResponse_Name:
			p.Done("pulled installer image " + payload.Name)

			return nil
		}
	}
}

func runUpgrade(ctx context.Context, c *client.Client, containerdInstance *common.ContainerdInstance, imageRef string, p progress.Progress) error {
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

		switch update := progress.GetResponse().(type) {
		case *machine.LifecycleServiceInstallProgress_Message:
			p.Running(update.Message)
		case *machine.LifecycleServiceInstallProgress_ExitCode:
			if update.ExitCode != 0 {
				return fmt.Errorf("upgrade failed with exit code %d", update.ExitCode)
			}

			p.Done("upgrade artifacts installed")

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
