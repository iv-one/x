package x

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSensitive(t *testing.T) {
	s := SensitiveFromString("secret")
	assert.Equal(t, "****", s.String())
	assert.Equal(t, "****", fmt.Sprint(s), "formatting redacts the value")
	assert.Equal(t, "secret", s.Unwrap())
	assert.Equal(t, []byte("secret"), s.Bytes())
	assert.False(t, s.Empty())
	assert.True(t, s.NotEmpty())
	assert.True(t, s.Equal(Sensitive("secret")))
	assert.False(t, s.Equal(Sensitive("other")))
	assert.Equal(t, SensitiveStr("secret"), s.SensitiveStr())

	assert.True(t, EmptyKey.Empty())
	assert.False(t, EmptyKey.NotEmpty())
}

func TestSensitiveStr(t *testing.T) {
	s := SensitiveStr("secret")
	assert.Equal(t, "****", s.String())
	assert.Equal(t, "****", fmt.Sprint(s), "formatting redacts the value")
	assert.Equal(t, "secret", s.Unwrap())
	assert.Equal(t, []byte("secret"), s.Bytes())
	assert.False(t, s.Empty())
	assert.True(t, SensitiveStr("").Empty())
	assert.Equal(t, Sensitive("secret"), s.Sensitive())
	assert.True(t, s.Equal("secret"))
	assert.False(t, s.Equal("other"))
}

func TestNewRandomKey(t *testing.T) {
	k := NewRandomKey()
	require.Len(t, k, 32)
	assert.False(t, k.Equal(NewRandomKey()), "two keys should differ")

	var _ EncKey = k
}

func TestIDs(t *testing.T) {
	a, b := GUID(), GUID()
	assert.Len(t, a, 20)
	assert.NotEqual(t, a, b)

	u := UUIDv4()
	assert.Regexp(t, `^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`, u)
	assert.NotEqual(t, u, UUIDv4())
}

func TestNoError(t *testing.T) {
	assert.True(t, NoError(nil))
	assert.False(t, NoError(assert.AnError))
}

func TestReduceOptions(t *testing.T) {
	type opts struct{ a, b int }
	got := ReduceOptions(&opts{a: 1},
		func(o *opts) { o.b = 2 },
		func(o *opts) { o.a += 10 },
	)
	assert.Equal(t, &opts{a: 11, b: 2}, got)
	assert.Equal(t, &opts{a: 1}, ReduceOptions(&opts{a: 1}))
}
