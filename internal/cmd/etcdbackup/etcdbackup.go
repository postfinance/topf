// Copyright 2026 PostFinance AG
// SPDX-License-Identifier: MIT

// Package etcdbackup contains the logic to create and list etcd snapshot backups
package etcdbackup

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/postfinance/topf/internal/topf"
	"go.etcd.io/etcd/etcdutl/v3/snapshot"
	"go.uber.org/zap"
	"go.yaml.in/yaml/v4"
)

const (
	// SnapshotSuffix is the file extension of etcd snapshot files
	SnapshotSuffix = ".snapshot"
	// ManifestSuffix is appended to the snapshot file name to form the
	// manifest sidecar file name
	ManifestSuffix = ".meta.yaml"

	// timestampFormat is a compact, filename-safe UTC timestamp
	timestampFormat = "20060102T150405Z"
)

// Manifest describes a single etcd backup. It is stored as a sidecar object
// next to the snapshot so that backups stay self-describing when copied to
// other storage.
type Manifest struct {
	Name              string    `yaml:"name"`
	ClusterName       string    `yaml:"clusterName,omitempty"`
	Node              string    `yaml:"node,omitempty"`
	CreatedAt         time.Time `yaml:"createdAt"`
	TalosVersion      string    `yaml:"talosVersion,omitempty"`
	KubernetesVersion string    `yaml:"kubernetesVersion,omitempty"`
	Size              int64     `yaml:"size"`
	SHA256            string    `yaml:"sha256,omitempty"`
	EtcdRevision      int64     `yaml:"etcdRevision,omitempty"`
	EtcdTotalKeys     int       `yaml:"etcdTotalKeys,omitempty"`
	EtcdHash          string    `yaml:"etcdHash,omitempty"`
}

// Store persists verified etcd snapshots and their manifests
type Store interface {
	// Store persists the verified snapshot file at snapshotPath together
	// with its manifest. The file at snapshotPath is consumed:
	// implementations may move it.
	Store(ctx context.Context, manifest *Manifest, snapshotPath string) error

	// List returns the manifests of all stored backups, newest first.
	// Snapshots without a manifest are listed with the information the
	// storage itself can provide.
	List(ctx context.Context) ([]Manifest, error)

	// Location returns a human-readable description of the storage location
	Location() string
}

