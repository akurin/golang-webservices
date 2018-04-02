package stat

import (
	"time"
)

type Subscription struct {
	calls  chan call
	Events chan Event
	ticker *time.Ticker
}

func (s Subscription) Dispose() {
	s.ticker.Stop()
	close(s.Events)
}
