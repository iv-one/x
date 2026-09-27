package cryptox

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const alphanumeric = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func TestGenerateRandomKey(t *testing.T) {
	k := GenerateRandomKey(64)
	require.Len(t, k, 64)
	for _, c := range k.Unwrap() {
		assert.True(t, strings.ContainsRune(alphanumeric, c), "unexpected character %q", c)
	}
	assert.False(t, k.Equal(GenerateRandomKey(64)), "two keys should differ")
	assert.True(t, GenerateRandomKey(0).Empty())
}

func TestGeneratePassword(t *testing.T) {
	p := GeneratePassword()
	assert.Len(t, p, 32)
	assert.Equal(t, "****", p.String(), "passwords stay redacted")
}

func TestGenerateStrictRandomKey(t *testing.T) {
	k, err := GenerateStrictRandomKey(32)
	require.NoError(t, err)
	assert.Len(t, k, 32)

	other, err := GenerateStrictRandomKey(32)
	require.NoError(t, err)
	assert.False(t, k.Equal(other), "two keys should differ")
}
