package errors

import (
	"errors"
	"testing"
)

func TestPrintError(t *testing.T) {
	// 测试正常错误码
	err := errors.New("test error")
	PrintError(ErrConnectFailed, err)
	PrintError(ErrAuthFailed, err)

	// 测试未知错误码
	PrintError(9999, err)

	// 测试nil错误
	PrintError(ErrConnectFailed, nil)
}

func TestErrorMessages(t *testing.T) {
	// 验证错误码映射存在
	if errorMessages[ErrConnectFailed] == "" {
		t.Error("ErrConnectFailed 消息未定义")
	}
	if errorMessages[ErrAuthFailed] == "" {
		t.Error("ErrAuthFailed 消息未定义")
	}
}
