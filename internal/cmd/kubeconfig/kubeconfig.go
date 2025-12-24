// Package kubeconfig contains the logic to generate a valid kubeconfig file
package kubeconfig

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/postfinance/topf/internal/topf"
	"github.com/siderolabs/talos/pkg/machinery/config/generate/secrets"
	"k8s.io/client-go/tools/clientcmd/api"
)

// Generate creates a kubeconfig with a client certificate using the topf runtime
func Generate(t topf.Topf) (*api.Config, error) {
	secretsBundle, err := t.Secrets()
	if err != nil {
		return nil, err
	}
	// Generate client certificate
	clientCert, clientKey, err := generateClientCertificate(secretsBundle)
	if err != nil {
		return nil, fmt.Errorf("failed to generate client certificate: %w", err)
	}

	clusterName := t.Config().ClusterName

	// Generate kubeconfig
	kubeconfig := &api.Config{
		APIVersion:     "v1",
		Kind:           "Config",
		CurrentContext: "topf@" + clusterName,
		Clusters: map[string]*api.Cluster{
			clusterName: {
				Server:                   t.Config().ClusterEndpoint.String(),
				CertificateAuthorityData: secretsBundle.Certs.K8s.Crt,
			},
		},
		Contexts: map[string]*api.Context{
			"topf@" + clusterName: {
				Cluster:  clusterName,
				AuthInfo: "topf",
			},
		},
		AuthInfos: map[string]*api.AuthInfo{
			"topf": {
				ClientCertificateData: clientCert,
				ClientKeyData:         clientKey,
			},
		},
	}

	return kubeconfig, nil
}

// generateClientCertificate creates a new client certificate signed by the Kubernetes CA
// with 12h validity, system:masters group, and topf username
func generateClientCertificate(secretsBundle *secrets.Bundle) ([]byte, []byte, error) {
	// Generate EC private key
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate private key: %w", err)
	}

	// Create certificate template
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName:   "topf",
			Organization: []string{"system:masters"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(12 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
	}

	// Parse the Kubernetes CA certificate and key
	caCertBlock, _ := pem.Decode(secretsBundle.Certs.K8s.Crt)
	if caCertBlock == nil {
		return nil, nil, errors.New("failed to decode CA certificate")
	}

	caCert, err := x509.ParseCertificate(caCertBlock.Bytes)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse CA certificate: %w", err)
	}

	caKeyBlock, _ := pem.Decode(secretsBundle.Certs.K8s.Key)
	if caKeyBlock == nil {
		return nil, nil, errors.New("failed to decode CA private key")
	}

	var signer crypto.Signer

	switch caKeyBlock.Type {
	case "RSA PRIVATE KEY":
		signer, err = x509.ParsePKCS1PrivateKey(caKeyBlock.Bytes)
	case "EC PRIVATE KEY":
		signer, err = x509.ParseECPrivateKey(caKeyBlock.Bytes)
	default:
		return nil, nil, fmt.Errorf("unsupported key type: %s", caKeyBlock.Type)
	}

	if err != nil {
		return nil, nil, err
	}

	// Create the certificate
	certDER, err := x509.CreateCertificate(rand.Reader, &template, caCert, &privateKey.PublicKey, signer)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create certificate: %w", err)
	}

	// Encode certificate to PEM
	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certDER,
	})

	// Encode private key to PEM
	keyDER, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal private key: %w", err)
	}

	keyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: keyDER,
	})

	return certPEM, keyPEM, nil
}
