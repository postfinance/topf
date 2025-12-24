package topf

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/siderolabs/talos/pkg/machinery/client"
	"github.com/siderolabs/talos/pkg/machinery/config/bundle"
	"github.com/siderolabs/talos/pkg/machinery/config/generate"
	"github.com/siderolabs/talos/pkg/machinery/config/generate/secrets"
)

// Client returns a talos API client, either insecure (if maintenance
// mode) or authenticated (if bootstrapped already) by probing the node endpoint
// for mTLS.
func (n *Node) Client(ctx context.Context) (*client.Client, error) {
	mTLSRequired, err := checkForMTLS(net.JoinHostPort(n.Node.Endpoint(), "50000"))
	if err != nil {
		return nil, err
	}

	if mTLSRequired {
		secrets, err := n.t.Secrets()
		if err != nil {
			return nil, err
		}

		return createAuthenticatedClient(ctx, secrets, n.t.Config().ClusterName, n.Node.Endpoint())
	}

	return createInsecureClient(ctx, n.Node.Endpoint())
}

// createAuthenticatedClient creates a talos client using the given secrets bundle
func createAuthenticatedClient(ctx context.Context, secretsBundle *secrets.Bundle, clusterName string, endpoints ...string) (*client.Client, error) {
	// Generate config bundle from secrets
	configBundleOpts := []bundle.Option{
		bundle.WithVerbose(false), // prevent printing "generating PKI and tokens"
		bundle.WithInputOptions(
			&bundle.InputOptions{
				ClusterName: clusterName,
				Endpoint:    "", // endpoint will be set from client options
				GenOptions: []generate.Option{
					generate.WithSecretsBundle(secretsBundle),
				},
			},
		),
	}

	configBundle, err := bundle.NewBundle(configBundleOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to generate config bundle: %w", err)
	}

	// Create client from talos config
	talosConfig := configBundle.TalosCfg

	c, err := client.New(ctx,
		client.WithConfig(talosConfig),
		client.WithEndpoints(endpoints...),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create talos client: %w", err)
	}

	return c, nil
}

// createInsecureClient creates a talos client that skips TLS verification. Only
// useful for maintenance mode nodes.
func createInsecureClient(ctx context.Context, endpoint string) (*client.Client, error) {
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true, //nolint:gosec // Required for Talos maintenance mode
	}

	c, err := client.New(ctx,
		client.WithTLSConfig(tlsConfig),
		client.WithEndpoints(endpoint),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create insecure talos client: %w", err)
	}

	return c, nil
}

// checkForMTLS checks if the given endpoint requires mTLS by attempting a TLS
// connection and seeing if a client certificate is requested.
func checkForMTLS(endpoint string) (bool, error) {
	certRequested := false

	tlsConfig := &tls.Config{
		InsecureSkipVerify: true, //nolint:gosec // Can't know which PKI is used
		GetClientCertificate: func(_ *tls.CertificateRequestInfo) (*tls.Certificate, error) {
			certRequested = true
			return nil, errors.New("aborting because client certificate requested")
		},
	}

	// Create dialer with context
	dialer := &net.Dialer{
		Timeout: 3 * time.Second,
	}

	// Attempt TLS connection
	conn, err := tls.DialWithDialer(dialer, "tcp", endpoint, tlsConfig)
	if err != nil {
		// Check if error is due to missing client cert
		// If so, we know mTLS is required
		if certRequested {
			return true, nil
		}
		// Other error - might be network, might be other TLS issue
		return false, fmt.Errorf("connection failed: %w", err)
	}
	defer conn.Close()

	// Connection succeeded
	return certRequested, nil
}
