package ha

import (
	"encoding/json"
	"gotunnel/pkg/log"
	"gotunnel/pkg/protocol"
	"net"
	"time"
)

// HeartbeatManager manages heartbeat sending and monitoring. Used for client and server control channel health.
type HeartbeatManager struct {
	Conn      net.Conn      // Associated network connection
	Interval  time.Duration // Heartbeat packet send interval
	OnTimeout func()        // Timeout disconnect callback
	stopChan  chan struct{}
}

// StartHeartbeat starts a goroutine that periodically sends ping packets. Suitable for clients.
func (h *HeartbeatManager) StartHeartbeat() {
	h.stopChan = make(chan struct{})
	go func() {
		ticker := time.NewTicker(h.Interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				ping := protocol.HeartbeatPing{Type: "ping", Time: time.Now().Unix()}
				b, _ := json.Marshal(ping)
				err := protocol.WritePacket(h.Conn, b)
				if err != nil {
					log.Errorf("ha", "ha.heartbeat_send_failed", err)
					if h.OnTimeout != nil {
						h.OnTimeout()
					}
					return
				}
			case <-h.stopChan:
				return
			}
		}
	}()
}

// StopHeartbeat stops the heartbeat.
func (h *HeartbeatManager) StopHeartbeat() {
	close(h.stopChan)
}

// HeartbeatCheckLoop runs a periodic heartbeat check loop for the server.
// It calls checkFunc every 5 seconds to check client heartbeat status.
func HeartbeatCheckLoop(checkFunc func()) {
	for {
		checkFunc()
		time.Sleep(5 * time.Second)
	}
}

// Example usage:
// cMgr := &ha.HeartbeatManager{Conn: conn, Interval: 10 * time.Second, OnTimeout: func(){...}}
// cMgr.StartHeartbeat() // Client heartbeat
// cMgr.StopHeartbeat() // Stop heartbeat
// ha.HeartbeatCheckLoop(customCheckFunc) // Server heartbeat polling
