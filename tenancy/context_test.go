package tenancy

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/iv-one/x/entity"
)

func TestTenantFromContext(t *testing.T) {
	_, err := TenantFromContext(context.Background())
	require.ErrorIs(t, err, ErrNoTenant)
	assert.False(t, IsTenantPresented(context.Background()))

	bad := context.WithValue(context.Background(), TenantIDKey, "not an id")
	_, err = TenantFromContext(bad)
	require.ErrorIs(t, err, ErrInvalidTenant)
	assert.False(t, IsTenantPresented(bad))

	ctx := TestContext(t, "t1")
	assert.True(t, IsTenantPresented(ctx))
	id, err := TenantFromContext(ctx)
	require.NoError(t, err)
	assert.Equal(t, "t1", id.Str())
}

func TestTenantFromContextTyped(t *testing.T) {
	ctx := TestContext(t, "t1")

	raw, err := TenantFromContextTyped[entity.RawID](ctx)
	require.NoError(t, err)
	assert.Equal(t, entity.RawID("t1"), raw)

	_, err = TenantFromContextTyped[*otherID](ctx)
	require.ErrorIs(t, err, ErrInvalidTenant, "a tenant of another type is refused")

	_, err = TenantFromContextTyped[entity.RawID](context.Background())
	require.ErrorIs(t, err, ErrNoTenant)
}

type otherID struct{ entity.RawID }
