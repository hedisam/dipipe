package master

import "sync"

// threadSafeIdlesQueue implements a thread-safe queue upon a normal one.
type threadSafeIdlesQueue struct {
	cond *sync.Cond
	q IdlesQueue
}

func newThreadSafeIdlesQueue(q IdlesQueue) *threadSafeIdlesQueue {
	return &threadSafeIdlesQueue{
		cond: sync.NewCond(&sync.Mutex{}),
		q: q,
	}
}

// Enqueue a worker in a thread-safe way.
func (t *threadSafeIdlesQueue) Enqueue(w Worker) {
	t.cond.L.Lock()
	// signal to the first waiting goroutine
	defer t.cond.Signal()
	defer t.cond.L.Unlock()
	t.q.Enqueue(w)
}

// Dequeue blocks until it gets an idle worker.
func (t *threadSafeIdlesQueue) Dequeue() (w Worker) {
	t.cond.L.Lock()
	defer t.cond.L.Unlock()

	for w = t.q.Dequeue(); w == nil; w = t.q.Dequeue() {
		// wait for a signal
		t.cond.Wait()
	}
	return w
}

// TryDequeue tries to get an idle worker, it returns nil if the queue is empty.
func (t *threadSafeIdlesQueue) TryDequeue() Worker {
	t.cond.L.Lock()
	defer t.cond.L.Unlock()

	return t.q.Dequeue()
}
