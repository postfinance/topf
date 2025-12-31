package topf

import (
	"log/slog"

	"github.com/postfinance/topf/pkg/config"
	"github.com/siderolabs/talos/pkg/machinery/config/bundle"
	"github.com/siderolabs/talos/pkg/machinery/resources/runtime"
)

const (
	// DefaultSchematic is the schematic version used by Talos when no
	// extensions or command line flags are defined
	DefaultSchematic = "376567988ad370138ad8b2698212367b8edcb69b5fd68c80be1f2ec7d603b4ba"
)

// Node contains runtime state information about a Talos node
// wrapping the configured Node from Topf config
type Node struct {
	t    Topf
	Node *config.Node

	MachineStatus runtime.MachineStatusSpec
	Schematic     string
	TalosVersion  string
	ConfigBundle  *bundle.Bundle `yaml:"-"`
	Error         error          `yaml:",omitempty"`
}

// MarshalYAML implements custom YAML marshalling to properly serialize the Error field
func (n *Node) MarshalYAML() (interface{}, error) {
	// Create a struct with only the exported fields we want to marshal
	aux := &struct {
		Node          *config.Node              `yaml:"node"`
		MachineStatus runtime.MachineStatusSpec `yaml:"machinestatus"`
		Schematic     string                    `yaml:"schematic"`
		TalosVersion  string                    `yaml:"talosversion"`
		Error         string                    `yaml:"error,omitempty"`
	}{
		Node:          n.Node,
		MachineStatus: n.MachineStatus,
		Schematic:     n.Schematic,
		TalosVersion:  n.TalosVersion,
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
