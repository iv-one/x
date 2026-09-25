package cache

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type rec struct{ n int }

func newRec(n int) NewFn[rec] {
	return func(context.Context) (*rec, error) { return &rec{n: n}, nil }
}

var errBoom = errors.New("boom")

func failNew(context.Context) (*rec, error) { return nil, errBoom }

// userID is a domain type that satisfies Key via fmt.Stringer.
type userID int

func (u userID) String() string { return fmt.Sprintf("user:%d", u) }

var _ fmt.Stringer = userID(0)

func TestSyncCache_GetCreatesOnce(t *testing.T) {
	c := NewSyncCache[rec]()
	var calls int
	create := func(context.Context) (*rec, error) {
		calls++
		return &rec{n: calls}, nil
	}

	first, err := c.Get(t.Context(), StrKey("a"), create)
	require.NoError(t, err)
	second, err := c.Get(t.Context(), StrKey("a"), create)
	require.NoError(t, err)

	assert.Same(t, first, second)
	assert.Equal(t, 1, calls)
	assert.Equal(t, 1, c.Len())
}

func TestSyncCache_StringerKey(t *testing.T) {
	c := NewSyncCache[rec]()

	_, err := c.Get(t.Context(), userID(7), newRec(1))
	require.NoError(t, err)

	// Same rendered key, different Key type, same record.
	got, err := c.Get(t.Context(), StrKey("user:7"), newRec(2))
	require.NoError(t, err)
	assert.Equal(t, 1, got.n)
}

func TestSyncCache_GetError(t *testing.T) {
	c := NewSyncCache[rec]()

	got, err := c.Get(t.Context(), StrKey("a"), failNew)
	require.ErrorIs(t, err, errBoom)
	assert.Nil(t, got)
	assert.Equal(t, 0, c.Len(), "failed create must not store anything")

	got, err = c.Get(t.Context(), StrKey("a"), newRec(7))
	require.NoError(t, err)
	assert.Equal(t, 7, got.n, "a later create must run again")
}

func TestSyncCache_Update(t *testing.T) {
	t.Run("absent key inserts", func(t *testing.T) {
		c := NewSyncCache[rec]()
		got, err := c.Update(t.Context(), StrKey("a"), func(_ context.Context, cur *rec) (*rec, error) {
			require.Nil(t, cur)
			return &rec{n: 1}, nil
		})
		require.NoError(t, err)
		assert.Equal(t, 1, got.n)
		assert.Equal(t, 1, c.Len())
	})

	t.Run("present key sees current", func(t *testing.T) {
		c := NewSyncCache[rec]()
		_, err := c.Get(t.Context(), StrKey("a"), newRec(1))
		require.NoError(t, err)

		got, err := c.Update(t.Context(), StrKey("a"), func(_ context.Context, cur *rec) (*rec, error) {
			require.Equal(t, 1, cur.n)
			return &rec{n: cur.n + 1}, nil
		})
		require.NoError(t, err)
		assert.Equal(t, 2, got.n)

		stored, err := c.Get(t.Context(), StrKey("a"), failNew)
		require.NoError(t, err)
		assert.Same(t, got, stored)
	})

	t.Run("error leaves cache untouched", func(t *testing.T) {
		c := NewSyncCache[rec]()
		orig, err := c.Get(t.Context(), StrKey("a"), newRec(1))
		require.NoError(t, err)

		_, err = c.Update(t.Context(), StrKey("a"), func(context.Context, *rec) (*rec, error) { return nil, errBoom })
		require.ErrorIs(t, err, errBoom)

		stored, err := c.Get(t.Context(), StrKey("a"), failNew)
		require.NoError(t, err)
		assert.Same(t, orig, stored)
	})

	t.Run("nil result deletes", func(t *testing.T) {
		c := NewSyncCache[rec]()
		_, err := c.Get(t.Context(), StrKey("a"), newRec(1))
		require.NoError(t, err)

		got, err := c.Update(t.Context(), StrKey("a"), func(context.Context, *rec) (*rec, error) { return nil, nil })
		require.NoError(t, err)
		assert.Nil(t, got)
		assert.Equal(t, 0, c.Len())
	})
}

