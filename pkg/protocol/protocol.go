package protocol

import (
	"encoding/binary"
	"errors"
	"gotunnel/pkg/log"
	"io"
)

// HeartbeatPing represents a heartbeat ping packet for the control channel to keep client-server connection alive.
// Server should immediately reply with HeartbeatPong upon receiving.
// Type is fixed: "ping"
type HeartbeatPing struct {
	Type string `json:"type"` // "ping"
	Time int64  `json:"time"` // Optional, current timestamp, can be used for monitoring logs
}

// HeartbeatPong represents a heartbeat pong response packet.
// Type is fixed: "pong"
type HeartbeatPong struct {
	Type string `json:"type"` // "pong"
	Time int64  `json:"time"` // Optional
}

// RegisterRequest represents a control message structure for client port registration (for registering ports that need to be proxied by server).
type RegisterRequest struct {
	Type       string `json:"type"`        // Fixed as "register"
	LocalPort  int    `json:"local_port"`  // Local port on client that needs to be mapped
	RemotePort int    `json:"remote_port"` // Public port on server opened for this mapping
	Protocol   string `json:"protocol"`    // Protocol "tcp"/"http" etc.
	Token      string `json:"token"`       // Authentication token
	Name       string `json:"name"`        // Client custom name
}

// RegisterResponse represents a control message for server registration response, used for confirmation/rejection.
type RegisterResponse struct {
	Type   string `json:"type"`             // Fixed as "register_resp"
	Status string `json:"status"`           // "ok" / "fail"
	Reason string `json:"reason,omitempty"` // Reason for failure
}

// OfflinePortRequest notifies server that a port (remote_port) should go offline, stop public listening and mapping.
// Type: "offline_port"
type OfflinePortRequest struct {
	Type string `json:"type"` // "offline_port"
	Port int    `json:"port"` // remote_port to be removed
}

// OnlinePortRequest notifies server that a port (remote_port) has recovered health and should re-register relay.
// Type: "online_port"
type OnlinePortRequest struct {
	Type string `json:"type"` // "online_port"
	Port int    `json:"port"` // remote_port to be restored
}

// WritePacket writes a complete message to the connection, format: 4-byte payload length (big-endian) + original message content (payload).
// Parameters:
//
//	w: target io.Writer, typically a network connection
//	payload: message content to send (usually json/protobuf encoded result)
//
// Returns:
//
//	error encountered during writing (if any), returns nil on success
func WritePacket(w io.Writer, payload []byte) error {
	// Maximum message length limit to prevent malicious large packet attacks
	if len(payload) > 0x7fffffff {
		log.Errorf("protocol", "error.payload_too_large", len(payload))
		return errors.New("payload too large")
	}
	// Store payload length in 4 bytes big-endian
	var lenBuf [4]byte
	binary.BigEndian.PutUint32(lenBuf[:], uint32(len(payload)))
	// Write packet length first, then payload content
	if _, err1 := w.Write(lenBuf[:]); err1 != nil {
		return err1
	}
	_, err2 := w.Write(payload)
	return err2
}

// ReadPacket reads a complete message from the connection, format requirement same as above (4-byte payload length + actual content).
// Parameters:
//
//	r: source io.Reader, typically a network connection
//
// Returns:
//
//	[]byte: message content
//	error: error encountered during reading or unpacking
func ReadPacket(r io.Reader) ([]byte, error) {
	// Read 4-byte packet length first
	var lenBuf [4]byte
	if _, err1 := io.ReadFull(r, lenBuf[:]); err1 != nil {
		return nil, err1
	}
	length := binary.BigEndian.Uint32(lenBuf[:])
	if length == 0 {
		return nil, nil // Empty payload case
	}
	// Read actual message content according to length
	buf := make([]byte, length)
	if _, err2 := io.ReadFull(r, buf); err2 != nil {
		return nil, err2
	}
	return buf, nil
}

// Core purpose of this file:
//   - Clearly define message boundaries, completely solve TCP packet sticking and splitting issues
//   - Facilitate upper layer custom messages with json/protobuf, implement secure and extensible communication format
//   - This solution is compatible with production environment and local development debugging, future protocol upgrades can directly reuse this transport layer logic
