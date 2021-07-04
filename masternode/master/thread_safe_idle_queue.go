package master

import (
	"context"
	"sync"
	"sync/atomic"
)

// threadSafeIdlesQueue implements a thread-safe queue upon a normal one.
type threadSafeIdlesQueue struct {
	// cond is used to signal the waiting goroutines
	cond *sync.Cond
	// q is the underlying queue
	q IdlesQueue
	// ctx is used to cancel out the waiting goroutines. a sync.Cond is not cancellable, therefore we utilize a
	// global (struct-level) context object
	ctx context.Context
	// state is set to 1 if the context object has been cancelled. we could've determine a disposed state by
	// checking the context's done channel but atomic operations are way faster.
	state int32
}

func newThreadSafeIdlesQueue(ctx context.Context, q IdlesQueue) *threadSafeIdlesQueue {
	tsQ := &threadSafeIdlesQueue{
		cond: sync.NewCond(&sync.Mutex{}),
		q: q,
		ctx: ctx,
	}
	// monitor the context object and dispose the queue if it got cancelled
	go tsQ.monitorContext()

	return tsQ
}

func (t *threadSafeIdlesQueue) monitorContext() {
	// block until the context's done channel gets closed
	<-t.ctx.Done()

	// dispose the queue before emitting the signal
	t.dispose()

	// the context is cancelled so we need to dispose the queue. we broadcast a signal to all the waiting goroutines
	// so they exit the sync.Cond.Wait blocking method and get the chance to notice the disposed state of the queue.
	t.cond.Broadcast()
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
func (t *threadSafeIdlesQueue) Dequeue() (w Worker, disposed bool) {
	t.cond.L.Lock()
	defer t.cond.L.Unlock()

	for {
		if t.disposed() {
			return nil, true
		}
		w = t.q.Dequeue()
		if w != nil {
			break
		}
		// wait for a signal. it could be a signal sent by the Enqueue method or it could be from a cancelled context
		// object which wants all the waiting goroutines to stop and return, in that case the queue would disposed.
		t.cond.Wait()
	}

	return w, false
}

// TryDequeue tries to get an idle worker, it returns nil if the queue is empty.
func (t *threadSafeIdlesQueue) TryDequeue() Worker {
	if t.disposed() {
		return nil
	}

	t.cond.L.Lock()
	defer t.cond.L.Unlock()

	return t.q.Dequeue()
}

func (t *threadSafeIdlesQueue) disposed() bool {
	// a value of 1 means a disposed state
	return atomic.LoadInt32(&t.state) == 1
}

func (t *threadSafeIdlesQueue) dispose() {
	atomic.StoreInt32(&t.state, 1)
}