func TestSyncCache_DeleteAndDeleter(t *testing.T) {
	c := NewSyncCache[rec]()

	_, err := c.Get(t.Context(), StrKey("a"), newRec(1))
	require.NoError(t, err)
	_, err = c.Get(t.Context(), StrKey("b"), newRec(2))
	require.NoError(t, err)

	require.NoError(t, c.Delete(t.Context(), StrKey("a")))
	require.NoError(t, c.Delete(t.Context(), StrKey("missing")), "deleting an absent key is a no-op")
	assert.Equal(t, 1, c.Len())

	del := c.Deleter()
	require.NoError(t, del(t.Context(), StrKey("b")))
	assert.Equal(t, 0, c.Len())
}

func TestSyncCache_DeleteFunc(t *testing.T) {
	c := NewSyncCache[rec]()
	for i, k := range []string{"a", "b", "c"} {
		_, err := c.Get(t.Context(), StrKey(k), newRec(i))
		require.NoError(t, err)
	}

	n := c.DeleteFunc(func(r *rec) bool { return r.n != 1 })
	assert.Equal(t, 2, n)
	assert.Equal(t, 1, c.Len())

	kept, err := c.Get(t.Context(), StrKey("b"), failNew)
	require.NoError(t, err)
	assert.Equal(t, 1, kept.n)

	assert.Equal(t, 0, c.DeleteFunc(func(*rec) bool { return false }))
}

func TestSyncCache_Cap(t *testing.T) {
	t.Run("stays within cap", func(t *testing.T) {
		c := NewSyncCacheCap[rec](2)
		for i, k := range []string{"a", "b", "c", "d"} {
			_, err := c.Get(t.Context(), StrKey(k), newRec(i))
			require.NoError(t, err)
		}
		assert.Equal(t, 2, c.Len())

		// The newest record always survives its own insert.
		got, err := c.Get(t.Context(), StrKey("d"), failNew)
		require.NoError(t, err)
		assert.Equal(t, 3, got.n)
	})

	t.Run("evicted record is rebuilt on demand", func(t *testing.T) {
		c := NewSyncCacheCap[rec](1)
		_, err := c.Get(t.Context(), StrKey("a"), newRec(1))
		require.NoError(t, err)
		_, err = c.Get(t.Context(), StrKey("b"), newRec(2))
		require.NoError(t, err)

		got, err := c.Get(t.Context(), StrKey("a"), newRec(3)) // "a" was evicted
		require.NoError(t, err)
		assert.Equal(t, 3, got.n)
	})

	t.Run("update inserts respect the cap", func(t *testing.T) {
		c := NewSyncCacheCap[rec](1)
		_, err := c.Get(t.Context(), StrKey("a"), newRec(1))
		require.NoError(t, err)

		got, err := c.Update(t.Context(), StrKey("b"), func(context.Context, *rec) (*rec, error) {
			return &rec{n: 2}, nil
		})
		require.NoError(t, err)
		assert.Equal(t, 2, got.n)
		assert.Equal(t, 1, c.Len())
	})

	t.Run("settled records are evicted before in-flight creates", func(t *testing.T) {
		c := NewSyncCacheCap[rec](2)
		release := startBlockedGet(t, c, "flight", &rec{n: 1}, nil)
		defer release()
		_, err := c.Get(t.Context(), StrKey("a"), newRec(2))
		require.NoError(t, err)

		_, err = c.Get(t.Context(), StrKey("b"), newRec(3)) // evicts "a", not the flight
		require.NoError(t, err)
		assert.Equal(t, 2, c.Len())

		release()
		got, err := c.Get(t.Context(), StrKey("flight"), failNew)
		require.NoError(t, err)
		assert.Equal(t, 1, got.n, "the in-flight create kept its slot")
	})

	t.Run("non-positive cap is unbounded", func(t *testing.T) {
		c := NewSyncCacheCap[rec](0)
		for i, k := range []string{"a", "b", "c"} {
			_, err := c.Get(t.Context(), StrKey(k), newRec(i))
			require.NoError(t, err)
		}
		assert.Equal(t, 3, c.Len())
	})
}

