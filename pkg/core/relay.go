package core

import (
	"io"
	"net"
)

// RelayConn 实现 A、B 两端TCP连接的全双工（双向）字节流转发，直到任意一方关闭。
// 调用后本函数会阻塞，直到任意一端关闭连接。适用于ssh/http等所有全透传业务。
func RelayConn(a, b net.Conn) {
	// a->b
	go func() {
		_, _ = io.Copy(b, a)
		_ = b.Close()
	}()
	// b->a
	_, _ = io.Copy(a, b)
	_ = a.Close()
}
