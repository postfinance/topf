// Copyright 2026 PostFinance AG
// SPDX-License-Identifier: MIT

package etcdbackup

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/postfinance/topf/pkg/config"
)

// fakeS3 is a minimal in-memory S3 server covering the operations the store
// uses: PutObject, GetObject and ListObjectsV2.
type fakeS3 struct {
	mu      sync.Mutex
	objects map[string][]byte // key -> content
	mtimes  map[string]time.Time
}

func newFakeS3() *fakeS3 {
	return &fakeS3{
		objects: make(map[string][]byte),
		mtimes:  make(map[string]time.Time),
	}
}

type listEntry struct {
	Key          string `xml:"Key"`
	LastModified string `xml:"LastModified"`
	ETag         string `xml:"ETag"`
	Size         int64  `xml:"Size"`
}

type listResult struct {
	XMLName     xml.Name    `xml:"ListBucketResult"`
	Name        string      `xml:"Name"`
	Prefix      string      `xml:"Prefix"`
	KeyCount    int         `xml:"KeyCount"`
	IsTruncated bool        `xml:"IsTruncated"`
	Contents    []listEntry `xml:"Contents"`
}

func (f *fakeS3) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()

	// path style: /<bucket>/<key...>
	bucket, key, _ := strings.Cut(strings.TrimPrefix(r.URL.Path, "/"), "/")

	switch {
	case r.Method == http.MethodPut && key != "":
		var buf bytes.Buffer
		if _, err := buf.ReadFrom(r.Body); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		content := buf.Bytes()

		// over plain HTTP the client signs with the streaming signature,
		// wrapping the payload in aws-chunked encoding
		if strings.HasPrefix(r.Header.Get("X-Amz-Content-Sha256"), "STREAMING-") {
			decoded, err := decodeAWSChunked(content)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			content = decoded
		}

		f.objects[key] = content
		f.mtimes[key] = time.Now().UTC()
		w.Header().Set("ETag", `"fake"`)
		w.WriteHeader(http.StatusOK)

	case r.Method == http.MethodGet && key != "":
		content, ok := f.objects[key]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintf(w, `<Error><Code>NoSuchKey</Code><Key>%s</Key></Error>`, key)

			return
		}

		w.Header().Set("Content-Length", fmt.Sprint(len(content)))
		w.Header().Set("Last-Modified", f.mtimes[key].Format(http.TimeFormat))
		w.WriteHeader(http.StatusOK)
		w.Write(content) //nolint:errcheck // test server

	case r.Method == http.MethodHead && key != "":
		content, ok := f.objects[key]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Length", fmt.Sprint(len(content)))
		w.Header().Set("Last-Modified", f.mtimes[key].Format(http.TimeFormat))
		w.WriteHeader(http.StatusOK)

	case r.Method == http.MethodGet && key == "":
		prefix := r.URL.Query().Get("prefix")

		result := listResult{
			Name:   bucket,
			Prefix: prefix,
		}

		keys := make([]string, 0, len(f.objects))
		for k := range f.objects {
			if strings.HasPrefix(k, prefix) {
				keys = append(keys, k)
			}
		}

		sort.Strings(keys)

		for _, k := range keys {
			result.Contents = append(result.Contents, listEntry{
				Key:          k,
				LastModified: f.mtimes[k].Format("2006-01-02T15:04:05.000Z"),
				ETag:         `"fake"`,
				Size:         int64(len(f.objects[k])),
			})
		}

		result.KeyCount = len(result.Contents)

		w.Header().Set("Content-Type", "application/xml")
		out, _ := xml.Marshal(result)
		w.Write([]byte(xml.Header)) //nolint:errcheck // test server
		w.Write(out)                //nolint:errcheck // test server

	default:
		w.WriteHeader(http.StatusNotImplemented)
	}
}

// decodeAWSChunked unwraps aws-chunked encoding: repeated
// "<hex-size>[;chunk-signature=...]\r\n<data>\r\n" chunks terminated by a
// zero-size chunk (optionally followed by trailers).
func decodeAWSChunked(body []byte) ([]byte, error) {
	var out bytes.Buffer

	rest := body

	for {
		header, tail, ok := bytes.Cut(rest, []byte("\r\n"))
		if !ok {
			return nil, fmt.Errorf("aws-chunked: missing chunk header terminator")
		}

		sizeHex, _, _ := strings.Cut(string(header), ";")

		size, err := strconv.ParseInt(sizeHex, 16, 64)
		if err != nil {
			return nil, fmt.Errorf("aws-chunked: bad chunk size %q: %w", sizeHex, err)
		}

		if size == 0 {
			return out.Bytes(), nil
		}

		if int64(len(tail)) < size+2 {
			return nil, fmt.Errorf("aws-chunked: truncated chunk")
		}

		out.Write(tail[:size])
		rest = tail[size+2:] // skip trailing \r\n
	}
}

