package event

type Queue struct {
	ch chan Event
}

func NewQueue(capacity int) *Queue {
	return &Queue{ch: make(chan Event, capacity)}
}

func (q *Queue) Enqueue(e Event) {
	q.ch <- e
}

func (q *Queue) Dequeue() Event {
	return <-q.ch
}

func (q *Queue) Chan() <-chan Event {
	return q.ch
}
