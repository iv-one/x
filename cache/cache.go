package cache

import (
	"context"
	"errors"
	"sync"
)

// ErrNilRecord is returned to waiters when a factory produced neither a
// record nor an error (it returned (nil, nil) or panicked).
var ErrNilRecord = errors.New("cache: factory produced no record")

// Key identifies a cached record. Any type with a String method — including
// every fmt.Stringer — can be used as a key; plain strings use [StrKey].
// A nil Key is a programming error and panics.
type Key interface {
	String() string
}

// StrKey adapts a plain string to [Key].
type StrKey string

// String returns the key as a string.
func (s StrKey) String() string { return string(s) }

// NewFn builds a record for a key that is not yet cached. Returning an error
// aborts the lookup and leaves the cache unchanged, so a later Get will call
// NewFn again. Returning (nil, nil) is treated as [ErrNilRecord].
type NewFn[T any] func(context.Context) (*T, error)

// UpdateFn transforms the current record for a key and returns its
// replacement. current is nil when the key is absent. Returning a nil record
// removes the key; returning an error leaves the cache unchanged.
type UpdateFn[T any] func(ctx context.Context, current *T) (*T, error)

// DeleteFn removes the record stored under a key.
type DeleteFn func(context.Context, Key) error

// Cache is a key/value store whose records are created on demand.
//
// Records are shared, not copied: Get returns the stored pointer, so callers
// that mutate *T concurrently must synchronize themselves or go through
// Update.
type Cache[T any] interface {
	// Get returns the record for key, calling create to build and store it
	// if absent.
	Get(ctx context.Context, key Key, create NewFn[T]) (*T, error)
	// Update applies fn to the record for key and stores the result. The
	// returned record is nil when fn removed the key.
	Update(ctx context.Context, key Key, fn UpdateFn[T]) (*T, error)
	// Delete removes the record for key. Deleting an absent key is a no-op.
	Delete(ctx context.Context, key Key) error
	// Deleter returns Delete as a standalone [DeleteFn], handy for passing
	// eviction to code that must not hold the whole cache.
	Deleter() DeleteFn
}

// entry is one slot in the map. done is closed once rec/err are final; a
// slot whose done is still open is an in-flight create that other callers
// wait on.
type entry[T any] struct {
	done chan struct{}
	rec  *T
	err  error
}

func readyEntry[T any](rec *T) *entry[T] {
	e := &entry[T]{done: make(chan struct{}), rec: rec}
	close(e.done)
	return e
}

func (e *entry[T]) ready() bool {
	select {
	case <-e.done:
		return true
	default:
		return false
	}
}

// SyncCache is an in-memory [Cache] that is safe for concurrent use. It is a
// per-key singleflight: concurrent Gets for a missing key share one create
// call, while Gets for other keys proceed independently. Only the map is
// guarded by the mutex; factories run outside it, but [UpdateFn] and the
// match function of [SyncCache.DeleteFunc] still run under it — keep them
// quick. The cache never inspects ctx; it is only forwarded to callbacks.
//
// Construct with [NewSyncCache] or [NewSyncCacheCap]; the zero value is not
// usable.
type SyncCache[T any] struct {
	mu      sync.Mutex
	max     int // entry cap; 0 is unbounded
	records map[string]*entry[T]
}

var _ Cache[struct{}] = (*SyncCache[struct{}])(nil)

// NewSyncCache returns an empty, unbounded SyncCache.
func NewSyncCache[T any]() *SyncCache[T] {
	return &SyncCache[T]{records: make(map[string]*entry[T])}
}

// NewSyncCacheCap returns an empty SyncCache holding at most max entries.
// When an insert would exceed the cap, an arbitrary other record is evicted
// first — settled records before in-flight creates, never the record being
// inserted. Eviction is deliberately not LRU: the cap is a memory safety
// valve for caches whose correctness never depends on what is cached (an
// evicted record is simply rebuilt on its next Get), so per-Get recency
// bookkeeping would buy nothing. A capacity < 1 means unbounded.
func NewSyncCacheCap[T any](capacity int) *SyncCache[T] {
	return &SyncCache[T]{max: capacity, records: make(map[string]*entry[T])}
}

