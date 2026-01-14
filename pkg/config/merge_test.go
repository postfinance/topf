package config

import (
	"reflect"
	"testing"
)

func TestDeepMerge(t *testing.T) {
	tests := []struct {
		name     string
		dst      map[string]any
		src      map[string]any
		expected map[string]any
	}{
		{
			name:     "simple override",
			dst:      map[string]any{"key1": "old"},
			src:      map[string]any{"key1": "new"},
			expected: map[string]any{"key1": "new"},
		},
		{
			name: "nested merge",
			dst: map[string]any{
				"outer": map[string]any{
					"inner1": "keep",
					"inner2": "old",
				},
			},
			src: map[string]any{
				"outer": map[string]any{
					"inner2": "new",
					"inner3": "add",
				},
			},
			expected: map[string]any{
				"outer": map[string]any{
					"inner1": "keep",
					"inner2": "new",
					"inner3": "add",
				},
			},
		},
		{
			name: "different types override - map to scalar",
			dst: map[string]any{
				"key": map[string]any{"nested": "value"},
			},
			src: map[string]any{
				"key": "scalar",
			},
			expected: map[string]any{
				"key": "scalar",
			},
		},
		{
			name: "different types override - scalar to map",
			dst: map[string]any{
				"key": "scalar",
			},
			src: map[string]any{
				"key": map[string]any{"nested": "value"},
			},
			expected: map[string]any{
				"key": map[string]any{"nested": "value"},
			},
		},
		{
			name: "array override (not merge)",
			dst: map[string]any{
				"arr": []any{1, 2, 3},
			},
			src: map[string]any{
				"arr": []any{4, 5},
			},
			expected: map[string]any{
				"arr": []any{4, 5},
			},
		},
		{
			name:     "nil destination",
			dst:      nil,
			src:      map[string]any{"key": "value"},
			expected: map[string]any{"key": "value"},
		},
		{
			name:     "empty source",
			dst:      map[string]any{"key": "value"},
			src:      map[string]any{},
			expected: map[string]any{"key": "value"},
		},
		{
			name: "add new keys",
			dst: map[string]any{
				"key1": "value1",
			},
			src: map[string]any{
				"key2": "value2",
				"key3": "value3",
			},
			expected: map[string]any{
				"key1": "value1",
				"key2": "value2",
				"key3": "value3",
			},
		},
		{
			name: "deeply nested merge",
			dst: map[string]any{
				"level1": map[string]any{
					"level2": map[string]any{
						"level3": map[string]any{
							"keep": "this",
							"old":  "value",
						},
					},
				},
			},
			src: map[string]any{
				"level1": map[string]any{
					"level2": map[string]any{
						"level3": map[string]any{
							"old": "new",
							"add": "this",
						},
					},
				},
			},
			expected: map[string]any{
				"level1": map[string]any{
					"level2": map[string]any{
						"level3": map[string]any{
							"keep": "this",
							"old":  "new",
							"add":  "this",
						},
					},
				},
			},
		},
		{
			name: "mixed types",
			dst: map[string]any{
				"string": "value",
				"int":    42,
				"float":  3.14,
				"bool":   true,
			},
			src: map[string]any{
				"string": "newvalue",
				"int":    100,
			},
			expected: map[string]any{
				"string": "newvalue",
				"int":    100,
				"float":  3.14,
				"bool":   true,
			},
		},
		{
			name: "complex example",
			dst: map[string]any{
				"cluster": map[string]any{
					"name":   "prod",
					"region": "us-east-1",
				},
				"settings": map[string]any{
					"timeout": 30,
				},
			},
			src: map[string]any{
				"cluster": map[string]any{
					"region": "us-west-2",
					"zone":   "a",
				},
				"settings": map[string]any{
					"retries": 3,
				},
			},
			expected: map[string]any{
				"cluster": map[string]any{
					"name":   "prod",
					"region": "us-west-2",
					"zone":   "a",
				},
				"settings": map[string]any{
					"timeout": 30,
					"retries": 3,
				},
			},
		},
		{
			name:     "both empty",
			dst:      map[string]any{},
			src:      map[string]any{},
			expected: map[string]any{},
		},
		{
			name: "nil values in source",
			dst: map[string]any{
				"key": "value",
			},
			src: map[string]any{
				"key": nil,
			},
			expected: map[string]any{
				"key": nil,
			},
		},
		{
			name: "map with slice values",
			dst: map[string]any{
				"items": []any{"a", "b"},
			},
			src: map[string]any{
				"items": []any{"c", "d", "e"},
			},
			expected: map[string]any{
				"items": []any{"c", "d", "e"},
			},
		},
		{
			name: "nested maps with different structures",
			dst: map[string]any{
				"config": map[string]any{
					"database": map[string]any{
						"host": "localhost",
						"port": 5432,
					},
				},
			},
			src: map[string]any{
				"config": map[string]any{
					"database": map[string]any{
						"password": "secret",
					},
					"cache": map[string]any{
						"enabled": true,
					},
				},
			},
			expected: map[string]any{
				"config": map[string]any{
					"database": map[string]any{
						"host":     "localhost",
						"port":     5432,
						"password": "secret",
					},
					"cache": map[string]any{
						"enabled": true,
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := deepMerge(tt.dst, tt.src)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("deepMerge() = %v, want %v", result, tt.expected)
			}
		})
	}
}
