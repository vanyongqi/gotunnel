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
	// ...后续可扩展更多错误码
)

// Error message keys for i18n
const (
	ErrorKeyConnectFailed = "error.connect_failed"
	ErrorKeyAuthFailed    = "error.auth_failed"
	ErrorKeyUnknown       = "error.unknown"
)

// PrintError 统一错误输出（使用 i18n）
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
