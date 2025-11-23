package ha

import (
	"gotunnel/pkg/log"
	"math/rand"
	"time"
)

// sleepHook can be replaced in unit tests to control duration/efficiency
var sleepHook = time.Sleep

// ReconnectLoop automatically reconnects using exponential backoff + random jitter.
// dialFunc is a function that attempts to establish a connection and returns whether it succeeded.
// baseInterval, maxInterval: base and maximum backoff interval (seconds)
// maxTries: maximum number of attempts, 0 means unlimited retries.
// Returns true if connection is successfully established, false if it ultimately fails.
func ReconnectLoop(dialFunc func() bool, baseInterval, maxInterval int, maxTries int) bool {
	tries := 0
	interval := baseInterval
	for {
		if maxTries > 0 && tries >= maxTries {
			log.Errorf("ha", "ha.reconnect_max_tries", maxTries)
			return false
		}
		if dialFunc() {
			return true
		}
		d := time.Duration(interval) * time.Second
		jitter := time.Duration(rand.Intn(1000)) * time.Millisecond
		log.Warnf("ha", "ha.reconnect_retry", tries+1, d+jitter)
		sleepHook(d + jitter)
		interval = interval * 2
		if interval > maxInterval {
			interval = maxInterval
		}
		tries++
	}
}

// Example usage can be found in tests.
