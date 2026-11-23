// Copyright 2026 PostFinance AG
// SPDX-License-Identifier: MIT

package config

import (
	"cmp"
	"errors"
	"net/netip"
	"slices"

	"go.yaml.in/yaml/v4"
)

// Node is the struct containing the configuration info for a node
type Node struct {
	// Host is the name of the node. Can be FQDN or short name
	Host string `yaml:"host"`
	// IP is optional and used instead of Host to address the node
	IP *netip.Addr `yaml:"ip,omitempty"`
	// Endpoint overrides the address (host or host:port) used to connect to the node's Talos API
	Endpoint string         `yaml:"endpoint,omitempty"`
	Role     NodeRole       `yaml:"role"`
	Classes  []string       `yaml:"classes,omitempty"`
	Data     map[string]any `yaml:"data,omitempty"`

	// TalosVersion overrides the cluster-level Talos version for this node
	TalosVersion string `yaml:"talosVersion,omitempty"`
	// SchematicID overrides the cluster-level schematic ID for this node
	SchematicID string `yaml:"schematicId,omitempty"`
	// Factory overrides the cluster-level image factory address for this node
	Factory string `yaml:"factory,omitempty"`
	// Platform overrides the cluster-level platform identifier for this node
	Platform string `yaml:"platform,omitempty"`
	// SecureBoot enables the secure boot installer variant for this node
	SecureBoot bool `yaml:"secureboot,omitempty"`
	// RuntimeData holds values collected from the live node.
	RuntimeData RuntimeData `yaml:"runtimeData,omitempty"`
}

// RuntimeData contains values collected from a live node at runtime.
type RuntimeData struct {
	// TalosVersion is the Talos version reported by the live node. (without v prefix)
	TalosVersion string `yaml:"talosVersion,omitempty"`
	// SchematicID is the schematic ID reported by the live node.
	SchematicID string `yaml:"schematicId,omitempty"`
	// Stage is the machine stage reported by the live node
	Stage string `yaml:"stage,omitempty"`
}

// Address returns the address under which the cluster reaches the node: IP if set, otherwise Host.
func (n *Node) Address() string {
	if n.IP != nil {
		return n.IP.String()
	}

	return n.Host
}

// EffectiveEndpoint returns the address topf connects to: Endpoint if set, otherwise Address.
func (n *Node) EffectiveEndpoint() string {
	return cmp.Or(n.Endpoint, n.Address())
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

	if slices.Contains(n.Classes, "") {
		return errors.New("node 'classes' can't contain an empty class")
	}

	return nil
}
