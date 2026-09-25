package x

import (
	"crypto/rand"
	"io"
)

// EncKey is the interface for encryption keys.
type EncKey interface {
	Bytes() []byte
}

// EmptyKey is the empty encryption key.
var EmptyKey = Sensitive([]byte{})

// NewRandomKey generates a new random encryption key.
func NewRandomKey() Sensitive {
	const length = 32
	bytes := make([]byte, length)
	if _, err := io.ReadFull(rand.Reader, bytes); err != nil {
		panic(err)
	}
	return Sensitive(bytes)
}

// Sensitive implements the Stringer interface to redact its contents.
// Use this type for sensitive info such as keys, passwords, or secrets so it doesn't leak
// as output such as logs.
// Idea is taken from https://github.com/dgraph-io/dgraph/blob/main/x/types.go#L27
type Sensitive []byte

// SensitiveFromString creates a new Sensitive from the given string.
func SensitiveFromString(s string) Sensitive {
	return Sensitive([]byte(s))
}

// String implements the Stringer interface to redact its contents.
func (Sensitive) String() string {
	return "****"
}

// Unwrap returns the underlying string value. This should be used carefully because it
// defeats the purpose of the Sensitive type.
func (s Sensitive) Unwrap() string {
	return string(s)
}

// Bytes returns the underlying bytes value.
func (s Sensitive) Bytes() []byte {
	return []byte(s)
}

// Empty checks if the Sensitive is empty.
func (s Sensitive) Empty() bool {
	return len(s) == 0
}

// NotEmpty checks if the Sensitive is not empty.
func (s Sensitive) NotEmpty() bool {
	return !s.Empty()
}

// Equal checks if the Sensitive is equal to the given Sensitive.
func (s Sensitive) Equal(to Sensitive) bool {
	return string(s) == string(to)
}

// SensitiveStr converts the Sensitive to a SensitiveStr.
func (s Sensitive) SensitiveStr() SensitiveStr {
	return SensitiveStr(s)
}

// SensitiveStr is the string type for sensitive information.
type SensitiveStr string

// String implements the Stringer interface to redact its contents.
func (SensitiveStr) String() string {
	return "****"
}

// Unwrap returns the underlying string value. This should be used carefully because it
// defeats the purpose of the Sensitive type.
func (s SensitiveStr) Unwrap() string {
	return string(s)
}

// Bytes returns the underlying bytes value.
func (s SensitiveStr) Bytes() []byte {
	return []byte(s)
}

// Empty checks if the SensitiveStr is empty.
func (s SensitiveStr) Empty() bool {
	return len(s) == 0
}

// Sensitive converts the SensitiveStr to a Sensitive.
func (s SensitiveStr) Sensitive() Sensitive {
	return Sensitive(s)
}

// Equal checks if the SensitiveStr is equal to the given SensitiveStr.
func (s SensitiveStr) Equal(to SensitiveStr) bool {
	return string(s) == string(to)
}
