package spscqueue

import (
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
}

func TestPutGetLarge(t *testing.T) {
	const numItems = 8000
	q := New[int](numItems)

	for i := 0; i < numItems; i++ {
		q.Put(i)
	}

	for i := 0; i < numItems; i++ {
		v := q.Poll()
		if v != i {
			t.Errorf("Got incorrect value; %v != %v", v, i)
		}
		q.Advance()
	}
}

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