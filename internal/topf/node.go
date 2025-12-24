package topf

import (
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
