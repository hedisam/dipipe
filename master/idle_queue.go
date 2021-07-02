package master

import "github.com/Workiva/go-datastructures/queue"

// idleQNode represents a node of the idleQueue.
type idleQNode struct {
	worker Worker
	next *idleQNode
}

// idleQueue is used by a stage to queue its idle worker nodes.
type idleQueue struct {
	ring *queue.RingBuffer
}

// newIdleQueue returns a queue for saving idle worker nodes.
func newIdleQueue(capacity int) *idleQueue {
	if capacity <= 1 {
		// queue.RingBuffer has a bug with a capacity set to 1
		capacity = 2
	}
	return &idleQueue{ring: queue.NewRingBuffer(uint64(capacity))}
}

func (q *idleQueue) Enqueue(w Worker) {
	
}
