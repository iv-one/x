package tenancy

import (
	"context"

	"github.com/iv-one/x/cache"
	"github.com/iv-one/x/entity"
	"github.com/iv-one/x/errorsx/raise"
)

// registry is the process-wide fan-out for tenant-level invalidation. Every
// [TenantCache] registers with it, so [InvalidateTenant] reaches them all.
// A package singleton for now. Registration is permanent, so caches built in
// tests stay registered for the rest of the test binary; eviction of a key
// they never held is a no-op, so this is harmless.
var registry = cache.NewRegistry()

// TenantCache memoises one T per tenant: built by a factory on first use,
// read often, and dropped wholesale by [InvalidateTenant] whenever anything
// about the tenant changes. Rebuilding unrelated records on a tenant-level
// change is deliberate — it is cheap and happens only when an admin edits
// the tenant's apps or settings.
//
// Entries are keyed by the tenant in ctx (see [ContextWithTenant]); there is
// no other way to address them, which is what lets a single tenant ID evict
// every cache at once.
type TenantCache[T any] struct {
	records *cache.SyncCache[T]
}

// NewTenantCache creates an empty cache and registers it for tenant-level
// invalidation.
func NewTenantCache[T any]() *TenantCache[T] {
	c := &TenantCache[T]{records: cache.NewSyncCache[T]()}
	registry.Register(c.records.Deleter())
	return c
}

// Get returns the record for the tenant in ctx, calling create to build it
// if absent. Fails when ctx carries no tenant.
func (c *TenantCache[T]) Get(ctx context.Context, create cache.NewFn[T]) (*T, error) {
	tenantID, err := TenantFromContext(ctx)
	if err != nil {
		return nil, raise.Error(err)
	}

	return c.records.Get(ctx, tenantKey(tenantID), create)
}

// InvalidateTenant drops the tenant's entry from every [TenantCache] in the
// process. Call it after any write that changes what a tenant's caches were
// built from: config, apps, domains, keys.
func InvalidateTenant(ctx context.Context, tenantID entity.ID) error {
	return registry.Invalidate(ctx, tenantKey(tenantID))
}

func tenantKey(id entity.ID) cache.Key { return cache.StrKey(id.Str()) }
