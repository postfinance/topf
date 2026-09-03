// Copyright 2026 PostFinance AG
// SPDX-License-Identifier: MIT

package topf

import (
	"encoding/base64"

	"github.com/siderolabs/crypto/x509"
	talosconfig "github.com/siderolabs/talos/pkg/machinery/config"
	configconfig "github.com/siderolabs/talos/pkg/machinery/config/config"
)

type caConfig interface {
	IssuingCA() *x509.PEMEncodedCertificateAndKey
	AcceptedCAs() []*x509.PEMEncodedCertificate
}

// collectCurrentConfigSecrets extracts sensitive strings from a node's current
// machine config so they can be added to the redaction pool. During a key
// rotation the old certs still live on the node; without this they would leak
// through the masked writer.
func collectCurrentConfigSecrets(cfg talosconfig.Config) []string {
	var secrets []string

	if cluster := cfg.Cluster(); cluster != nil {
		collectClusterSecrets(&secrets, cluster)
	}

	collectCAConfigSecrets(&secrets, cfg.K8sAPIServerCAConfig())
	collectCAConfigSecrets(&secrets, cfg.K8sAggregatorCAConfig())
	collectServiceAccountSecrets(&secrets, cfg.K8sServiceAccountConfig())

	if identity := cfg.DiscoveryIdentityConfig(); identity != nil {
		secrets = append(secrets, identity.ClusterSecret())
	}

	if machine := cfg.Machine(); machine != nil {
		collectMachineSecrets(&secrets, machine)
	}

	return secrets
}

func collectClusterSecrets(secrets *[]string, cluster configconfig.ClusterConfig) {
	if t := cluster.Token(); t != nil {
		*secrets = append(*secrets, t.ID()+"."+t.Secret())
	}

	if etcd := cluster.Etcd(); etcd != nil {
		addCAKeyPair(secrets, etcd.CA())
	}

	*secrets = append(*secrets,
		cluster.AESCBCEncryptionSecret(),
		cluster.SecretboxEncryptionSecret(),
	)
}

func collectCAConfigSecrets(secrets *[]string, ca caConfig) {
	if ca == nil {
		return
	}

	addCAKeyPair(secrets, ca.IssuingCA())
	addPEMEncodedCerts(secrets, ca.AcceptedCAs())
}

func collectServiceAccountSecrets(secrets *[]string, sa configconfig.K8sServiceAccountConfig) {
	if sa == nil {
		return
	}

	if key := sa.IssuingKey(); key != nil && len(key.Key) > 0 {
		*secrets = append(*secrets, base64.StdEncoding.EncodeToString(key.Key))
	}

	addPEMEncodedKeys(secrets, sa.AcceptedKeys())
}

func collectMachineSecrets(secrets *[]string, machine configconfig.MachineConfig) {
	if sec := machine.Security(); sec != nil {
		addCAKeyPair(secrets, sec.IssuingCA())
		addPEMEncodedCerts(secrets, sec.AcceptedCAs())
		*secrets = append(*secrets, sec.Token())
	}
}

func addCAKeyPair(secrets *[]string, ca *x509.PEMEncodedCertificateAndKey) {
	if ca == nil {
		return
	}

	if len(ca.Crt) > 0 {
		*secrets = append(*secrets, base64.StdEncoding.EncodeToString(ca.Crt))
	}

	if len(ca.Key) > 0 {
		*secrets = append(*secrets, base64.StdEncoding.EncodeToString(ca.Key))
	}
}

func addPEMEncodedCerts(secrets *[]string, certs []*x509.PEMEncodedCertificate) {
	for _, c := range certs {
		if c != nil && len(c.Crt) > 0 {
			*secrets = append(*secrets, base64.StdEncoding.EncodeToString(c.Crt))
		}
	}
}

func addPEMEncodedKeys(secrets *[]string, keys []*x509.PEMEncodedKey) {
	for _, k := range keys {
		if k != nil && len(k.Key) > 0 {
			*secrets = append(*secrets, base64.StdEncoding.EncodeToString(k.Key))
		}
	}
}
