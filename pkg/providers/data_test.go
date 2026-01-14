package providers

import (
	"errors"
	"testing"
)

// mockDataProvider is a test implementation of DataProvider
type mockDataProvider struct {
	output []byte
	err    error
}

func (m *mockDataProvider) GetData(_ string) ([]byte, error) {
	return m.output, m.err
}

func TestLoadDataYAML(t *testing.T) {
	tests := []struct {
		name        string
		provider    DataProvider
		clusterName string
		wantErr     bool
		errContains string
		wantData    map[string]any
	}{
		{
			name: "valid YAML map",
			provider: &mockDataProvider{
				output: []byte("key1: value1\nkey2: value2\n"),
			},
			clusterName: "test-cluster",
			wantErr:     false,
			wantData: map[string]any{
				"key1": "value1",
				"key2": "value2",
			},
		},
		{
			name: "nested YAML map",
			provider: &mockDataProvider{
				output: []byte("outer:\n  inner: value\n"),
			},
			clusterName: "test-cluster",
			wantErr:     false,
			wantData: map[string]any{
				"outer": map[string]any{
					"inner": "value",
				},
			},
		},
		{
			name: "provider returns error",
			provider: &mockDataProvider{
				err: errors.New("provider failed"),
			},
			clusterName: "test-cluster",
			wantErr:     true,
			errContains: "failed to load data from provider",
		},
		{
			name: "invalid YAML syntax",
			provider: &mockDataProvider{
				output: []byte("invalid: yaml: syntax: error\n  bad indentation"),
			},
			clusterName: "test-cluster",
			wantErr:     true,
			errContains: "failed to parse data provider output: invalid YAML",
		},
		{
			name: "YAML array instead of map",
			provider: &mockDataProvider{
				output: []byte("- item1\n- item2\n"),
			},
			clusterName: "test-cluster",
			wantErr:     true,
			errContains: "failed to parse data provider output: invalid YAML",
		},
		{
			name: "YAML scalar instead of map",
			provider: &mockDataProvider{
				output: []byte("just a string\n"),
			},
			clusterName: "test-cluster",
			wantErr:     true,
			errContains: "failed to parse data provider output: invalid YAML",
		},
		{
			name: "empty YAML",
			provider: &mockDataProvider{
				output: []byte(""),
			},
			clusterName: "test-cluster",
			wantErr:     true,
			errContains: "data provider returned null/empty data",
		},
		{
			name: "null YAML",
			provider: &mockDataProvider{
				output: []byte("null\n"),
			},
			clusterName: "test-cluster",
			wantErr:     true,
			errContains: "data provider returned null/empty data",
		},
		{
			name: "empty map",
			provider: &mockDataProvider{
				output: []byte("{}\n"),
			},
			clusterName: "test-cluster",
			wantErr:     false,
			wantData:    map[string]any{},
		},
		{
			name: "complex nested structure",
			provider: &mockDataProvider{
				output: []byte(`
region: us-west-2
environment: production
settings:
  timeout: 30
  retries: 3
  features:
    - feature1
    - feature2
`),
			},
			clusterName: "test-cluster",
			wantErr:     false,
			wantData: map[string]any{
				"region":      "us-west-2",
				"environment": "production",
				"settings": map[string]any{
					"timeout": 30,
					"retries": 3,
					"features": []any{
						"feature1",
						"feature2",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := LoadDataYAML(tt.provider, tt.clusterName)

			if tt.wantErr {
				if err == nil {
					t.Errorf("LoadDataYAML() expected error but got none")
					return
				}
				if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("LoadDataYAML() error = %v, want error containing %v", err, tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("LoadDataYAML() unexpected error = %v", err)
				return
			}

			if !mapsEqual(data, tt.wantData) {
				t.Errorf("LoadDataYAML() = %v, want %v", data, tt.wantData)
			}
		})
	}
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && containsHelper(s, substr)))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// mapsEqual compares two maps for equality
func mapsEqual(a, b map[string]any) bool {
	if len(a) != len(b) {
		return false
	}

	for k, v := range a {
		bv, ok := b[k]
		if !ok {
			return false
		}

		// Handle nested maps
		aMap, aIsMap := v.(map[string]any)
		bMap, bIsMap := bv.(map[string]any)
		if aIsMap && bIsMap {
			if !mapsEqual(aMap, bMap) {
				return false
			}
			continue
		}

		// Handle slices
		aSlice, aIsSlice := v.([]any)
		bSlice, bIsSlice := bv.([]any)
		if aIsSlice && bIsSlice {
			if !slicesEqual(aSlice, bSlice) {
				return false
			}
			continue
		}

		// Direct comparison for other types
		if v != bv {
			return false
		}
	}

	return true
}

// slicesEqual compares two slices for equality
func slicesEqual(a, b []any) bool {
	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}