// Get implements [Cache].
func (c *SyncCache[T]) Get(ctx context.Context, key Key, create NewFn[T]) (rec *T, err error) {
	k := key.String()

	c.mu.Lock()
	e, ok := c.records[k]
	if ok {
		c.mu.Unlock()
		<-e.done // immediate for a stored record, waits for an in-flight create
		return e.rec, e.err
	}
	e = &entry[T]{done: make(chan struct{})}
	c.records[k] = e
	c.evictOverCap(k)
	c.mu.Unlock()

	// Claim the flight. Whatever happens — error, (nil, nil), panic — the
	// placeholder must not survive, and waiters must be released.
	defer func() {
		if e.err == nil && e.rec == nil {
			e.err = ErrNilRecord
		}
		if e.err != nil {
			c.mu.Lock()
			if c.records[k] == e {
				delete(c.records, k)
			}
			c.mu.Unlock()
		}
		close(e.done)
		rec, err = e.rec, e.err // named results: reflect what the defer settled
	}()
	e.rec, e.err = create(ctx)
	return e.rec, e.err
}

// Update implements [Cache]. It waits for an in-flight create on the key so
// fn always sees the settled record.
func (c *SyncCache[T]) Update(ctx context.Context, key Key, fn UpdateFn[T]) (*T, error) {
	k := key.String()

	c.mu.Lock()
	defer c.mu.Unlock()

	// A Delete may drop the flight we waited on and a new Get may start
	// another, so re-check after every wake-up.
	e := c.records[k]
	for e != nil && !e.ready() {
		c.mu.Unlock()
		<-e.done
		c.mu.Lock()
		e = c.records[k]
	}

	var current *T
	if e != nil {
		current = e.rec // a settled entry in the map always holds a record
	}

	next, err := fn(ctx, current)
	if err != nil {
		return nil, err
	}
	if next == nil {
		delete(c.records, k)
		return nil, nil
	}
	c.records[k] = readyEntry(next)
	c.evictOverCap(k)
	return next, nil
}

// evictOverCap removes arbitrary records until the cache respects its cap,
// sparing keep (the record just inserted). Settled records are evicted before
// in-flight creates; an evicted in-flight create still delivers its record to
// waiters, as with Delete. Callers must hold mu.
func (c *SyncCache[T]) evictOverCap(keep string) {
	if c.max < 1 {
		return
	}
	for len(c.records) > c.max {
		if !c.evictAny(keep, true) && !c.evictAny(keep, false) {
			return // only keep is left; the cap cannot be honored
		}
	}
}

// evictAny deletes one record other than keep — only a settled one when
// readyOnly — and reports whether it deleted anything. Callers must hold mu.
func (c *SyncCache[T]) evictAny(keep string, readyOnly bool) bool {
	for k, e := range c.records {
		if k == keep || (readyOnly && !e.ready()) {
			continue
		}
		delete(c.records, k)
		return true
	}
	return false
}

// Delete implements [Cache]. It never returns an error. An in-flight create
// for the key is dropped from the map as well; its waiters still receive
// the record they were promised, but the next Get starts afresh.
func (c *SyncCache[T]) Delete(_ context.Context, key Key) error {
	c.mu.Lock()
	delete(c.records, key.String())
	c.mu.Unlock()
	return nil
}

// Deleter implements [Cache].
func (c *SyncCache[T]) Deleter() DeleteFn { return c.Delete }

// DeleteFunc removes every stored record for which match returns true and
// reports how many were removed. match runs under the lock; keep it quick.
// In-flight creates are removed too, without consulting match: their result
// is unknown and may predate the change that triggered the eviction.
// Use it to evict by a property of the record when the keys are not known,
// e.g. every entry pointing at an entity that just changed.
func (c *SyncCache[T]) DeleteFunc(match func(*T) bool) int {
	c.mu.Lock()
	defer c.mu.Unlock()

	n := 0
	for k, e := range c.records {
		if !e.ready() || match(e.rec) {
			delete(c.records, k)
			n++
		}
	}
	return n
}

// Len reports the number of entries, including creates still in flight.
func (c *SyncCache[T]) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.records)
}
