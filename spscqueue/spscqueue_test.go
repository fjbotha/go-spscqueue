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
	q.Push(1)
	if l := q.Len(); l != 1 {
		t.Errorf("Unexpected length; %v != 1", l)
	}
	q.Push(2)
	if l := q.Len(); l != 2 {
		t.Errorf("Unexpected length; %v != 2", l)
	}
	for i := 3; i <= 8; i++ {
		q.Push(i)
	}
	if l := q.Len(); l != 8 {
		t.Errorf("Unexpected length; %v != 8", l)
	}

	q.Pop()
	if l := q.Len(); l != 7 {
		t.Errorf("Unexpected length; %v != 7", l)
	}
	q.Pop()
	if l := q.Len(); l != 6 {
		t.Errorf("Unexpected length; %v != 6", l)
	}

	q.Push(11)
	if l := q.Len(); l != 7 {
		t.Errorf("Unexpected length; %v != 7", l)
	}
	q.Push(12)
	if l := q.Len(); l != 8 {
		t.Errorf("Unexpected length; %v != 8", l)
	}

	for i := 1; i <= 8; i++ {
		q.Pop()
		if l := int(q.Len()); l != 8-i {
			t.Errorf("Unexpected length; %v != %v", l, 8-i)
			t.Logf("rIdx = %v; wIdx = %v", q.rIdx, q.wIdx)
		}
	}
}

// Single threaded test.
func TestPushPop(t *testing.T) {
	q := New[int](8)

	for i := 0; i < 8; i++ {
		q.Push(i)
	}

	for i := 0; i < 8; i++ {
		v := q.Pop()
		if v != i {
			t.Errorf("Got incorrect value; %v != %v", v, i)
		}
	}

	for i := 10; i < 18; i++ {
		q.Push(i)
	}

	for i := 10; i < 18; i++ {
		v := q.Pop()
		if v != i {
			t.Errorf("Got incorrect value; %v != %v", v, i)
		}
	}

	const bigQSize = 8000
	bigQ := New[int](bigQSize)
	for i := 0; i < bigQSize; i++ {
		bigQ.Push(i)
	}

	for i := 0; i < bigQSize; i++ {
		v := bigQ.Pop()
		if v != i {
			t.Errorf("Got incorrect value; %v != %v", v, i)
		}
	}
}

// SPSC test.
func TestPushPopSPSC(t *testing.T) {
	const numItems = 10000
	q := New[int](64)
	wg := sync.WaitGroup{}

	wg.Add(1)
	go func(wg *sync.WaitGroup) {
		defer wg.Done()
		for i := 0; i < numItems; i++ {
			q.Push(i)
		}
	}(&wg)

	wg.Add(1)
	go func(wg *sync.WaitGroup) {
		defer wg.Done()
		for i := 0; i < numItems; i++ {
			v := q.Pop()
			if v != i {
				t.Errorf("Got incorrect value; %v != %v", v, i)
			}
		}
	}(&wg)

	wg.Wait()
}

func TestOfferPeekAdvanceSPSC(t *testing.T) {
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
			v, ok := q.Front()
			for !ok {
				runtime.Gosched()
				v, ok = q.Front()
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
			q.Push(strconv.Itoa(i))
		}
	}(&wg)

	wg.Add(1)
	go func(wg *sync.WaitGroup) {
		defer wg.Done()
		for i := 0; i < numItems; i++ {
			v := q.Pop()
			if v != strconv.Itoa(i) {
				t.Errorf("Got incorrect value; %v != %v", v, i)
			}
		}
	}(&wg)

	wg.Wait()
}

// SPSC test for the Reserve-Commit pattern.
func TestReserveCommitSPSC(t *testing.T) {
	const numItems = 10000
	q := New[*int](64)
	wg := sync.WaitGroup{}

	{
		// Allocate the underlying queue storage.
		q.Fill(func() *int { var v int; return &v })
	}

	wg.Add(1)
	go func(wg *sync.WaitGroup) {
		defer wg.Done()
		for i := 0; i < numItems; i++ {
			v, ok := q.Reserve()
			for !ok {
				v, ok = q.Reserve()
			}
			*v = i
			q.Commit()
		}
	}(&wg)

	wg.Add(1)
	go func(wg *sync.WaitGroup) {
		defer wg.Done()
		for i := 0; i < numItems; i++ {
			v := q.Pop()
			if *v != i {
				t.Errorf("Got incorrect value; %v != %v", *v, i)
			}
		}
	}(&wg)

	wg.Wait()
}

// Single threaded benchmark; not the primary usecase.
func BenchmarkPushPop(b *testing.B) {
	q := New[int](1)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		q.Push(i)
		_ = q.Pop()
	}
}

// Channel reference single threaded benchmark.
func BenchmarkChannelPushPop(b *testing.B) {
	q := make(chan int, 1)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		q <- i
		_ = <-q
	}
}

// SPSC benchmark.
func BenchmarkPushPopSPSC(b *testing.B) {
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
			q.Push(i)
		}
	}(&wg)

	wg.Add(1)
	go func(wg *sync.WaitGroup) {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		defer wg.Done()
		<-start
		for i := 0; i < b.N; i++ {
			_ = q.Pop()
		}
	}(&wg)

	b.ReportAllocs()
	b.ResetTimer()
	close(start)

	wg.Wait()
}

// SPSC benchmark.
func BenchmarkOfferPeekAdvanceSPSC(b *testing.B) {
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
			for q.Offer(i) == false {
				runtime.Gosched()
			}
		}
	}(&wg)

	wg.Add(1)
	go func(wg *sync.WaitGroup) {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		defer wg.Done()
		<-start
		for i := 0; i < b.N; i++ {
			_, ok := q.Front()
			for !ok {
				runtime.Gosched()
				_, ok = q.Front()
			}
			q.Advance()
		}
	}(&wg)

	b.ReportAllocs()
	b.ResetTimer()
	close(start)

	wg.Wait()
}

// Channel reference SPSC benchmark.
func BenchmarkChannelPushPopSPSC(b *testing.B) {
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
