package cache

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegistry_InvalidateFansOut(t *testing.T) {
	reg := NewRegistry()
	a, b := NewSyncCache[rec](), NewSyncCache[rec]()
	reg.Register(a.Deleter())
	reg.Register(b.Deleter())

	for _, c := range []*SyncCache[rec]{a, b} {
		for _, k := range []string{"t1", "t2"} {
			_, err := c.Get(t.Context(), StrKey(k), newRec(1))
			require.NoError(t, err)
		}
	}

	require.NoError(t, reg.Invalidate(t.Context(), StrKey("t1")))

	for _, c := range []*SyncCache[rec]{a, b} {
		assert.Equal(t, 1, c.Len())
		_, err := c.Get(t.Context(), StrKey("t2"), failNew) // still cached
		require.NoError(t, err)
	}
}

func TestRegistry_InvalidateJoinsErrors(t *testing.T) {
	reg := NewRegistry()
	boom1, boom2 := errors.New("boom1"), errors.New("boom2")
	var calls int
	reg.Register(func(context.Context, Key) error { calls++; return boom1 })
	reg.Register(func(context.Context, Key) error { calls++; return nil })
	reg.Register(func(context.Context, Key) error { calls++; return boom2 })

	err := reg.Invalidate(t.Context(), StrKey("k"))
	require.ErrorIs(t, err, boom1)
	require.ErrorIs(t, err, boom2)
	assert.Equal(t, 3, calls, "a failing deleter must not stop the fan-out")
}

func TestRegistry_Empty(t *testing.T) {
	require.NoError(t, NewRegistry().Invalidate(t.Context(), StrKey("k")))
}
