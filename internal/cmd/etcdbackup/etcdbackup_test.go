// Copyright 2026 PostFinance AG
// SPDX-License-Identifier: MIT

package etcdbackup

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// snapshotBytes returns n bytes shaped like an etcd snapshot: a random
// payload followed by its sha256 integrity trailer, so that the total size
// is 512*x + 32 and the trailer verifies.
func snapshotBytes(t *testing.T, n int) []byte {
	t.Helper()

	if n%512 != sha256.Size {
		t.Fatalf("snapshot size %d does not leave room for the trailer", n)
	}

	data := make([]byte, n)
	if _, err := rand.Read(data[:n-sha256.Size]); err != nil {
		t.Fatal(err)
	}

	trailer := sha256.Sum256(data[:n-sha256.Size])
	copy(data[n-sha256.Size:], trailer[:])

	return data
}

// stageSnapshot writes data to a staging file the way Create does
func stageSnapshot(t *testing.T, data []byte) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "staged.snapshot")
	if _, _, err := writeSnapshot(path, bytes.NewReader(data)); err != nil {
		t.Fatal(err)
	}

	return path
}

func TestWriteSnapshot(t *testing.T) {
	t.Run("valid size is written atomically", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "test.snapshot")
		data := snapshotBytes(t, 2*512+sha256.Size)

		size, sum, err := writeSnapshot(path, bytes.NewReader(data))
		if err != nil {
			t.Fatal(err)
		}

		if size != int64(len(data)) {
			t.Errorf("size = %d, want %d", size, len(data))
		}

		wantSum := sha256.Sum256(data)
		if sum != hex.EncodeToString(wantSum[:]) {
			t.Errorf("sha256 = %s, want %s", sum, hex.EncodeToString(wantSum[:]))
		}

		written, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}

		if !bytes.Equal(written, data) {
			t.Error("written file differs from input")
		}

		if _, err := os.Stat(path + ".part"); !os.IsNotExist(err) {
			t.Error("temporary .part file was left behind")
		}
	})

	t.Run("invalid size is rejected and cleaned up", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "test.snapshot")

		junk := make([]byte, 1000)

		_, _, err := writeSnapshot(path, bytes.NewReader(junk))
		if err == nil {
			t.Fatal("expected error for snapshot without sha256 trailer")
		}

		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Error("invalid snapshot was renamed into place")
		}

		if _, err := os.Stat(path + ".part"); !os.IsNotExist(err) {
			t.Error("temporary .part file was left behind")
		}
	})
}

func TestVerifyTrailer(t *testing.T) {
	t.Run("valid trailer passes", func(t *testing.T) {
		path := stageSnapshot(t, snapshotBytes(t, 544))

		if err := verifyTrailer(path); err != nil {
			t.Errorf("expected valid trailer to verify, got: %v", err)
		}
	})

	t.Run("corrupted payload is detected", func(t *testing.T) {
		data := snapshotBytes(t, 544)
		data[100] ^= 0xff // flip one payload byte after the trailer was computed

		path := filepath.Join(t.TempDir(), "corrupt.snapshot")
		if err := os.WriteFile(path, data, 0o600); err != nil {
			t.Fatal(err)
		}

		if err := verifyTrailer(path); err == nil {
			t.Error("expected trailer mismatch for corrupted payload")
		}
	})
}

func TestVerifySnapshot(t *testing.T) {
	t.Run("garbage is rejected", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "garbage.snapshot")

		// valid trailer, but not a bbolt database
		if err := os.WriteFile(path, snapshotBytes(t, 512+sha256.Size), 0o600); err != nil {
			t.Fatal(err)
		}

		if _, err := verifySnapshot(path); err == nil {
			t.Error("expected verification to fail for non-bbolt data")
		}
	})

	t.Run("missing file is an error", func(t *testing.T) {
		if _, err := verifySnapshot(filepath.Join(t.TempDir(), "missing.snapshot")); err == nil {
			t.Error("expected verification to fail for missing file")
		}
	})
}

func TestManifestRoundTrip(t *testing.T) {
	want := &Manifest{
		Name:              "mycluster-20260823T190000Z.snapshot",
		ClusterName:       "mycluster",
		Node:              "controlplane-01",
		CreatedAt:         time.Date(2026, 8, 23, 19, 0, 0, 0, time.UTC),
		TalosVersion:      "1.13.4",
		KubernetesVersion: "1.35.3",
		Size:              41943072,
		SHA256:            "abc123",
		EtcdRevision:      123456,
		EtcdTotalKeys:     1200,
		EtcdHash:          "8f2a0000",
	}

	content, err := marshalManifest(want)
	if err != nil {
		t.Fatal(err)
	}

	got, err := unmarshalManifest(content)
	if err != nil {
		t.Fatal(err)
	}

	if *got != *want {
		t.Errorf("manifest round trip mismatch:\ngot:  %+v\nwant: %+v", got, want)
	}
}