func newTestS3Store(t *testing.T) (*S3Store, *fakeS3) {
	t.Helper()

	fake := newFakeS3()
	server := httptest.NewServer(fake)
	t.Cleanup(server.Close)

	parsed, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}

	store, err := NewS3Store(&config.S3Config{
		Endpoint:        parsed.Host,
		Bucket:          "backups",
		Region:          "us-east-1",
		Insecure:        true,
		AccessKeyID:     "test",
		SecretAccessKey: "test",
	}, "mycluster")
	if err != nil {
		t.Fatal(err)
	}

	return store, fake
}

func TestS3Store(t *testing.T) {
	t.Run("store uploads snapshot and manifest under the cluster prefix", func(t *testing.T) {
		store, fake := newTestS3Store(t)

		data := snapshotBytes(t, 544)
		staged := stageSnapshot(t, data)

		manifest := &Manifest{
			Name:      "mycluster-20260823T000000Z.snapshot",
			CreatedAt: time.Date(2026, 8, 23, 0, 0, 0, 0, time.UTC),
			Size:      544,
		}

		if err := store.Store(t.Context(), manifest, staged); err != nil {
			t.Fatal(err)
		}

		uploaded, ok := fake.objects["mycluster/"+manifest.Name]
		if !ok {
			t.Fatalf("snapshot object missing, have keys: %v", objectKeys(fake))
		}

		if !bytes.Equal(uploaded, data) {
			t.Error("uploaded snapshot differs from staged data")
		}

		if _, ok := fake.objects["mycluster/"+manifest.Name+ManifestSuffix]; !ok {
			t.Error("manifest object missing")
		}
	})

	t.Run("list returns manifests newest first with listing fallback", func(t *testing.T) {
		store, fake := newTestS3Store(t)

		older := &Manifest{
			Name:      "mycluster-20260820T000000Z.snapshot",
			CreatedAt: time.Date(2026, 8, 20, 0, 0, 0, 0, time.UTC),
			Size:      544,
		}
		if err := store.Store(t.Context(), older, stageSnapshot(t, snapshotBytes(t, 544))); err != nil {
			t.Fatal(err)
		}

		newer := &Manifest{
			Name:      "mycluster-20260823T000000Z.snapshot",
			CreatedAt: time.Date(2026, 8, 23, 0, 0, 0, 0, time.UTC),
			Size:      544,
		}
		if err := store.Store(t.Context(), newer, stageSnapshot(t, snapshotBytes(t, 544))); err != nil {
			t.Fatal(err)
		}

		// bare snapshot object without a manifest sidecar
		fake.objects["mycluster/imported.snapshot"] = snapshotBytes(t, 544)
		fake.mtimes["mycluster/imported.snapshot"] = time.Date(2026, 8, 24, 0, 0, 0, 0, time.UTC)

		// objects outside the prefix must be ignored
		fake.objects["othercluster/other.snapshot"] = snapshotBytes(t, 544)
		fake.mtimes["othercluster/other.snapshot"] = time.Now().UTC()

		manifests, err := store.List(t.Context())
		if err != nil {
			t.Fatal(err)
		}

		if len(manifests) != 3 {
			t.Fatalf("expected 3 backups, got %d: %+v", len(manifests), manifests)
		}

		if manifests[0].Name != "imported.snapshot" {
			t.Errorf("expected listing fallback entry first, got %s", manifests[0].Name)
		}

		if manifests[0].Size != 544 {
			t.Errorf("listing fallback size = %d, want 544", manifests[0].Size)
		}

		if manifests[1].Name != newer.Name || manifests[2].Name != older.Name {
			t.Errorf("backups not sorted newest first: %s, %s", manifests[1].Name, manifests[2].Name)
		}
	})

	t.Run("list of empty prefix means no backups", func(t *testing.T) {
		store, _ := newTestS3Store(t)

		manifests, err := store.List(t.Context())
		if err != nil {
			t.Fatal(err)
		}

		if len(manifests) != 0 {
			t.Errorf("expected no backups, got %d", len(manifests))
		}
	})
}

func TestNewS3Store(t *testing.T) {
	tests := []struct {
		name         string
		cfg          config.S3Config
		clusterName  string
		wantLocation string
	}{
		{
			name:         "prefix defaults to cluster name",
			cfg:          config.S3Config{Endpoint: "s3.example.com", Bucket: "b"},
			clusterName:  "mycluster",
			wantLocation: "s3://b/mycluster/",
		},
		{
			name:         "explicit prefix is trimmed and used",
			cfg:          config.S3Config{Endpoint: "s3.example.com", Bucket: "b", Prefix: "/team/talos/"},
			clusterName:  "mycluster",
			wantLocation: "s3://b/team/talos/",
		},
		{
			name:         "scheme in endpoint is accepted",
			cfg:          config.S3Config{Endpoint: "https://s3.example.com:9000", Bucket: "b"},
			clusterName:  "c",
			wantLocation: "s3://b/c/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store, err := NewS3Store(&tt.cfg, tt.clusterName)
			if err != nil {
				t.Fatal(err)
			}

			if store.Location() != tt.wantLocation {
				t.Errorf("Location() = %s, want %s", store.Location(), tt.wantLocation)
			}
		})
	}
}

func objectKeys(f *fakeS3) []string {
	keys := make([]string, 0, len(f.objects))
	for k := range f.objects {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	return keys
}
