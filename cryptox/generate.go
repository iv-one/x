// Package cryptox generates random keys and passwords.
package cryptox

import (
	"crypto/rand"
	"io"

	"github.com/iv-one/x"
	"github.com/iv-one/x/errorsx/raise"
)

// GenerateRandomKey generates a random key with the given length.
// It has Modulo Bias problem, but it is not critical for this use case
func GenerateRandomKey(length int) x.Sensitive {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	k := make([]byte, length)

	if _, err := io.ReadFull(rand.Reader, k); err != nil {
		return x.Sensitive{}
	}

	for i, b := range k {
		k[i] = letters[b%byte(len(letters))]
	}

	return x.Sensitive(k)
}

// GeneratePassword generates a random password with the given length.
// It is recommended to provide secure password through config or change it after the first login
func GeneratePassword() x.Sensitive {
	// default password size is 32
	return GenerateRandomKey(32)
}

// GenerateStrictRandomKey generates a random key with the given length
func GenerateStrictRandomKey(length int) (x.Sensitive, error) {
	k := make([]byte, length)

	if _, err := io.ReadFull(rand.Reader, k); err != nil {
		return x.Sensitive{}, raise.Error(err)
	}

	return x.Sensitive(k), nil
}
