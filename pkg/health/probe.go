package health

import (
	"gotunnel/pkg/log"
	"net"
	"time"
)

// ProbeTCPAlive checks the liveness of a local port, timeout is configurable
func ProbeTCPAlive(addr string, timeout time.Duration) bool {
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err == nil {
		_ = conn.Close()
		return true
	}
	return false
}

// PeriodicProbe periodically probes a local port, can notify main process to go offline when down is detected
func PeriodicProbe(target string, interval time.Duration, onDead func(), onAlive func()) {
	aliveLast := true
	for {
		ok := ProbeTCPAlive(target, time.Second)
		if ok {
			if !aliveLast && onAlive != nil {
				onAlive()
			}
			aliveLast = true
			log.Debugf("health", "health.port_healthy", target)
		} else {
			if aliveLast && onDead != nil {
				onDead()
			}
			aliveLast = false
			log.Warnf("health", "health.port_unreachable", target)
		}
		time.Sleep(interval)
	}
}

// Example usage:
// health.PeriodicProbe("127.0.0.1:2222", 30*time.Second, func(){...}, func(){...})
