package errors

import (
	"gotunnel/pkg/log"
)

// Error codes for gotunnel.
const (
	// ErrConnectFailed indicates a connection failure to the server.
	ErrConnectFailed = 1001
	// ErrAuthFailed indicates an authentication failure.
	ErrAuthFailed = 1002
	// ...more error codes can be extended in the future
)

// Error message keys for i18n
const (
	ErrorKeyConnectFailed = "error.connect_failed"
	ErrorKeyAuthFailed    = "error.auth_failed"
	ErrorKeyUnknown       = "error.unknown"
)

// PrintError provides unified error output (using i18n)
func PrintError(code int, detail error) {
	var key string
	switch code {
	case ErrConnectFailed:
		key = ErrorKeyConnectFailed
	case ErrAuthFailed:
		key = ErrorKeyAuthFailed
	default:
		key = ErrorKeyUnknown
	}

	data := map[string]interface{}{
		"Code": code,
	}
	if detail != nil {
		data["Error"] = detail.Error()
		log.Error("error", key, data)
	} else {
		log.Error("error", key, data)
	}
}
