package tenancy

import (
	"context"
	"testing"

	"github.com/iv-one/x/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type strID string

func (s strID) Str() string   { return string(s) }
func (s strID) Bytes() []byte { return []byte(s) }
func (s strID) Eq(o entity.ID) bool {
	return o != nil && o.Str() == string(s)
}
func (s strID) IsNil() bool { return s == "" }

type thing struct{ built int }

func countingFactory() (func(context.Context) (*thing, error), *int) {
	n := 0
	return func(context.Context) (*thing, error) {
		n++
		return &thing{built: n}, nil
	}, &n
}

func TestTenantCache_GetPerTenant(t *testing.T) {
	c := NewTenantCache[thing]()
	create, calls := countingFactory()

	ctxA := ContextWithTenant(t.Context(), strID("a"))
	ctxB := ContextWithTenant(t.Context(), strID("b"))

	a1, err := c.Get(ctxA, create)
	require.NoError(t, err)
	a2, err := c.Get(ctxA, create)
	require.NoError(t, err)
	assert.Same(t, a1, a2, "same tenant, same record")

	b, err := c.Get(ctxB, create)
	require.NoError(t, err)
	assert.NotSame(t, a1, b, "tenants do not share records")
	assert.Equal(t, 2, *calls)
}

func TestTenantCache_RequiresTenant(t *testing.T) {
	c := NewTenantCache[thing]()
	create, calls := countingFactory()

	_, err := c.Get(t.Context(), create)
	require.Error(t, err)
	assert.Equal(t, 0, *calls, "factory must not run without a tenant")
}

// Fan-out across several caches is the Registry's contract and is tested in
// package cache; this pins only the tenancy binding: same key on Get and on
// InvalidateTenant, other tenants untouched.
func TestInvalidateTenant(t *testing.T) {
	c := NewTenantCache[thing]()
	create, calls := countingFactory()

	ctxA := ContextWithTenant(t.Context(), strID("a"))
	ctxB := ContextWithTenant(t.Context(), strID("b"))
	for _, ctx := range []context.Context{ctxA, ctxB} {
		_, err := c.Get(ctx, create)
		require.NoError(t, err)
	}

	require.NoError(t, InvalidateTenant(t.Context(), strID("a")))

	_, err := c.Get(ctxA, create) // rebuilt
	require.NoError(t, err)
	_, err = c.Get(ctxB, create) // untouched
	require.NoError(t, err)
	assert.Equal(t, 3, *calls)
}
