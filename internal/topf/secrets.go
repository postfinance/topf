// Copyright 2026 PostFinance AG
// SPDX-License-Identifier: MIT

package topf

import (
	"encoding/base64"
	"errors"
	"time"

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

	provider := t.GetSecretsProvider()

	// use a clock skewed slightly to the past to ensure generated certs are valid even
	// if there is some time drift between the talos node and the machine running topf
	skewedClock := secrets.NewFixedClock(time.Now().Add(-time.Second * 1))

	bundle, err := providers.LoadSecretsBundle(provider, t.ClusterName)
	if errors.Is(err, providers.ErrSecretsNotFound) {
		t.logger.Warn("generating new secrets.yaml", "cluster", t.ClusterName)

		bundle, err = secrets.NewBundle(skewedClock, nil)
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
	} else if err != nil {
		return nil, err
	}

	bundle.Clock = skewedClock

	t.secretsBundle = bundle

	t.maskedPrinter.AddSecrets(collectSecrets(bundle))

	return bundle, nil
}

// collectSecrets extracts all sensitive strings from the secrets bundle.
// assumes a validated secrets bundle, with all fields present
func collectSecrets(bundle *secrets.Bundle) []string {
	return []string{
		base64.StdEncoding.EncodeToString(bundle.Certs.Etcd.Key),
		base64.StdEncoding.EncodeToString(bundle.Certs.K8s.Key),
		base64.StdEncoding.EncodeToString(bundle.Certs.K8sAggregator.Key),
		base64.StdEncoding.EncodeToString(bundle.Certs.K8sServiceAccount.Key),
		base64.StdEncoding.EncodeToString(bundle.Certs.OS.Key),
		bundle.Secrets.BootstrapToken,
		bundle.Secrets.AESCBCEncryptionSecret,
		bundle.Secrets.SecretboxEncryptionSecret,
		bundle.TrustdInfo.Token,
		bundle.Cluster.Secret,
	}
}
