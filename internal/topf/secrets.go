// Copyright 2026 PostFinance AG
// SPDX-License-Identifier: MIT

package topf

import (
	"errors"
	"fmt"
	"time"

	"github.com/postfinance/topf/internal/interactive"
	"github.com/postfinance/topf/pkg/providers"
	"github.com/siderolabs/talos/pkg/machinery/config/generate/secrets"
	"go.yaml.in/yaml/v4"
)

// Secrets returns the secrets bundle, loading it from the secrets provider if not already loaded
func (t *topf) Secrets() (*secrets.Bundle, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.secretsBundle != nil {
		return t.secretsBundle, nil
	}

	provider := t.GetSecretsProvider(t.decryptCache)

	// use a clock skewed slightly to the past to ensure generated certs are valid even
	// if there is some time drift between the talos node and the machine running topf
	skewedClock := secrets.NewFixedClock(time.Now().Add(-time.Second * 1))

	bundle, err := providers.LoadSecretsBundle(provider, t.ClusterName)
	if errors.Is(err, providers.ErrSecretsNotFound) {
		return t.generateAndStoreSecrets(provider, skewedClock)
	} else if err != nil {
		return nil, err
	}

	bundle.Clock = skewedClock

	t.secretsBundle = bundle

	t.AddSecretsToMask(collectSecrets(bundle))

	return bundle, nil
}

func (t *topf) generateAndStoreSecrets(provider providers.SecretsProvider, clock secrets.Clock) (*secrets.Bundle, error) {
	if t.Confirm() {
		if interactive.ConfirmPrompt(fmt.Sprintf("No secrets.yaml found for cluster %s. Generate a new one?", t.ClusterName)) == 'n' {
			return nil, fmt.Errorf("secrets.yaml not found for cluster %s", t.ClusterName)
		}
	}

	t.logger.Warn("generating new secrets.yaml", "cluster", t.ClusterName)

	bundle, err := secrets.NewBundle(clock, nil)
	if err != nil {
		return nil, err
	}

	bytes, err := yaml.Marshal(bundle)
	if err != nil {
		return nil, err
	}

	if err := provider.Put(t.ClusterName, bytes); err != nil {
		return nil, err
	}

	t.logger.Info("secrets stored")

	bundle.Clock = clock

	t.secretsBundle = bundle

	t.AddSecretsToMask(collectSecrets(bundle))

	return bundle, nil
}

func collectSecrets(bundle *secrets.Bundle) []string {
	s := make([]string, 0, 14)

	s = append(s, encodingVariations(bundle.Certs.Etcd.Key)...)
	s = append(s, encodingVariations(bundle.Certs.K8s.Key)...)
	s = append(s, encodingVariations(bundle.Certs.K8sAggregator.Key)...)
	s = append(s, encodingVariations(bundle.Certs.K8sServiceAccount.Key)...)
	s = append(s, encodingVariations(bundle.Certs.OS.Key)...)

	s = append(s,
		bundle.Secrets.BootstrapToken,
		bundle.Secrets.AESCBCEncryptionSecret,
		bundle.Secrets.SecretboxEncryptionSecret,
		bundle.TrustdInfo.Token,
		bundle.Cluster.Secret,
	)

	return s
}
