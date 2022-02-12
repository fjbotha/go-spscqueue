package spscqueue

import (
	"runtime"
	"sync/atomic"
)

type Queue[T any] struct {
	items      []T
	rIdx       uint64
	wIdxCached uint64
	wIdx       uint64
	rIdxCached uint64
}

// New[T any] returns an empty single-producer single-consumer bounded queue. The queue has capacity
// for `size` elements of type `T`.
func New[T any](size uint) *Queue[T] {
	return &Queue[T]{items: make([]T, size+1)}
}

// Put adds the passed element to the queue. Put will block if the queue is full.
func (q *Queue[T]) Put(el T) {
	// Producer is the only writer to q.wIdx, so it can read it without atomics.
	wIdxNext := q.wIdx + 1
	if wIdxNext == uint64(len(q.items)) {
		wIdxNext = 0
	}

	// Wait if we ran into the consumer.
	for wIdxNext == q.rIdxCached {
		runtime.Gosched()
		q.rIdxCached = atomic.LoadUint64(&q.rIdx)
	}
	q.items[q.wIdx] = el
	// Set atomically, otherwise consumer may read an intermediate result of setting the value.
	atomic.StoreUint64(&q.wIdx, wIdxNext)
}

// Poll returns the oldest element in the queue. Poll will block if no element is available.
// Subsequent calls to Poll without a call to Advance will return the same element.
func (q *Queue[T]) Poll() T {
	// Wait for an item to be available.
	// Consumer is the only writer of q.rIdx, so it can read it without atomics.
	for q.rIdx == q.wIdxCached {
		runtime.Gosched()
		q.wIdxCached = atomic.LoadUint64(&q.wIdx)
	}

	return q.items[q.rIdx]
}

// Advance moves the consumer forward. Advance should be called after using the data returned from
// Poll.
func (q *Queue[T]) Advance() {
	// Consumer is the only writer of q.rIdx, so it can read it without atomics.
	rIdxNext := q.rIdx + 1
	if rIdxNext == uint64(len(q.items)) {
		rIdxNext = 0
	}
	// Set atomically, otherwise producer may read an intermediate result of setting the value.
	atomic.StoreUint64(&q.rIdx, rIdxNext)
}

// Len returns the number of elements in the queue.
func (q *Queue[T]) Len() uint64 {
	rIdx := atomic.LoadUint64(&q.rIdx)
	wIdx := atomic.LoadUint64(&q.wIdx)
	if wIdx == rIdx {
		return 0
	} else if wIdx > rIdx {
		return wIdx - rIdx
	}

	return uint64(len(q.items)) - (rIdx - wIdx)
}
