package time

import "time"

// Every will send the time with a period specified by the duration argument.
// It id equivalent to time.NewTicker(d).C
// It adjusts the intervals or drops ticks to make up for slow receivers.
// The duration d must be greater than zero; if not, NewTicker will panic.
// If efficiency is a concern, use NewTicker and call Ticker.Stop
// if the ticker is no longer needed.
func Every(d time.Duration) <-chan time.Time {
	return time.NewTicker(d).C
}
