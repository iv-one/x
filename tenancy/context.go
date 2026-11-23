package tenancy

import (
	"context"
	"errors"

	"github.com/iv-one/x/entity"
	"github.com/iv-one/x/errorsx"
	"github.com/iv-one/x/errorsx/raise"
)

var (
	// ErrNoTenant is an error that occurs when no tenant information is available in the provided context.
	ErrNoTenant = errorsx.ErrUnauthorized.WithReason("No tenant information available in the provided context. Please ensure the context is initialized with tenant data.")

	// ErrInvalidTenant is an error that occurs when the tenant information is invalid.
	ErrInvalidTenant = errors.New("invalid tenant format")
)

type tenantIDKeyType string

// TenantIDKey is the key for the tenant ID in the context.
const TenantIDKey tenantIDKeyType = "tenant"

// ContextWithTenant clone a new context with tenant information.
func ContextWithTenant(ctx context.Context, id entity.ID) context.Context {
	return context.WithValue(ctx, TenantIDKey, id)
}

// TenantFromContext gets the tenant ID from the context.
func TenantFromContext(ctx context.Context) (entity.ID, error) {
	tenantValue := ctx.Value(TenantIDKey)
	if tenantValue == nil {
		return nil, raise.Error(ErrNoTenant)
	}
	if res, ok := tenantValue.(entity.ID); ok {
		return res, nil
	}
	return nil, raise.Error(ErrInvalidTenant)
}

// TenantFromContextTyped gets the tenant ID from the context.
func TenantFromContextTyped[T entity.ID](ctx context.Context) (T, error) {
	var v T
	tenantID, err := TenantFromContext(ctx)
	if err != nil {
		return v, err
	}
	if res, ok := tenantID.(T); ok {
		return res, nil
	}
	return v, raise.Error(ErrInvalidTenant)
}

// IsTenantPresented checks if the tenant ID is presented in the context.
func IsTenantPresented(ctx context.Context) bool {
	_, err := TenantFromContext(ctx)
	return err == nil
}

// TestContext returns the test's context carrying the given tenant id, the
// shape every tenant-scoped call under test expects.
func TestContext(t interface{ Context() context.Context }, id string) context.Context {
	return ContextWithTenant(t.Context(), entity.RawID(id))
}
