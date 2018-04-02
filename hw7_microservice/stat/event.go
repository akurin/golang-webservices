package stat

import "time"

type Event struct {
	Timestamp  time.Time
	ByConsumer map[string]uint64
	ByMethod   map[string]uint64
}
