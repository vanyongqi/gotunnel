package errors

import (
	"errors"
	"gotunnel/pkg/log"
	"testing"

	"golang.org/x/text/language"
)

func TestPrintError(t *testing.T) {
	// Initialize logger for testing
	log.Init(log.LevelInfo, language.Chinese)

	// 测试正常错误码
	err := errors.New("test error")
	PrintError(ErrConnectFailed, err)
	PrintError(ErrAuthFailed, err)

	// 测试未知错误码
	PrintError(9999, err)

	// 测试nil错误
	PrintError(ErrConnectFailed, nil)
}
