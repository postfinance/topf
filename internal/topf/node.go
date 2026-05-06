// Copyright 2026 PostFinance AG
// SPDX-License-Identifier: MIT

package topf

import (
	"cmp"
	"fmt"
	"log/slog"
	"strings"

	"github.com/postfinance/topf/pkg/config"
	talosconfig "github.com/siderolabs/talos/pkg/machinery/config"
	"github.com/siderolabs/talos/pkg/machinery/config/bundle"
	"github.com/siderolabs/talos/pkg/machinery/resources/runtime"
	"github.com/siderolabs/talos/pkg/machinery/version"
)

const (
	// DefaultSchematic is the schematic used by Talos when no extensions are configured
	// TODO (v1.14): replace with images.DefaultInstallerImageSchematic
	DefaultSchematic = "376567988ad370138ad8b2698212367b8edcb69b5fd68c80be1f2ec7d603b4ba"
	// DefaultFactory is the default Talos image factory address
	// TODO (v1.14): replace with gendata.ImageFactory
	DefaultFactory = "factory.talos.dev"
	// DefaultPlatform is the default Talos platform identifier
	DefaultPlatform = "metal"
)

// Node contains runtime state information about a Talos node
// wrapping the configured Node from Topf config
type Node struct {
	t    Topf
	Node *config.Node

	MachineStatus    runtime.MachineStatusSpec
	runningVersion   string
	runningSchematic string
	ConfigBundle     *bundle.Bundle `yaml:"-"`
	Error            error          `yaml:",omitempty"`
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

// InstallerImage returns the fully resolved installer image for this node.
// Resolution order for factory and platform: node -> cluster config -> default.
func (n *Node) InstallerImage() string {
	cfg := n.t.Config()
	factory := cmp.Or(n.Node.Factory, cfg.Factory, DefaultFactory)
	platform := cmp.Or(n.Node.Platform, cfg.Platform, DefaultPlatform)
	schematic := cmp.Or(cfg.SchematicID, DefaultSchematic)
	talosVersion := strings.TrimPrefix(cmp.Or(cfg.TalosVersion, version.Tag), "v")

	return fmt.Sprintf("%s/%s-installer/%s:v%s", factory, platform, schematic, talosVersion)
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
