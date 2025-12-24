package config

import (
	"errors"
	"net/netip"

	"go.yaml.in/yaml/v4"
)

// Node is the struct containing the configuration info for a node
type Node struct {
	// Host is the name of the node. Can be FQDN or short name
	Host string `yaml:"host"`
	// IP is optional and used to connect to nodes directly instead of using DNS to resolve "host"
	IP   *netip.Addr    `yaml:"ip,omitempty"`
	Role NodeRole       `yaml:"role"`
	Data map[string]any `yaml:"data,omitempty"`
}

// Endpoint returns the IP address if set, otherwise returns the Host.
// Use this for connections; use Host for display/logging and certificate validation.
func (n *Node) Endpoint() string {
	if n.IP != nil {
		return n.IP.String()
	}

	return n.Host
}

// UnmarshalYAML implements yaml.Unmarshaler and performs additional validation
func (n *Node) UnmarshalYAML(yamlNode *yaml.Node) error {
	type raw Node

	if err := yamlNode.Decode((*raw)(n)); err != nil {
		return err
	}

	if n.Host == "" {
		return errors.New("node 'host' can't be empty")
	}

	if n.Role == "" {
		return errors.New("node 'role' can't be empty")
	}

	return nil
}
