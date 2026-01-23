package providers

import (
	"errors"
	"fmt"
	"time"

	"github.com/siderolabs/talos/pkg/machinery/config/generate/secrets"
	"go.yaml.in/yaml/v4"
)

// ErrSecretsNotFound is returned when no secrets.yaml is found for a given cluster
var ErrSecretsNotFound = errors.New("secrets.yaml not found")

// SecretsProvider is an interface for getting and storing talos 'secrets.yaml' files
type SecretsProvider interface {
	// Get return secrets.yaml for the given cluster
	// If there is no secrets.yaml for the given cluster, returns nil
	Get(clusterName string) ([]byte, error)

	// Put stores the secrets.yaml for the given cluster
	Put(clusterName string, bundle []byte) error
}

// LoadSecretsBundle gets and unmarshals secrets from a provider.
func LoadSecretsBundle(provider SecretsProvider, clusterName string) (*secrets.Bundle, error) {
	bytes, err := provider.Get(clusterName)
	if err != nil {
		return nil, err
	}

	if len(bytes) == 0 {
		return nil, fmt.Errorf("%w for cluster %s", ErrSecretsNotFound, clusterName)
	}

	bundle := &secrets.Bundle{Clock: nowFunc(time.Now)}
	if err := yaml.Unmarshal(bytes, &bundle); err != nil {
		return nil, fmt.Errorf("failed to unmarshal secrets: %w", err)
	}

	if err := bundle.Validate(); err != nil {
		return nil, fmt.Errorf("invalid secrets bundle: %w", err)
	}

	return bundle, nil
}

// helper type, because secrets.Bundles needs a Now() time.Time interface and we
// don't want to include third-party libaries just for that
type nowFunc func() time.Time

func (f nowFunc) Now() time.Time { return f() }
