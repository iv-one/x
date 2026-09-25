// Package cache provides small, generic in-memory cache primitives keyed by
// string-like values.
//
// A [Key] is anything that can render itself as a string; plain strings are
// wrapped with [StrKey]. Records are created lazily by a [NewFn] passed to
// [Cache.Get], mutated atomically with [Cache.Update], and removed with
// [Cache.Delete]. A [Registry] fans one Invalidate out to every cache that
// shares a key space.
//
// # Concurrency
//
// [SyncCache] is the bundled implementation and is safe for concurrent use.
// It is a per-key singleflight: the first Get for an absent key runs the
// factory, later Gets for the same key wait for that one result, and Gets
// for other keys are never blocked by it. The internal mutex guards only
// the map, so a slow factory for one key costs nothing to the rest of the
// cache. Records are shared pointers, not copies.
//
// A factory that returns an error (or panics) leaves nothing behind: its
// waiters receive the error and the next Get starts a fresh attempt. Update
// waits for an in-flight factory on its key before applying the update;
// Delete and DeleteFunc drop in-flight entries too, so a load that started
// before an invalidation cannot resurrect stale data.
package cache