func TestSyncCache_ConcurrentGet(t *testing.T) {
	c := NewSyncCache[rec]()
	var created atomic.Int32

	const goroutines = 64
	var wg sync.WaitGroup
	results := make([]*rec, goroutines)

	for i := range goroutines {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r, err := c.Get(t.Context(), StrKey("shared"), func(context.Context) (*rec, error) {
				created.Add(1)
				return &rec{n: 1}, nil
			})
			assert.NoError(t, err)
			results[i] = r
		}()
	}
	wg.Wait()

	assert.Equal(t, int32(1), created.Load(), "record must be created exactly once")
	for _, r := range results {
		assert.Same(t, results[0], r)
	}
}

// TestSyncCache_ConcurrentMutation mixes Get, Update and Delete on the same
// key; it exists for the race detector and only checks invariants.
func TestSyncCache_ConcurrentMutation(t *testing.T) {
	c := NewSyncCache[rec]()
	key := StrKey("k")

	const goroutines = 32
	var wg sync.WaitGroup
	for i := range goroutines {
		wg.Add(1)
		go func() {
			defer wg.Done()
			switch i % 3 {
			case 0:
				_, err := c.Get(t.Context(), key, newRec(i))
				assert.NoError(t, err)
			case 1:
				_, err := c.Update(t.Context(), key, func(_ context.Context, cur *rec) (*rec, error) {
					if cur == nil {
						return &rec{n: i}, nil
					}
					return &rec{n: cur.n + 1}, nil
				})
				assert.NoError(t, err)
			default:
				assert.NoError(t, c.Delete(t.Context(), key))
			}
		}()
	}
	wg.Wait()

	assert.LessOrEqual(t, c.Len(), 1)
}

// startBlockedGet runs a Get for key whose factory blocks until the returned
// release func is called. It returns once the factory is running.
func startBlockedGet(t *testing.T, c *SyncCache[rec], key string, result *rec, err error) (release func()) {
	t.Helper()
	gate := make(chan struct{})
	started := make(chan struct{})
	go func() {
		_, _ = c.Get(t.Context(), StrKey(key), func(context.Context) (*rec, error) {
			close(started)
			<-gate
			return result, err
		})
	}()
	<-started
	return sync.OnceFunc(func() { close(gate) })
}

// TestSyncCache_KeysDoNotBlockEachOther is the per-key singleflight property:
// a factory stuck on key A must not delay a Get for key B.
func TestSyncCache_KeysDoNotBlockEachOther(t *testing.T) {
	c := NewSyncCache[rec]()
	release := startBlockedGet(t, c, "slow", &rec{n: 1}, nil)
	defer release()

	done := make(chan struct{})
	go func() {
		defer close(done)
		got, err := c.Get(t.Context(), StrKey("fast"), newRec(2))
		assert.NoError(t, err)
		assert.Equal(t, 2, got.n)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Get for an unrelated key blocked behind an in-flight factory")
	}
}

func TestSyncCache_WaitersShareFailure(t *testing.T) {
	c := NewSyncCache[rec]()
	release := startBlockedGet(t, c, "k", nil, errBoom)

	var calls atomic.Int32
	var wg sync.WaitGroup
	for range 4 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := c.Get(t.Context(), StrKey("k"), func(context.Context) (*rec, error) {
				calls.Add(1)
				return nil, errBoom
			})
			assert.ErrorIs(t, err, errBoom)
		}()
	}
	time.Sleep(20 * time.Millisecond) // let the waiters join the flight
	release()
	wg.Wait()

	assert.Equal(t, int32(0), calls.Load(), "waiters must not start their own flight")
	assert.Equal(t, 0, c.Len(), "failed placeholder must be removed")

	got, err := c.Get(t.Context(), StrKey("k"), newRec(7)) // retry works
	require.NoError(t, err)
	assert.Equal(t, 7, got.n)
}

