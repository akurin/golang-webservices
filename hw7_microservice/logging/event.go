package logging

import "time"

type Event struct {
	Timestamp time.Time
	Consumer  string
	Method    string
	Host      string
}
