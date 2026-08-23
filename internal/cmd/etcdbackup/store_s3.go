// Copyright 2026 PostFinance AG
// SPDX-License-Identifier: MIT

package etcdbackup

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/url"
	"path"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/postfinance/topf/pkg/config"
)

// S3Store stores backups as objects in an S3-compatible bucket, with the
// manifest as a sidecar object next to each snapshot.
type S3Store struct {
	client *minio.Client
	bucket string
	prefix string
}

// NewS3Store returns a store writing to the bucket described by cfg.
func NewS3Store(cfg *config.S3Config, clusterName string) (*S3Store, error) {
	endpoint := cfg.Endpoint
	secure := !cfg.Insecure

	// accept endpoints with a scheme and derive TLS usage from it
	if strings.Contains(endpoint, "://") {
		parsed, err := url.Parse(endpoint)
		if err != nil {
			return nil, fmt.Errorf("invalid s3 endpoint %q: %w", cfg.Endpoint, err)
		}

		endpoint = parsed.Host
		secure = parsed.Scheme != "http"
	}

	var creds *credentials.Credentials
	if cfg.AccessKeyID != "" || cfg.SecretAccessKey != "" {
		creds = credentials.NewStaticV4(cfg.AccessKeyID, cfg.SecretAccessKey, "")
	} else {
		creds = credentials.NewChainCredentials([]credentials.Provider{
			&credentials.EnvAWS{},
			&credentials.EnvMinio{},
			&credentials.FileAWSCredentials{},
			&credentials.IAM{},
		})
	}

	client, err := minio.New(endpoint, &minio.Options{
		Creds:  creds,
		Secure: secure,
		Region: cfg.Region,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create s3 client: %w", err)
	}

	prefix := cfg.Prefix
	if prefix == "" {
		prefix = clusterName
	}

	if prefix = strings.Trim(prefix, "/"); prefix != "" {
		prefix += "/"
	}

	return &S3Store{
		client: client,
		bucket: cfg.Bucket,
		prefix: prefix,
	}, nil
}

// Location implements Store
func (s *S3Store) Location() string {
	return "s3://" + s.bucket + "/" + s.prefix
}

// Store implements Store by uploading the snapshot and its manifest sidecar
// object. The manifest is uploaded last, so a backup with a manifest is
// always a completely transferred backup.
func (s *S3Store) Store(ctx context.Context, manifest *Manifest, snapshotPath string) error {
	key := s.prefix + manifest.Name

	if _, err := s.client.FPutObject(ctx, s.bucket, key, snapshotPath, minio.PutObjectOptions{
		ContentType: "application/octet-stream",
	}); err != nil {
		return fmt.Errorf("failed to upload snapshot to %s: %w", s.Location(), err)
	}

	content, err := marshalManifest(manifest)
	if err != nil {
		return err
	}

	if _, err := s.client.PutObject(ctx, s.bucket, key+ManifestSuffix, bytes.NewReader(content), int64(len(content)), minio.PutObjectOptions{
		ContentType: "application/yaml",
	}); err != nil {
		return fmt.Errorf("failed to upload backup manifest to %s: %w", s.Location(), err)
	}

	return nil
}

// List implements Store. Snapshot objects without a manifest sidecar are
// listed with the information the object listing itself provides.
func (s *S3Store) List(ctx context.Context) ([]Manifest, error) {
	var manifests []Manifest

	for object := range s.client.ListObjects(ctx, s.bucket, minio.ListObjectsOptions{
		Prefix:    s.prefix,
		Recursive: true,
	}) {
		if object.Err != nil {
			return nil, fmt.Errorf("failed to list backups in %s: %w", s.Location(), object.Err)
		}

		if !strings.HasSuffix(object.Key, SnapshotSuffix) {
			continue
		}

		manifest, err := s.readManifestObject(ctx, object.Key+ManifestSuffix)
		if err != nil {
			// a corrupt or unreadable manifest must surface, not silently
			// degrade into the fallback below
			if minio.ToErrorResponse(err).Code != minio.NoSuchKey {
				return nil, fmt.Errorf("failed to read backup manifest %s: %w", object.Key+ManifestSuffix, err)
			}

			// no manifest sidecar: fall back to what the object listing
			// itself can tell us
			manifest = &Manifest{
				CreatedAt: object.LastModified.UTC(),
				Size:      object.Size,
			}
		}

		// name the entry after the object actually in the bucket, so renamed
		// or copied backup pairs are listed truthfully
		manifest.Name = path.Base(object.Key)

		manifests = append(manifests, *manifest)
	}

	sortManifests(manifests)

	return manifests, nil
}

func (s *S3Store) readManifestObject(ctx context.Context, key string) (*Manifest, error) {
	object, err := s.client.GetObject(ctx, s.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}
	defer object.Close()

	content, err := io.ReadAll(object)
	if err != nil {
		return nil, err
	}

	return unmarshalManifest(content)
}
