## go-spscqueue

This repository contains a type generic single-producer single-consumer bounded queue go package.
The implementation is based on go's generic capabilities aimed for go version 1.18.

> Beware that go's generics infrastructure is not yet stable. This was tested with
`go version go1.18rc1 linux/amd64`.

The algorithm is similar to the C++ implementation described [here](https://rigtorp.se/ringbuffer/) (and available [here](https://github.com/rigtorp/SPSCQueue)).
It is lock-free and tries to be cache-friendly.

### Usage

```
q := spscqueue.New[int](16)

// The producer may call these functions:
q.Push(1)        // Add 1 to the back of the queue.
ok := q.Offer(2) // Try to add 2 to the back of the queue.
fmt.Println(q.Len())

// The consumer may call these:
v := q.Pop()        // Retrieve and remove the item at the front of the queue.
v2, ok := q.Front() // Try to retrieve the item at the front of the queue.
if ok {
	q.Advance() // Remove the item at the front of the queue.
}
fmt.Println(q.Len())
```

The combination of `Front()` and `Advance()` achieves the same as `Pop()`. Using them allows for
more fine grained control as to when the particular queue slot is marked as available for re-use by
the producer.

### Installation

```
go get github.com/fjbotha/go-spscqueue/spscqueue
```
