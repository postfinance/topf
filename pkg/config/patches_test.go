package config

import "testing"

func TestIsEmpty(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected bool
	}{
		{
			name:     "completely empty",
			content:  "",
			expected: true,
		},
		{
			name: "only whitespace",
			content: `

  `,
			expected: true,
		},
		{
			name: "only comments - unmarshals to nil",
			content: `# This is a comment
# Another comment`,
			expected: true,
		},
		{
			name: "comments with whitespace - unmarshals to nil",
			content: `
  # Comment 1

  # Comment 2

`,
			expected: true,
		},
		{
			name:     "empty yaml map",
			content:  "{}",
			expected: true,
		},
		{
			name:     "empty yaml list",
			content:  "[]",
			expected: true,
		},
		{
			name:     "yaml null",
			content:  "null",
			expected: true,
		},
		{
			name:     "yaml tilde (null)",
			content:  "~",
			expected: true,
		},
		{
			name: "yaml content",
			content: `machine:
  type: worker`,
			expected: false,
		},
		{
			name: "yaml with comments",
			content: `# Comment
machine:
  type: worker`,
			expected: false,
		},
		{
			name: "yaml with trailing comment",
			content: `machine:
  type: worker
# Comment`,
			expected: false,
		},
		{
			name: "yaml list with data",
			content: `- op: add
  path: /machine/network`,
			expected: false,
		},
		{
			name:     "simple string value",
			content:  "foo: bar",
			expected: false,
		},
		{
			name:     "yaml number",
			content:  "123",
			expected: false,
		},
		{
			name:     "yaml boolean",
			content:  "true",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isEmpty([]byte(tt.content))
			if result != tt.expected {
				t.Errorf("isEmpty() = %v, want %v for content:\n%q", result, tt.expected, tt.content)
			}
		})
	}
}
