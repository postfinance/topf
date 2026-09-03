// Copyright 2026 PostFinance AG
// SPDX-License-Identifier: MIT

package topf

import (
	"cmp"
	"fmt"
	"log/slog"
	"strings"

	"github.com/postfinance/topf/pkg/config"
	"github.com/siderolabs/talos/pkg/images"
	talosconfig "github.com/siderolabs/talos/pkg/machinery/config"
	"github.com/siderolabs/talos/pkg/machinery/config/bundle"
	"github.com/siderolabs/talos/pkg/machinery/constants"
	"github.com/siderolabs/talos/pkg/machinery/gendata"
	"github.com/siderolabs/talos/pkg/machinery/resources/runtime"
	"github.com/siderolabs/talos/pkg/machinery/version"
)

var (
	// DefaultSchematic is the schematic used by Talos when no extensions are configured
	DefaultSchematic = images.DefaultInstallerImageSchematic
	// DefaultFactory is the default Talos image factory address
	DefaultFactory = gendata.ImageFactory
	// DefaultPlatform is the default Talos platform identifier
	DefaultPlatform = constants.PlatformMetal
)

// Node contains runtime state information about a Talos node
// wrapping the configured Node from Topf config
type Node struct {
	t    Topf
	Node *config.Node

	MachineStatus     runtime.MachineStatusSpec
	runningVersion    string
	runningSchematic  string
	resolvedSchematic string
	ConfigBundle      *bundle.Bundle `yaml:"-"`
	Error             error          `yaml:",omitempty"`
}

// TalosVersion returns the Talos version to use for config generation.
// Fallback chain: running (from live node) -> topf.yaml -> bundled Talos version.
func (n *Node) TalosVersion() string {
	return strings.TrimPrefix(cmp.Or(n.runningVersion, n.t.Config().TalosVersion, version.Tag), "v")
}

// RunningVersion returns the Talos version reported by the live node.
// Empty if collectNodeInfo has not been called.
func (n *Node) RunningVersion() string {
	return n.runningVersion
}

// RunningSchematic returns the schematic ID reported by the live node.
// Empty if collectNodeInfo has not been called.
func (n *Node) RunningSchematic() string {
	return n.runningSchematic
}

// ResolvedSchematic returns the schematic ID after @-prefixed references
// have been expanded. Empty if generateNodeConfig has not been called.
func (n *Node) ResolvedSchematic() string {
	return n.resolvedSchematic
}

// InstallerImage returns the fully resolved installer image for this node.
// The schematic ID is the resolved value (after @-prefixed references have been expanded).
// Factory, platform, talosVersion, and secureboot resolve per node -> cluster config -> default.
func (n *Node) InstallerImage() string {
	cfg := n.t.Config()
	factory := cmp.Or(n.Node.Factory, cfg.Factory, DefaultFactory)
	platform := cmp.Or(n.Node.Platform, cfg.Platform, DefaultPlatform)
	schematic := cmp.Or(n.resolvedSchematic, DefaultSchematic)
	talosVersion := strings.TrimPrefix(cmp.Or(n.Node.TalosVersion, cfg.TalosVersion, version.Tag), "v")

	installer := platform + "-installer"
	if n.Node.SecureBoot || cfg.SecureBoot {
		installer += "-secureboot"
	}

	return fmt.Sprintf("%s/%s/%s:v%s", factory, installer, schematic, talosVersion)
}

// MarshalYAML implements custom YAML marshalling to properly serialize the Error field
func (n *Node) MarshalYAML() (any, error) {
	aux := &struct {
		Node          *config.Node              `yaml:"node"`
		MachineStatus runtime.MachineStatusSpec `yaml:"machinestatus"`
		Schematic     string                    `yaml:"schematic"`
		TalosVersion  string                    `yaml:"talosversion"`
		Error         string                    `yaml:"error,omitempty"`
	}{
		Node:          n.Node,
		MachineStatus: n.MachineStatus,
		Schematic:     n.runningSchematic,
		TalosVersion:  n.runningVersion,
	}

	if n.Error != nil {
		aux.Error = n.Error.Error()
	}

	return aux, nil
}

// Attrs returns a key/value for use with slog.Logger.With
func (n *Node) Attrs() slog.Attr {
	return slog.String("node", n.Node.Host)
}

// ConfigProvider returns the config bundle associated with the node's role
func (n *Node) ConfigProvider() talosconfig.Provider {
	var provider talosconfig.Provider

	if n.Node.Role == config.RoleControlPlane {
		provider = n.ConfigBundle.ControlPlaneCfg
	} else {
		provider = n.ConfigBundle.WorkerCfg
	}

	return provider
}

// InstallerImageRef returns the installer image from the generated config.
func (n *Node) InstallerImageRef() string {
	if image := n.ConfigProvider().Machine().Install().Image(); image != "" {
		return image
	}

	if unattended := n.ConfigProvider().UnattendedInstallConfig(); unattended != nil {
		return unattended.InstallerImage()
	}

	return ""
}
