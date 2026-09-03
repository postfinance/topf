// Copyright 2026 PostFinance AG
// SPDX-License-Identifier: MIT

package topf

import (
	"encoding/base64"
	"strings"

	talosconfig "github.com/siderolabs/talos/pkg/machinery/config"
)

// collectCurrentConfigSecrets extracts sensitive strings from a node's current
// machine config so they can be added to the redaction pool. During a key
// rotation the old keys still live on the node; without this they would leak
// through the masked writer. Public certificates are not redacted.
func collectCurrentConfigSecrets(cfg talosconfig.Config) []string {
	var secrets []string

	if cluster := cfg.Cluster(); cluster != nil {
		if t := cluster.Token(); t != nil {
			secrets = append(secrets, encodingVariations([]byte(t.Secret()))...)
		}

		if etcd := cluster.Etcd(); etcd != nil && etcd.CA() != nil {
			secrets = append(secrets, encodingVariations(etcd.CA().Key)...)
		}

		secrets = append(secrets, encodingVariations([]byte(cluster.AESCBCEncryptionSecret()))...)
		secrets = append(secrets, encodingVariations([]byte(cluster.SecretboxEncryptionSecret()))...)
	}

	if ca := cfg.K8sAPIServerCAConfig(); ca != nil && ca.IssuingCA() != nil {
		secrets = append(secrets, encodingVariations(ca.IssuingCA().Key)...)
	}

	if ca := cfg.K8sAggregatorCAConfig(); ca != nil && ca.IssuingCA() != nil {
		secrets = append(secrets, encodingVariations(ca.IssuingCA().Key)...)
	}

	if sa := cfg.K8sServiceAccountConfig(); sa != nil && sa.IssuingKey() != nil {
		secrets = append(secrets, encodingVariations(sa.IssuingKey().Key)...)
	}

	if identity := cfg.DiscoveryIdentityConfig(); identity != nil {
		secrets = append(secrets, encodingVariations([]byte(identity.ClusterSecret()))...)
	}

	if machine := cfg.Machine(); machine != nil {
		if sec := machine.Security(); sec != nil && sec.IssuingCA() != nil {
			secrets = append(secrets, encodingVariations(sec.IssuingCA().Key)...)
			secrets = append(secrets, encodingVariations([]byte(sec.Token()))...)
		}
	}

	return secrets
}

// encodingVariations returns the base64 form (used by v1alpha1) and each
// non-empty line of the raw value (used by v1.14 multi-doc block scalars).
func encodingVariations(secret []byte) []string {
	if len(secret) == 0 {
		return nil
	}

	ret := []string{base64.StdEncoding.EncodeToString(secret)}

	for _, line := range strings.Split(string(secret), "\n") {
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			ret = append(ret, trimmed)
		}
	}

	return ret
}
