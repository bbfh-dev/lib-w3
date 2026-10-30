package internal

import (
	"sync"
)

// Pool is a fixed-size object pool with a dense stack free-list.
// Refer to [NewPool].
type Pool[T any] struct {
	objects []T
	free    []uint16
	top     uint16

	mu        sync.Mutex
	available chan struct{}
}

// Create a new [Pool] with specified size and object factory.
func NewPool[T any](n uint16, factory func() T) *Pool[T] {
	if n == 0 {
		panic("pool size must be greater than zero")
	}

	objects := make([]T, int(n))
	free := make([]uint16, int(n))

	var useFactory = factory != nil
	for i := range n {
		if useFactory {
			objects[i] = factory()
		}
		free[i] = i
	}

	available := make(chan struct{}, int(n))

	for range n {
		available <- struct{}{}
	}

	return &Pool[T]{
		objects:   objects,
		free:      free,
		top:       n,
		available: available,
	}
}

// Acquire an index to an available object.
// Waits when no object is available.
func (pool *Pool[T]) Acquire() (index uint16) {
	<-pool.available

	pool.mu.Lock()

	pool.top--
	index = pool.free[pool.top]

	pool.mu.Unlock()
	return index
}

// AcquireObject is sugar code for calling [Pool.Acquire] and [Pool.Get].
func (pool *Pool[T]) AcquireObject() (index uint16, object *T) {
	index = pool.Acquire()
	return index, pool.Get(index)
}

// Release an object back into the pool.
func (pool *Pool[T]) Release(index uint16) {
	pool.mu.Lock()

	pool.free[pool.top] = index
	pool.top++

	pool.mu.Unlock()

	pool.available <- struct{}{}
}

// Get the reference to an object.
// Not to be mistaken with [Pool.Acquire].
func (pool *Pool[T]) Get(index uint16) *T {
	return &pool.objects[index]
}
