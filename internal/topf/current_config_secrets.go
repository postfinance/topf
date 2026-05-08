// Copyright 2026 PostFinance AG
// SPDX-License-Identifier: MIT

package topf

import (
	"encoding/base64"

	"github.com/siderolabs/crypto/x509"
	talosconfig "github.com/siderolabs/talos/pkg/machinery/config"
)

// collectCurrentConfigSecrets extracts sensitive strings from a node's current
// machine config so they can be added to the redaction pool. During a key
// rotation the old certs still live on the node; without this they would leak
// through the masked writer.
func collectCurrentConfigSecrets(cfg talosconfig.Config) []string {
	var secrets []string

	addCAKeyPair := func(ca *x509.PEMEncodedCertificateAndKey) {
		if ca == nil {
			return
		}

		if len(ca.Crt) > 0 {
			secrets = append(secrets, base64.StdEncoding.EncodeToString(ca.Crt))
		}

		if len(ca.Key) > 0 {
			secrets = append(secrets, base64.StdEncoding.EncodeToString(ca.Key))
		}
	}

	addPEMEncodedCerts := func(certs []*x509.PEMEncodedCertificate) {
		for _, c := range certs {
			if c != nil && len(c.Crt) > 0 {
				secrets = append(secrets, base64.StdEncoding.EncodeToString(c.Crt))
			}
		}
	}

	if cluster := cfg.Cluster(); cluster != nil {
		addCAKeyPair(cluster.IssuingCA())
		addCAKeyPair(cluster.AggregatorCA())
		addPEMEncodedCerts(cluster.AcceptedCAs())

		if sa := cluster.ServiceAccount(); sa != nil && len(sa.Key) > 0 {
			secrets = append(secrets, base64.StdEncoding.EncodeToString(sa.Key))
		}

		if etcd := cluster.Etcd(); etcd != nil {
			addCAKeyPair(etcd.CA())
		}

		if t := cluster.Token(); t != nil {
			secrets = append(secrets, t.ID()+"."+t.Secret())
		}

		secrets = append(secrets,
			cluster.Secret(),
			cluster.AESCBCEncryptionSecret(),
			cluster.SecretboxEncryptionSecret(),
		)
	}

	if machine := cfg.Machine(); machine != nil {
		if sec := machine.Security(); sec != nil {
			addCAKeyPair(sec.IssuingCA())
			addPEMEncodedCerts(sec.AcceptedCAs())
			secrets = append(secrets, sec.Token())
		}
	}

	return secrets
}
