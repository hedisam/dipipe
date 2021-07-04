package master

import "sync"

// idleQNode represents a node in the idleQueue.
type idleQNode struct {
	worker Worker
	next   *idleQNode
}

// idleQueue is used by a stageRunner to keep track of and reuse its idle workers.
type idleQueue struct {
	head    *idleQNode
	tail    *idleQNode
	mutex sync.Mutex
}

// newIdleQueue returns a queue for saving idle worker nodes.
func newIdleQueue() *idleQueue {
	return &idleQueue{}
}

func (q *idleQueue) Enqueue(w Worker) {
	q.mutex.Lock()
	defer q.mutex.Unlock()

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

func (q *idleQueue) Dequeue() Worker {
	q.mutex.Lock()
	defer q.mutex.Unlock()

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
