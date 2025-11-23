package errors

// Error codes for gotunnel.
const (
	// ErrConnectFailed indicates a connection failure to the server.
	ErrConnectFailed = 1001
	// ErrAuthFailed indicates an authentication failure.
	ErrAuthFailed = 1002
	// ...后续可扩展更多错误码
)

var errorMessages = map[int]string{
	ErrConnectFailed: "无法连接服务端",
	ErrAuthFailed:    "认证失败，请检查token",
}

// PrintError 统一错误输出
func PrintError(code int, detail error) {
	msg, ok := errorMessages[code]
	if !ok {
		msg = "未知错误"
	}
	if detail != nil {
		println("[ERROR][", code, "]", msg, ":", detail.Error())
	} else {
		println("[ERROR][", code, "]", msg)
	}
}
