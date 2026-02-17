// Copyright 2026 PostFinance AG
// SPDX-License-Identifier: MIT

package topf

import (
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

	bundle, err := providers.LoadSecretsBundle(provider, t.ClusterName)
	if errors.Is(err, providers.ErrSecretsNotFound) {
		t.logger.Warn("generating new secrets.yaml", "cluster", t.ClusterName)

		// use a clock skewed slightly to the past to ensure generated certs are valid even
		// if there is some time drift between the talos node and the machine running topf
		skewedClock := secrets.NewFixedClock(time.Now().Add(-time.Second * 1))

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

	t.secretsBundle = bundle

	return bundle, nil
}
