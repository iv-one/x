package cache

import (
	"context"
	"errors"
	"sync"
)

// Registry fans out invalidation by key to every cache registered with it.
//
// Caches that share a key space — e.g. everything keyed by tenant ID —
// register their [DeleteFn] once; a single [Registry.Invalidate] then evicts
// that key everywhere. Registration is append-only: caches live as long as
// the process, so there is no Unregister.
type Registry struct {
	mu       sync.RWMutex
	deleters []DeleteFn
}

// NewRegistry returns an empty Registry.
func NewRegistry() *Registry { return &Registry{} }

// Register adds a cache's delete function to the fan-out.
func (r *Registry) Register(del DeleteFn) {
	r.mu.Lock()
	r.deleters = append(r.deleters, del)
	r.mu.Unlock()
}

// Invalidate evicts key from every registered cache. All deleters run even if
// some fail; their errors are joined.
func (r *Registry) Invalidate(ctx context.Context, key Key) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var errs []error
	for _, del := range r.deleters {
		if err := del(ctx, key); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}
