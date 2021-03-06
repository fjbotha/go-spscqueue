package spscqueue

import (
	"runtime"
	"sync/atomic"

	"golang.org/x/sys/cpu"
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

func (q *Queue[T]) Fill(f func() T) {
	for i := 0; i < len(q.items); i++ {
		q.items[i] = f()
	}
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

// Reserve returns the underlying element which the next Push operation will overwrite, i.e. the
// next open slot at the back of the queue. Data retrieved through Reserve can be made available to
// the consumer using Commit. The Reserve-Commit pattern can be used to work on the pre-allocated
// queue items. Subsequent calls to Reserve without a call to Commit will return the same element.
// Reserve should be called by the producer.
func (q *Queue[T]) Reserve() (T, bool) {
	wIdxNext := q.wIdx + 1
	if wIdxNext == uint64(len(q.items)) {
		wIdxNext = 0
	}

	// Check if we ran into the consumer.
	if wIdxNext == q.rIdxCached {
		q.rIdxCached = atomic.LoadUint64(&q.rIdx)
		if wIdxNext == q.rIdxCached {
			var ret T
			return ret, false
		}
	}

	return q.items[q.wIdx], true
}

// Commit advances the back of the queue. Commit can be used in conjunction with Reserve to work on
// the underlying queue data and present them to the consumer.
// Commit should be called by the producer.
func (q *Queue[T]) Commit() {
	wIdxNext := q.wIdx + 1
	if wIdxNext == uint64(len(q.items)) {
		wIdxNext = 0
	}
	atomic.StoreUint64(&q.wIdx, wIdxNext)
}

// Pop returns the oldest element in the queue and removes it. Pop will block if no element is
// available.
// Pop should be called by the consumer.
func (q *Queue[T]) Pop() T {
	defer q.Advance()
	// Wait for an item to be available.
	rIdx := q.rIdx
	if rIdx == q.wIdxCached {
		q.wIdxCached = atomic.LoadUint64(&q.wIdx)
		for rIdx == q.wIdxCached {
			runtime.Gosched()
			q.wIdxCached = atomic.LoadUint64(&q.wIdx)
		}
	}

	return q.items[rIdx]
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
