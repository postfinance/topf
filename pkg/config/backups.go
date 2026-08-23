// Copyright 2026 PostFinance AG
// SPDX-License-Identifier: MIT

package config

import (
	"errors"

	"go.yaml.in/yaml/v4"
)

// BackupsConfig configures the storage for etcd snapshot backups. Exactly
// one storage backend must be set.
type BackupsConfig struct {
	// S3 stores backups in an S3-compatible bucket
	S3 *S3Config `yaml:"s3,omitempty"`
}

// UnmarshalYAML implements yaml.Unmarshaler and performs additional validation
func (b *BackupsConfig) UnmarshalYAML(yamlNode *yaml.Node) error {
	type raw BackupsConfig

	if err := yamlNode.Decode((*raw)(b)); err != nil {
		return err
	}

	if b.S3 == nil {
		return errors.New("backups requires a storage backend: set 's3'")
	}

	return nil
}

// S3Config describes an S3-compatible object storage location
type S3Config struct {
	// Endpoint is the S3 host, e.g. "s3.amazonaws.com" or
	// "minio.example.com:9000". A scheme may be included; "http://" implies
	// Insecure.
	Endpoint string `yaml:"endpoint"`

	// Name of the bucket
	Bucket string `yaml:"bucket"`

	// Prefix is the object key prefix under which objects are stored.
	// Defaults to "<clusterName>/".
	Prefix string `yaml:"prefix,omitempty"`

	// Region is the bucket region (optional)
	Region string `yaml:"region,omitempty"`

	// Insecure disables TLS for the endpoint
	Insecure bool `yaml:"insecure,omitempty"`

	// AccessKeyID and SecretAccessKey static credentials, not recommended
	// to set here for production use. Instead use the standard credential chain.
	AccessKeyID     string `yaml:"accessKeyId,omitempty"`
	SecretAccessKey string `yaml:"secretAccessKey,omitempty"`
}

// UnmarshalYAML implements yaml.Unmarshaler and performs additional validation
func (s *S3Config) UnmarshalYAML(yamlNode *yaml.Node) error {
	type raw S3Config

	if err := yamlNode.Decode((*raw)(s)); err != nil {
		return err
	}

	if s.Endpoint == "" {
		return errors.New("s3 'endpoint' can't be empty")
	}

	if s.Bucket == "" {
		return errors.New("s3 'bucket' can't be empty")
	}

	return nil
}
