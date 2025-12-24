package config

import "fmt"

// NodeRole represents the role of a node in the cluster
type NodeRole string

const (
	// RoleWorker is the role for kubenetes worker node
	RoleWorker NodeRole = "worker"

	// RoleControlPlane is the role for kubernetes control plane nodes
	RoleControlPlane NodeRole = "control-plane"
)

// UnmarshalText implements encoding.TextUnmarshaler
func (nr *NodeRole) UnmarshalText(text []byte) error {
	s := string(text)

	switch NodeRole(s) {
	case RoleWorker, RoleControlPlane:
		*nr = NodeRole(s)
		return nil
	default:
		return fmt.Errorf("invalid node role %q: must be either %q or %q",
			s, RoleWorker, RoleControlPlane)
	}
}

// MarshalText implements encoding.TextMarshaler
func (nr NodeRole) MarshalText() ([]byte, error) {
	switch nr {
	case RoleWorker, RoleControlPlane:
		return []byte(nr), nil
	default:
		return nil, fmt.Errorf("invalid node role %q", nr)
	}
}
