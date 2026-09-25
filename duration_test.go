package x

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseDuration(t *testing.T) {
	assert := assert.New(t)

	tests := []struct {
		name     string
		input    string
		expected time.Duration
		hasError bool
	}{
		{
			name:     "Test with days",
			input:    "2d",
			expected: 48 * time.Hour,
			hasError: false,
		},
		{
			name:     "Test with weeks",
			input:    "3w",
			expected: 504 * time.Hour,
			hasError: false,
		},
		{
			name:     "Test with months",
			input:    "1mo",
			expected: 720 * time.Hour,
			hasError: false,
		},
		{
			name:     "Test with nanoseconds",
			input:    "100ns",
			expected: 100 * time.Nanosecond,
			hasError: false,
		},
		{
			name:     "Test with microseconds",
			input:    "100us",
			expected: 100 * time.Microsecond,
			hasError: false,
		},
		{
			name:     "Test with milliseconds",
			input:    "100ms",
			expected: 100 * time.Millisecond,
			hasError: false,
		},
		{
			name:     "Test with seconds",
			input:    "100s",
			expected: 100 * time.Second,
			hasError: false,
		},
		{
			name:     "Test with minutes",
			input:    "100m",
			expected: 100 * time.Minute,
			hasError: false,
		},
		{
			name:     "Test with hours",
			input:    "100h",
			expected: 100 * time.Hour,
			hasError: false,
		},
		{
			name:     "Test with invalid input",
			input:    "invalid",
			expected: 0,
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseDuration(tt.input)
			if tt.hasError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(tt.expected, result)
			}
		})
	}
}

// TestParseShort for durafmt time.Duration conversion, short version.
func TestParseShort(t *testing.T) {
	testTimes := []struct {
		test     time.Duration
		expected string
	}{
		{1 * time.Hour, "1 hour"},
		{1 * time.Minute, "1 minute"},
		{2 * time.Minute, "2 minutes"},
		{2 * time.Hour, "2 hours"},
		{24 * time.Hour, "1 day"},
		{48 * time.Hour, "2 days"},
		{120 * time.Hour, "5 days"},
		{168 * time.Hour, "1 week"},
		{672 * time.Hour, "4 weeks"},
	}

	for _, table := range testTimes {
		result := FormatDuration(table.test)
		if result != table.expected {
			t.Errorf("Parse(%q).String() = %q. got %q, expected %q",
				table.test, result, result, table.expected)
		}
	}
}
