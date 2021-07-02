package master

import "context"

// idleQNode represents a node of the idleQueue.
type idleQNode struct {
	worker Worker
	next *idleQNode
}

// idleQueue is used by a stage to queue its idle worker nodes.
type idleQueue struct {
	head    *idleQNode
	tail    *idleQNode
	readCh  chan chan Worker
	writeCh chan Worker
}

// newIdleQueue returns a queue for saving idle worker nodes.
func newIdleQueue(ctx context.Context) *idleQueue {
	q := &idleQueue{
		readCh: make(chan chan Worker),
		writeCh: make(chan Worker),
	}
	q.run(ctx)
	return q
}

func (q *idleQueue) Put(ctx context.Context, w Worker) {
	select {
	case <-ctx.Done(): return
	case q.writeCh <- w:
	}
}

func (q *idleQueue) Get(ctx context.Context) (Worker, bool) {
	ch := make(chan Worker)
	select {
	case <-ctx.Done(): return nil, false
	case q.readCh <- ch:
		select {
		case <-ctx.Done(): return nil, false
		case w := <-ch:
			return w, true
		}
	}
}

func (q *idleQueue) run(ctx context.Context) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case w := <-q.writeCh:
				// enqueue whenever there's a new idle worker node
				q.enqueue(w)
			case ch := <-q.readCh:
				// someone needs an idle worker, let's dequeue an idle node and sent it back through the 'ch' channel
				w := q.dequeue()
				if w == nil {
					// there's no idle workers at the moment so wait for one right here
					select {
					case <-ctx.Done(): return
					case w = <-q.writeCh:
						// there we go, we got our idle worker but we're not gonna enqueue it, we send it through
						// the 'ch' channel to be received by whoever that is waiting on it.
					}
				}
				// send the idle worker
				select {
				case <-ctx.Done(): return
				case ch <- w:
					// now we can close the 'ch' channel
					close(ch)
				}
			}
		}
	}()
}

func (q *idleQueue) enqueue(w Worker) {
	node := &idleQNode{worker: w}
	if q.tail == nil {
		// queue is empty
		q.tail = node
		q.head = node
		return
	}

	q.tail.next = node
	q.tail = node
}

func (q *idleQueue) dequeue() Worker {
	if q.head == nil {
		// queue is empty
		return nil
	}

	node := q.head
	q.head = q.head.next
	if q.head == nil {
		// queue becomes empty with this dequeue
		q.tail = nil
	}

	return node.worker
}
