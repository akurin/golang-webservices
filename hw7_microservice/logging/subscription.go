package logging

type Subscription struct {
	Events chan Event
}

func (s Subscription) Dispose() {
	close(s.Events)
}