// Create streams an etcd snapshot from the first reachable control-plane
// node into a local staging file, verifies its integrity offline and hands
// it to the store together with its manifest.
func Create(ctx context.Context, t topf.Topf, store Store) (*Manifest, error) {
	node, err := t.ControlPlaneNode(ctx)
	if err != nil {
		return nil, err
	}

	t.Logger().With(node.Attrs()).Info("streaming etcd snapshot")

	stream, err := node.EtcdSnapshot(ctx)
	if err != nil {
		return nil, err
	}
	defer stream.Close()

	stagingDir, err := os.MkdirTemp("", "topf-etcd-backup-")
	if err != nil {
		return nil, fmt.Errorf("failed to create staging directory: %w", err)
	}

	keepStaging := false

	defer func() {
		if !keepStaging {
			os.RemoveAll(stagingDir)
		}
	}()

	createdAt := time.Now().UTC()

	cfg := t.Config()
	if strings.ContainsAny(cfg.ClusterName, `/\`) {
		return nil, fmt.Errorf("cluster name %q contains a path separator and can't be used in a backup name", cfg.ClusterName)
	}

	name := cfg.ClusterName + "-" + createdAt.Format(timestampFormat) + SnapshotSuffix
	stagingPath := filepath.Join(stagingDir, name)

	size, sum, err := writeSnapshot(stagingPath, stream)
	if err != nil {
		return nil, err
	}

	status, err := verifySnapshot(stagingPath)
	if err != nil {
		keepStaging = true

		return nil, fmt.Errorf("integrity check of %s failed (file kept for inspection): %w", stagingPath, err)
	}

	manifest := &Manifest{
		Name:              name,
		ClusterName:       cfg.ClusterName,
		Node:              node.Node.Host,
		CreatedAt:         createdAt,
		TalosVersion:      node.RunningVersion(),
		KubernetesVersion: cfg.KubernetesVersion,
		Size:              size,
		SHA256:            sum,
		EtcdRevision:      status.Revision,
		EtcdTotalKeys:     status.TotalKey,
		EtcdHash:          fmt.Sprintf("%08x", status.Hash),
	}

	if err := store.Store(ctx, manifest, stagingPath); err != nil {
		return nil, err
	}

	return manifest, nil
}

// writeSnapshot streams r into path via a temporary ".part" file, fsyncs and
// atomically renames it into place. It returns the snapshot size and its
// sha256 checksum.
func writeSnapshot(path string, r io.Reader) (size int64, sum string, err error) {
	partPath := path + ".part"

	defer os.RemoveAll(partPath)

	//nolint:gosec // the path is derived from the user-configured backups directory
	dest, err := os.OpenFile(partPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return 0, "", fmt.Errorf("failed to create temporary file %s: %w", partPath, err)
	}
	defer dest.Close()

	hasher := sha256.New()

	size, err = io.Copy(io.MultiWriter(dest, hasher), r)
	if err != nil {
		return 0, "", fmt.Errorf("failed to write snapshot: %w", err)
	}

	if err = dest.Sync(); err != nil {
		return 0, "", fmt.Errorf("failed to fsync snapshot: %w", err)
	}

	// Snapshots streamed from the etcd maintenance API carry a sha256
	// integrity trailer, making the total size 512*x + sha256.Size
	// (see etcd v3_snapshot.go). Anything else means the stream is
	// truncated or not an etcd snapshot.
	if size%512 != sha256.Size {
		return 0, "", fmt.Errorf("etcd snapshot has invalid size %d: incomplete or missing sha256 trailer", size)
	}

	if err = dest.Close(); err != nil {
		return 0, "", fmt.Errorf("failed to close snapshot file: %w", err)
	}

	if err = os.Rename(partPath, path); err != nil {
		return 0, "", fmt.Errorf("failed to rename snapshot into place: %w", err)
	}

	return size, hex.EncodeToString(hasher.Sum(nil)), nil
}

// verifySnapshot performs an offline integrity check of the snapshot file
// (sha256 integrity trailer, bbolt structure and etcd snapshot hash walk)
// and returns the snapshot status containing hash, revision, total keys and
// total size.
func verifySnapshot(path string) (snapshot.Status, error) {
	if err := verifyTrailer(path); err != nil {
		return snapshot.Status{}, err
	}

	return snapshot.NewV3(zap.NewNop()).Status(path)
}

// verifyTrailer checks the sha256 integrity trailer appended by the etcd
// maintenance API: the last sha256.Size bytes must equal the sha256 sum of
// everything before them. This is the same check the restore path performs
// (see etcdutl copyAndVerifyDB), so a backup that passes here will not be
// rejected at disaster-recovery time.
func verifyTrailer(path string) error {
	//nolint:gosec // the path points at the staging file this process created
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return err
	}

	hasher := sha256.New()
	if _, err := io.CopyN(hasher, f, info.Size()-sha256.Size); err != nil {
		return fmt.Errorf("failed to hash snapshot: %w", err)
	}

	trailer := make([]byte, sha256.Size)
	if _, err := io.ReadFull(f, trailer); err != nil {
		return fmt.Errorf("failed to read sha256 trailer: %w", err)
	}

	if sum := hasher.Sum(nil); !bytes.Equal(sum, trailer) {
		return fmt.Errorf("sha256 trailer mismatch (expected %s, got %s): snapshot corrupted in transit or on disk",
			hex.EncodeToString(trailer), hex.EncodeToString(sum))
	}

	return nil
}

// sortManifests orders manifests newest first
func sortManifests(manifests []Manifest) {
	slices.SortFunc(manifests, func(a, b Manifest) int {
		return b.CreatedAt.Compare(a.CreatedAt)
	})
}

func marshalManifest(manifest *Manifest) ([]byte, error) {
	content, err := yaml.Marshal(manifest)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal backup manifest: %w", err)
	}

	return content, nil
}

func unmarshalManifest(content []byte) (*Manifest, error) {
	manifest := &Manifest{}
	if err := yaml.Unmarshal(content, manifest); err != nil {
		return nil, fmt.Errorf("failed to parse backup manifest: %w", err)
	}

	return manifest, nil
}
