package spscqueue

import (
	"runtime"
	"strconv"
	"sync"
	"testing"
)

func TestLength(t *testing.T) {
	q := New[int](8)
	if l := q.Len(); l != 0 {
		t.Errorf("Unexpected length; %v != 0", l)
	}
	q.Put(1)
	if l := q.Len(); l != 1 {
		t.Errorf("Unexpected length; %v != 1", l)
	}
	q.Put(2)
	if l := q.Len(); l != 2 {
		t.Errorf("Unexpected length; %v != 2", l)
	}
	for i := 3; i <= 8; i++ {
		q.Put(i)
	}
	if l := q.Len(); l != 8 {
		t.Errorf("Unexpected length; %v != 8", l)
	}

	q.Advance()
	if l := q.Len(); l != 7 {
		t.Errorf("Unexpected length; %v != 7", l)
	}
	q.Advance()
	if l := q.Len(); l != 6 {
		t.Errorf("Unexpected length; %v != 6", l)
	}

	q.Put(11)
	if l := q.Len(); l != 7 {
		t.Errorf("Unexpected length; %v != 7", l)
	}
	q.Put(12)
	if l := q.Len(); l != 8 {
		t.Errorf("Unexpected length; %v != 8", l)
	}

	for i := 1; i <= 8; i++ {
		q.Advance()
		if l := int(q.Len()); l != 8-i {
			t.Errorf("Unexpected length; %v != %v", l, 8-i)
			t.Logf("rIdx = %v; wIdx = %v", q.rIdx, q.wIdx)
		}
	}
}

// Single threaded test.
func TestPutGet(t *testing.T) {
	q := New[int](8)

	for i := 0; i < 8; i++ {
		q.Put(i)
	}

	for i := 0; i < 8; i++ {
		v := q.Poll()
		if v != i {
			t.Errorf("Got incorrect value; %v != %v", v, i)
		}
		q.Advance()
	}

	for i := 10; i < 18; i++ {
		q.Put(i)
	}

	for i := 10; i < 18; i++ {
		v := q.Poll()
		if v != i {
			t.Errorf("Got incorrect value; %v != %v", v, i)
		}
		q.Advance()
	}

	const bigQSize = 8000
	bigQ := New[int](bigQSize)
	for i := 0; i < bigQSize; i++ {
		bigQ.Put(i)
	}

	for i := 0; i < bigQSize; i++ {
		v := bigQ.Poll()
		if v != i {
			t.Errorf("Got incorrect value; %v != %v", v, i)
		}
		bigQ.Advance()
	}
}

// SPSC test.
func TestPutGetSPSC(t *testing.T) {
	const numItems = 10000
	q := New[int](64)
	wg := sync.WaitGroup{}

	wg.Add(1)
	go func(wg *sync.WaitGroup) {
		defer wg.Done()
		for i := 0; i < numItems; i++ {
			q.Put(i)
		}
	}(&wg)

	wg.Add(1)
	go func(wg *sync.WaitGroup) {
		defer wg.Done()
		for i := 0; i < numItems; i++ {
			v := q.Poll()
			if v != i {
				t.Errorf("Got incorrect value; %v != %v", v, i)
			}
			q.Advance()
		}
	}(&wg)

	wg.Wait()
}

func TestOfferPeekSPSC(t *testing.T) {
	q := New[int](0)

	if q.Offer(1) == true {
		t.Error("Managed to add element to empty queue!")
	}

	const numItems = 10000
	q = New[int](64)
	wg := sync.WaitGroup{}

	wg.Add(1)
	go func(wg *sync.WaitGroup) {
		defer wg.Done()
		for i := 0; i < numItems; i++ {
			for q.Offer(i) == false {
				runtime.Gosched()
			}
		}
	}(&wg)

	wg.Add(1)
	go func(wg *sync.WaitGroup) {
		defer wg.Done()
		for i := 0; i < numItems; i++ {
			v, ok := q.Peek()
			for !ok {
				runtime.Gosched()
				v, ok = q.Peek()
			}
			if v != i {
				t.Errorf("Got incorrect value; %v != %v", v, i)
			}
			q.Advance()
		}
	}(&wg)

	wg.Wait()
}

// SPSC test for a string type.
func TestString(t *testing.T) {
	const numItems = 10000
	q := New[string](64)
	wg := sync.WaitGroup{}

	wg.Add(1)
	go func(wg *sync.WaitGroup) {
		defer wg.Done()
		for i := 0; i < numItems; i++ {
			q.Put(strconv.Itoa(i))
		}
	}(&wg)

	wg.Add(1)
	go func(wg *sync.WaitGroup) {
		defer wg.Done()
		for i := 0; i < numItems; i++ {
			v := q.Poll()
			if v != strconv.Itoa(i) {
				t.Errorf("Got incorrect value; %v != %v", v, i)
			}
			q.Advance()
		}
	}(&wg)

	wg.Wait()
}

// Single threaded benchmark; not the primary usecase.
func BenchmarkPutGet(b *testing.B) {
	q := New[int](1)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		q.Put(i)
		_ = q.Poll()
		q.Advance()
	}
}

// Channel reference single threaded benchmark.
func BenchmarkChannelPutGet(b *testing.B) {
	q := make(chan int, 1)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		q <- i
		_ = <-q
	}
}

// SPSC benchmark.
func BenchmarkPutGetSPSC(b *testing.B) {
	q := New[int](1024)
	start := make(chan struct{})
	wg := sync.WaitGroup{}

	wg.Add(1)
	go func(wg *sync.WaitGroup) {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		defer wg.Done()
		<-start
		for i := 0; i < b.N; i++ {
			q.Put(i)
		}
	}(&wg)

	wg.Add(1)
	go func(wg *sync.WaitGroup) {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		defer wg.Done()
		<-start
		for i := 0; i < b.N; i++ {
			_ = q.Poll()
			q.Advance()
		}
	}(&wg)

	b.ReportAllocs()
	b.ResetTimer()
	close(start)

	wg.Wait()
}

// Channel reference SPSC benchmark.
func BenchmarkChannelPutGetSPSC(b *testing.B) {
	q := make(chan int, 1024)
	start := make(chan struct{})
	wg := sync.WaitGroup{}

	wg.Add(1)
	go func(wg *sync.WaitGroup) {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		defer wg.Done()
		<-start
		for i := 0; i < b.N; i++ {
			q <- i
		}
	}(&wg)

	wg.Add(1)
	go func(wg *sync.WaitGroup) {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		defer wg.Done()
		<-start
		for i := 0; i < b.N; i++ {
			_ = <-q
		}
	}(&wg)

	b.ReportAllocs()
	b.ResetTimer()
	close(start)

	wg.Wait()
}
