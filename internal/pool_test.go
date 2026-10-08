package internal_test

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"codeberg.org/bbfh/lib-w3/internal"
)

func TestPool_AcquireRelease(t *testing.T) {
	const n = 10

	pool := internal.NewPool[int](n, nil)
	seen := make(map[uint16]bool, n)

	for range n {
		index := pool.Acquire()

		if seen[index] {
			t.Fatalf("index %d was acquired more than once", index)
		}

		seen[index] = true

		*pool.Get(index) = int(index) * 100
	}

	if len(seen) != n {
		t.Fatalf("expected %d unique indexes, got %d", n, len(seen))
	}

	// All objects are now checked out. The next Acquire must block.
	acquired := make(chan uint16, 1)

	go func() {
		acquired <- pool.Acquire()
	}()

	select {
	case index := <-acquired:
		t.Fatalf("Acquire returned index %d while pool was exhausted", index)

	case <-time.After(100 * time.Millisecond):
		// Expected: Acquire is blocked.
	}

	// Release one object. The blocked Acquire should now complete.
	const releasedIndex = 3

	pool.Release(releasedIndex)

	select {
	case index := <-acquired:
		if index != releasedIndex {
			t.Fatalf("expected released index %d, got %d", releasedIndex, index)
		}

	case <-time.After(time.Second):
		t.Fatal("Acquire did not unblock after Release")
	}
}

func TestPool_Get(t *testing.T) {
	const n = 10

	type object struct {
		value int
	}

	pool := internal.NewPool[object](n, nil)

	index := pool.Acquire()

	obj := pool.Get(index)
	obj.value = 12345

	if got := pool.Get(index).value; got != 12345 {
		t.Fatalf("Get returned wrong object: got %d, want %d", got, 12345)
	}

	pool.Release(index)
}

func TestPool_AllIndexes(t *testing.T) {
	const n = 10

	pool := internal.NewPool[struct{}](n, nil)

	seen := make([]bool, n)

	for range n {
		index := pool.Acquire()

		if int(index) >= n {
			t.Fatalf("Acquire returned invalid index %d", index)
		}

		if seen[index] {
			t.Fatalf("index %d was returned twice before being released", index)
		}

		seen[index] = true
	}

	for i, exists := range seen {
		if !exists {
			t.Fatalf("index %d was never returned", i)
		}
	}
}

func TestPool_Concurrent20(t *testing.T) {
	const (
		poolSize     = 10
		goroutineNum = 20
		workDuration = time.Second
	)

	pool := internal.NewPool[int](poolSize, nil)

	var (
		wg sync.WaitGroup

		active atomic.Int32
		max    atomic.Int32

		acquired = make(chan uint16, goroutineNum)
	)

	wg.Add(goroutineNum)

	for i := range goroutineNum {
		go func(id int) {
			defer wg.Done()

			index := pool.Acquire()

			// No index may be simultaneously owned by two goroutines.
			acquired <- index

			current := active.Add(1)

			// Record maximum concurrency.
			for {
				previous := max.Load()

				if current <= previous {
					break
				}

				if max.CompareAndSwap(previous, current) {
					break
				}
			}

			// Do some dummy work while holding the pool slot.
			time.Sleep(workDuration)

			active.Add(-1)

			pool.Release(index)
		}(i)
	}

	// Collect all acquired indexes. Because the pool has only 10 slots,
	// eventually all 20 goroutines should acquire a slot, but never
	// more than 10 at the same time.
	seen := make(map[uint16]int, poolSize)

	for range goroutineNum {
		index := <-acquired
		seen[index]++
	}

	wg.Wait()

	if got := active.Load(); got != 0 {
		t.Fatalf("expected no active users after Wait, got %d", got)
	}

	gotMax := max.Load()

	if gotMax > poolSize {
		t.Fatalf(
			"pool allowed %d concurrent users, but capacity is %d",
			gotMax,
			poolSize,
		)
	}

	if gotMax != poolSize {
		t.Fatalf(
			"expected pool to reach %d concurrent users, got %d",
			poolSize,
			gotMax,
		)
	}

	// After all releases, every slot should be available again.
	for range poolSize {
		index := pool.Acquire()

		if int(index) >= poolSize {
			t.Fatalf("invalid index after concurrent use: %d", index)
		}

		pool.Release(index)
	}
}

func TestPool_ConcurrentObjectAccess(t *testing.T) {
	const (
		poolSize     = 10
		goroutineNum = 20
		iterations   = 3
	)

	type object struct {
		value int
	}

	pool := internal.NewPool[object](poolSize, nil)

	var wg sync.WaitGroup
	wg.Add(goroutineNum)

	for id := range goroutineNum {
		go func(id int) {
			defer wg.Done()

			for iteration := range iterations {
				index := pool.Acquire()

				obj := pool.Get(index)
				obj.value = id*1000 + iteration

				// Dummy work.
				time.Sleep(10 * time.Millisecond)

				if obj.value != id*1000+iteration {
					t.Errorf(
						"object changed while owned: got %d, want %d",
						obj.value,
						id*1000+iteration,
					)
				}

				pool.Release(index)
			}
		}(id)
	}

	wg.Wait()
}

func TestPool_ZeroSize(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected New(0) to panic")
		}
	}()

	internal.NewPool[int](0, nil)
}
