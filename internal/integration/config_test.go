// Copyright 2026 PostFinance AG
// SPDX-License-Identifier: MIT

//go:build integration

package integration

import (
	"context"
	"os"
	"path/filepath"
	"time"

	"github.com/postfinance/topf/internal/cmd/apply"
	"github.com/postfinance/topf/internal/cmd/kubeconfig"
	"github.com/postfinance/topf/internal/topf"
	"github.com/siderolabs/go-retry/retry"
	"github.com/siderolabs/talos/pkg/machinery/api/machine"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// ConfigSuite asserts the cluster is reconciled: dry-run converges, and a
// config change is applied through to the Kubernetes nodes.
type ConfigSuite struct {
	TopfSuite

	labelPatchPath string
}

func (s *ConfigSuite) TestDryRunCleanAfterApply() {
	err := apply.Execute(s.T().Context(), s.Runtime(), apply.Options{
		DryRun: true,
		Mode:   machine.ApplyConfigurationRequest_AUTO,
	})

	s.NoError(err, "dry-run after apply must report no changes")
}

func (s *ConfigSuite) TestLabelPatchLifecycle() {
	s.labelPatchPath = s.writeLabelPatch()
	s.ReloadRuntime()

	err := apply.Execute(s.T().Context(), s.Runtime(), apply.Options{
		DryRun: true,
		Mode:   machine.ApplyConfigurationRequest_AUTO,
	})
	s.ErrorIs(err, topf.ErrDryRunChangesDetected, "dry-run must detect the label patch")

	err = apply.Execute(s.T().Context(), s.Runtime(), apply.Options{
		StabilizationDuration: 5 * time.Second,
		Mode:                  machine.ApplyConfigurationRequest_AUTO,
	})
	s.Require().NoError(err, "apply with label patch failed")

	err = apply.Execute(s.T().Context(), s.Runtime(), apply.Options{
		DryRun: true,
		Mode:   machine.ApplyConfigurationRequest_AUTO,
	})
	s.NoError(err, "dry-run after reconcile must report no changes")

	s.WaitForNodeLabel(s.T().Context(), "topf.e2e", "works")
}

func (s *ConfigSuite) TearDownSuite() {
	if s.labelPatchPath != "" {
		if err := os.Remove(s.labelPatchPath); err != nil && !os.IsNotExist(err) {
			s.T().Logf("failed to remove label patch %s: %v", s.labelPatchPath, err)
		}
	}
}

func (s *TopfSuite) WaitForNodeLabel(ctx context.Context, key, value string) {
	err := retry.Constant(2*time.Minute, retry.WithUnits(5*time.Second)).Retry(func() error {
		labeled, err := labeledNodes(ctx, s.Runtime(), key, value)
		if err != nil {
			return retry.ExpectedError(err)
		}

		if labeled != *expectedNodesNum {
			return retry.ExpectedErrorf("%d/%d nodes carry label %s=%s", labeled, *expectedNodesNum, key, value)
		}

		return nil
	})
	s.Require().NoErrorf(err, "label %s=%s did not reach all nodes within 2 minutes", key, value)
}

func labeledNodes(ctx context.Context, t topf.Topf, key, value string) (int, error) {
	clientset, err := kubeClient(ctx, t)
	if err != nil {
		return 0, err
	}

	nodes, err := clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return 0, err
	}

	labeled := 0

	for i := range nodes.Items {
		if nodes.Items[i].Labels[key] == value {
			labeled++
		}
	}

	return labeled, nil
}

func kubeClient(ctx context.Context, t topf.Topf) (*kubernetes.Clientset, error) {
	cfg, err := kubeconfig.Generate(t, 1*time.Hour)
	if err != nil {
		return nil, err
	}

	restConfig, err := clientcmd.NewDefaultClientConfig(*cfg, &clientcmd.ConfigOverrides{}).ClientConfig()
	if err != nil {
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}

	// probe the connection; the suite polls on errors
	_, err = clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{Limit: 1})
	if err != nil {
		return nil, err
	}

	return clientset, nil
}

func (s *TopfSuite) writeLabelPatch() string {
	patch := `---
apiVersion: v1alpha1
kind: KubeNodeConfig
labels:
    topf.e2e: works
`

	allDir := filepath.Join(s.Runtime().Config().PatchesDir, "all")

	s.Require().NoError(os.MkdirAll(allDir, 0o750))

	path := filepath.Join(allDir, "01-e2e-label.yaml")
	s.Require().NoError(os.WriteFile(path, []byte(patch), 0o600))

	return path
}