func TestSyncCache_NilRecordIsError(t *testing.T) {
	c := NewSyncCache[rec]()

	_, err := c.Get(t.Context(), StrKey("nil"), func(context.Context) (*rec, error) { return nil, nil })
	require.ErrorIs(t, err, ErrNilRecord)
	assert.Equal(t, 0, c.Len())
}

func TestSyncCache_PanicReleasesWaiters(t *testing.T) {
	c := NewSyncCache[rec]()
	started := make(chan struct{})
	gate := make(chan struct{})

	go func() {
		defer func() { _ = recover() }()
		_, _ = c.Get(t.Context(), StrKey("boom"), func(context.Context) (*rec, error) {
			close(started)
			<-gate
			panic("factory")
		})
	}()
	<-started

	waited := make(chan error, 1)
	go func() {
		_, err := c.Get(t.Context(), StrKey("boom"), failNew)
		waited <- err
	}()
	time.Sleep(20 * time.Millisecond) // let the waiter join the flight
	close(gate)

	select {
	case err := <-waited:
		require.ErrorIs(t, err, ErrNilRecord)
	case <-time.After(2 * time.Second):
		t.Fatal("waiter was not released by the panicking flight")
	}
	assert.Equal(t, 0, c.Len(), "a panicking factory must not leave a placeholder")
}

func TestSyncCache_DeleteDropsInFlightCreate(t *testing.T) {
	c := NewSyncCache[rec]()
	release := startBlockedGet(t, c, "k", &rec{n: 1}, nil)
	require.Equal(t, 1, c.Len(), "in-flight entry is counted")

	waiter := make(chan *rec, 1)
	go func() {
		got, _ := c.Get(t.Context(), StrKey("k"), failNew)
		waiter <- got
	}()
	time.Sleep(20 * time.Millisecond) // let the waiter join the flight

	require.NoError(t, c.Delete(t.Context(), StrKey("k")))
	assert.Equal(t, 0, c.Len())

	// A Get after the Delete starts a fresh flight rather than joining the old one.
	fresh, err := c.Get(t.Context(), StrKey("k"), newRec(2))
	require.NoError(t, err)
	assert.Equal(t, 2, fresh.n)

	release()
	assert.Equal(t, 1, (<-waiter).n, "waiter still receives the old flight's result")
	got, _ := c.Get(t.Context(), StrKey("k"), failNew)
	assert.Equal(t, 2, got.n, "the old flight does not overwrite the fresh record")
}

func TestSyncCache_UpdateWaitsForInFlightCreate(t *testing.T) {
	c := NewSyncCache[rec]()
	release := startBlockedGet(t, c, "k", &rec{n: 1}, nil)

	updated := make(chan *rec, 1)
	go func() {
		got, err := c.Update(t.Context(), StrKey("k"), func(_ context.Context, cur *rec) (*rec, error) {
			if cur == nil {
				return &rec{n: -1}, nil // would mean Update raced the create
			}
			return &rec{n: cur.n + 1}, nil
		})
		assert.NoError(t, err)
		updated <- got
	}()

	select {
	case <-updated:
		t.Fatal("Update must wait for the in-flight create")
	case <-time.After(50 * time.Millisecond):
	}

	release()
	got := <-updated
	assert.Equal(t, 2, got.n, "Update saw the created record")
}

func ExampleSyncCache() {
	type session struct{ user string }

	c := NewSyncCache[session]()
	load := func(context.Context) (*session, error) {
		fmt.Println("loading")
		return &session{user: "ada"}, nil
	}

	s, _ := c.Get(context.Background(), StrKey("sess-1"), load)
	fmt.Println(s.user)
	s, _ = c.Get(context.Background(), StrKey("sess-1"), load) // cached
	fmt.Println(s.user)
	// Output:
	// loading
	// ada
	// ada
}
