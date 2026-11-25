// Copyright 2026 PostFinance AG
// SPDX-License-Identifier: MIT

// Package talosconfig contains the logic to generate talosconfig file
package talosconfig

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/postfinance/topf/internal/topf"
	"github.com/postfinance/topf/pkg/config"
	clientconfig "github.com/siderolabs/talos/pkg/machinery/client/config"
	"github.com/siderolabs/talos/pkg/machinery/constants"
	"github.com/siderolabs/talos/pkg/machinery/role"
)

// DefaultValidity is the default validity of the client certificate,
// matching the Talos default.
const DefaultValidity = constants.TalosAPIDefaultCertificateValidityDuration

// DefaultRole is the default Talos role of the client certificate.
const DefaultRole = role.Admin

// Generate returns a talosconfig with a client certificate for the given
// roles, valid for the given duration.
func Generate(t topf.Topf, roles []string, validity time.Duration) ([]byte, error) {
	if validity <= 0 {
		return nil, fmt.Errorf("validity must be positive, got %s", validity)
	}

	roleSet, unknownRoles := role.Parse(roles)
	if len(unknownRoles) > 0 {
		return nil, fmt.Errorf("unknown roles: %s", strings.Join(unknownRoles, ", "))
	}

	if roleSet.Empty() {
		return nil, errors.New("at least one role is required")
	}

	secretsBundle, err := t.Secrets()
	if err != nil {
		return nil, fmt.Errorf("failed to load secrets bundle: %w", err)
	}

	clientCert, err := secretsBundle.GenerateTalosAPIClientCertificateWithTTL(roleSet, validity)
	if err != nil {
		return nil, fmt.Errorf("failed to generate client certificate: %w", err)
	}

	clusterName := t.Config().ClusterName
	talosCfg := clientconfig.NewConfig(clusterName, nil, secretsBundle.Certs.OS.Crt, clientCert)
	setEndpointsAndNodes(talosCfg.Contexts[clusterName], t.Config().Nodes)

	return talosCfg.Bytes()
}

func setEndpointsAndNodes(talosCtx *clientconfig.Context, nodes []config.Node) {
	// For single-node clusters, add the single node as both endpoint and node
	if len(nodes) == 1 {
		endpoint := nodes[0].Endpoint()
		talosCtx.Endpoints = []string{endpoint}
		talosCtx.Nodes = []string{endpoint}

		return
	}

	// For multi-node clusters:
	// - endpoints: all control-plane nodes
	// - nodes: all nodes
	var (
		endpoints []string
		allNodes  []string
	)

	for _, node := range nodes {
		allNodes = append(allNodes, node.Endpoint())
		if node.Role == config.RoleControlPlane {
			endpoints = append(endpoints, node.Endpoint())
		}
	}

	if len(endpoints) > 0 {
		talosCtx.Endpoints = endpoints
	}

	if len(allNodes) > 0 {
		talosCtx.Nodes = allNodes
	}
}
