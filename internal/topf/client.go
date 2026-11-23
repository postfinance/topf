// Copyright 2026 PostFinance AG
// SPDX-License-Identifier: MIT

package topf

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"time"

	"github.com/siderolabs/talos/pkg/machinery/client"
	clientconfig "github.com/siderolabs/talos/pkg/machinery/client/config"
	"github.com/siderolabs/talos/pkg/machinery/client/dialer"
	"github.com/siderolabs/talos/pkg/machinery/client/resolver"
	"github.com/siderolabs/talos/pkg/machinery/config/bundle"
	"github.com/siderolabs/talos/pkg/machinery/config/generate"
	"github.com/siderolabs/talos/pkg/machinery/constants"
	"google.golang.org/grpc"
)

// Client returns a Talos API client for the node. Nodes in maintenance mode
// get an insecure client, all others an mTLS client which targets the node
// through its endpoint.
func (n *Node) Client(ctx context.Context) (*client.Client, error) {
	endpoint := n.Node.EffectiveEndpoint()

	mTLSRequired, err := checkForMTLS(ctx, endpoint)
	if err != nil {
		return nil, err
	}

	opts := []client.OptionFunc{client.WithDefaultGRPCDialOptions()}

	if mTLSRequired {
		talosConfig, err := n.talosConfig()
		if err != nil {
			return nil, err
		}

		opts = append(opts,
			client.WithConfig(talosConfig),
			client.WithEndpoints(endpoint),
			withNode(n.Node.Address()),
		)
	} else {
		opts = append(opts, client.WithMaintenanceMode(endpoint, nil))
	}

	c, err := client.New(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create talos client: %w", err)
	}

	return c, nil
}

// talosConfig generates a Talos client config from the cluster secrets.
func (n *Node) talosConfig() (*clientconfig.Config, error) {
	secretsBundle, err := n.t.Secrets()
	if err != nil {
		return nil, err
	}

	configBundle, err := bundle.NewBundle(
		bundle.WithVerbose(false),
		bundle.WithInputOptions(&bundle.InputOptions{
			ClusterName: n.t.Config().ClusterName,
			KubeVersion: n.KubernetesVersion(),
			GenOptions:  []generate.Option{generate.WithSecretsBundle(secretsBundle)},
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate config bundle: %w", err)
	}

	return configBundle.TalosCfg, nil
}

// withNode routes every request through the endpoint to the given node.
func withNode(node string) client.OptionFunc {
	return client.WithGRPCDialOptions(
		grpc.WithChainUnaryInterceptor(func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
			return invoker(client.WithNode(ctx, node), method, req, reply, cc, opts...)
		}),
		grpc.WithChainStreamInterceptor(func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
			return streamer(client.WithNode(ctx, node), desc, cc, method, opts...)
		}),
	)
}

// checkForMTLS checks if the given endpoint requires mTLS by attempting a TLS
// connection and seeing if a client certificate is requested.
// It respects HTTPS_PROXY/HTTP_PROXY environment variables.
func checkForMTLS(ctx context.Context, endpoint string) (bool, error) {
	certRequested := false

	tlsConfig := &tls.Config{
		InsecureSkipVerify: true, //nolint:gosec // Can't know which PKI is used
		GetClientCertificate: func(_ *tls.CertificateRequestInfo) (*tls.Certificate, error) {
			certRequested = true
			return nil, errors.New("aborting because client certificate requested")
		},
	}

	dialCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	address := resolver.EnsureEndpointsHavePorts([]string{endpoint}, constants.ApidPort)[0]

	conn, err := dialer.DynamicProxyDialer(dialCtx, address)
	if err != nil {
		return false, fmt.Errorf("connection failed: %w", err)
	}

	tlsConn := tls.Client(conn, tlsConfig)
	defer tlsConn.Close()

	if err := tlsConn.HandshakeContext(dialCtx); err != nil {
		if certRequested {
			return true, nil
		}

		return false, fmt.Errorf("connection failed: %w", err)
	}

	return certRequested, nil
}
