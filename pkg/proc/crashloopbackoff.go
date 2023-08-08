package proc

import (
	"time"
)

// calculates the next backOff based on the current backOff
func calculateNextBackOff(currBackOff, maxBackoff time.Duration) time.Duration {
	if currBackOff.Seconds() <= 1 {
		return 2 * time.Second
	}
	next := time.Duration(2*currBackOff.Seconds()) * time.Second
	if next.Seconds() > maxBackoff.Seconds() {
		return maxBackoff
	}
	return next
}
