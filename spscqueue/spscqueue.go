package spscqueue

import (
	"golang.org/x/sys/cpu"
	"runtime"
	"sync/atomic"
)

// Queue is the structure responsible for tracking the state of the bounded single-producer
// single-consumer queue.
type Queue[T any] struct {
	// Relevant struct elements are spaced out to separate cache lines, so as to prevent false
	// sharing/cache line invalidation.
	_          cpu.CacheLinePad
	items      []T
	_          cpu.CacheLinePad
	rIdx       uint64
	wIdxCached uint64
	_          cpu.CacheLinePad
	wIdx       uint64
	rIdxCached uint64
	_          cpu.CacheLinePad
}

// New[T any] returns an empty single-producer single-consumer bounded queue. The queue has capacity
// for `size` elements of type `T`.
func New[T any](size uint) *Queue[T] {
	return &Queue[T]{items: make([]T, size+1)}
}

// Push adds the passed element to the queue. Push will block if the queue is full.
// Push should be called by the producer.
func (q *Queue[T]) Push(el T) {
	wIdxNext := q.wIdx + 1
	if wIdxNext == uint64(len(q.items)) {
		wIdxNext = 0
	}

	// Wait if we ran into the consumer.
	if wIdxNext == q.rIdxCached {
		q.rIdxCached = atomic.LoadUint64(&q.rIdx)
		for wIdxNext == q.rIdxCached {
			runtime.Gosched()
			q.rIdxCached = atomic.LoadUint64(&q.rIdx)
		}
	}
	q.items[q.wIdx] = el
	atomic.StoreUint64(&q.wIdx, wIdxNext)
}

// Offer adds the passed element to the queue if there is an available slot. Offer returns true if
// the item was added successfully, otherwise false.
// Offer should be called by the producer.
func (q *Queue[T]) Offer(el T) bool {
	wIdxNext := q.wIdx + 1
	if wIdxNext == uint64(len(q.items)) {
		wIdxNext = 0
	}

	// Check if we ran into the consumer.
	if wIdxNext == q.rIdxCached {
		q.rIdxCached = atomic.LoadUint64(&q.rIdx)
		if wIdxNext == q.rIdxCached {
			return false
		}
	}
	q.items[q.wIdx] = el
	atomic.StoreUint64(&q.wIdx, wIdxNext)
	return true
}

// Pop returns the oldest element in the queue and removes it. Pop will block if no element is
// available.
// Pop should be called by the consumer.
func (q *Queue[T]) Pop() T {
	// Wait for an item to be available.
	if q.rIdx == q.wIdxCached {
		q.wIdxCached = atomic.LoadUint64(&q.wIdx)
		for q.rIdx == q.wIdxCached {
			runtime.Gosched()
			q.wIdxCached = atomic.LoadUint64(&q.wIdx)
		}
	}

	ret := q.items[q.rIdx]
	q.Advance()
	return ret
}

// Front is a non-blocking variant of Pop. It returns the oldest element in the queue if the queue
// is not empty, otherwise the zero-value for the type. A boolean indicator of success or failure is
// included as a second return value. Contrary to Pop, subsequent calls to Front without a call to
// Advance will return the same element.
// Front should be called by the consumer.
func (q *Queue[T]) Front() (T, bool) {
	// Check if an item is available.
	if q.rIdx == q.wIdxCached {
		q.wIdxCached = atomic.LoadUint64(&q.wIdx)
		if q.rIdx == q.wIdxCached {
			var t T
			return t, false
		}
	}

	return q.items[q.rIdx], true
}

// Advance moves the consumer forward. Advance may be called after using the data returned from
// Front.
// Advance should be called by the consumer if and only if it follows a successful call to Front.
func (q *Queue[T]) Advance() {
	rIdxNext := q.rIdx + 1
	if rIdxNext == uint64(len(q.items)) {
		rIdxNext = 0
	}
	atomic.StoreUint64(&q.rIdx, rIdxNext)
}

// Len returns the number of elements in the queue.
// Any thread may call Len.
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
