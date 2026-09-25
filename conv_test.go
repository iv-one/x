package x

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMem(t *testing.T) {
	assert := assert.New(t)

	tests := []struct {
		name     string
		input    string
		expected int64
		hasError bool
	}{
		{
			name:     "Test with KB",
			input:    "256KB",
			expected: 256 * 1024,
			hasError: false,
		},
		{
			name:     "Test with MB",
			input:    "256MB",
			expected: 256 * 1024 * 1024,
			hasError: false,
		},
		{
			name:     "Test with GB",
			input:    "256GB",
			expected: 256 * 1024 * 1024 * 1024,
			hasError: false,
		},
		{
			name:     "Test with lowercase",
			input:    "256mb",
			expected: 256 * 1024 * 1024,
			hasError: false,
		},
		{
			name:     "Test with invalid input",
			input:    "invalid",
			expected: 0,
			hasError: true,
		},
		{
			name:     "Test with no unit",
			input:    "256",
			expected: 0,
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseMem(tt.input)
			if tt.hasError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(tt.expected, result)
			}
		})
	}
}
