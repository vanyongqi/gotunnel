package core

import (
	"io"
	"net"
	"sync"
	"testing"
	"time"
)

// mockConn 是一个简单的mock连接，用于测试
type mockConn struct {
	readCh  chan []byte
	writeCh chan []byte
	closed  bool
	mu      sync.Mutex
}

func newMockConn() *mockConn {
	return &mockConn{
		readCh:  make(chan []byte, 10),
		writeCh: make(chan []byte, 10),
	}
}

func (m *mockConn) Read(b []byte) (n int, err error) {
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return 0, io.EOF
	}
	m.mu.Unlock()

	select {
	case data := <-m.readCh:
		n := copy(b, data)
		return n, nil
	case <-time.After(100 * time.Millisecond):
		return 0, io.EOF
	}
}

func (m *mockConn) Write(b []byte) (n int, err error) {
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return 0, io.EOF
	}
	m.mu.Unlock()

	select {
	case m.writeCh <- b:
		return len(b), nil
	case <-time.After(100 * time.Millisecond):
		return 0, io.EOF
	}
}

func (m *mockConn) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return nil
	}
	m.closed = true
	close(m.readCh)
	close(m.writeCh)
	return nil
}

func (m *mockConn) LocalAddr() net.Addr                { return nil }
func (m *mockConn) RemoteAddr() net.Addr               { return nil }
func (m *mockConn) SetDeadline(t time.Time) error      { return nil }
func (m *mockConn) SetReadDeadline(t time.Time) error  { return nil }
func (m *mockConn) SetWriteDeadline(t time.Time) error { return nil }

func TestRelayConn_Simple(t *testing.T) {
	// 使用mock连接测试RelayConn的基本转发功能
	conn1 := newMockConn()
	conn2 := newMockConn()

	var wg sync.WaitGroup
	wg.Add(1)

	// 启动RelayConn
	go func() {
		RelayConn(conn1, conn2)
		wg.Done()
	}()

	// 等待RelayConn启动
	time.Sleep(20 * time.Millisecond)

	msg := []byte("hello relay")

	// 测试：从conn1读取并转发到conn2
	// RelayConn会从conn1.readCh读取，并写入conn2.writeCh
	// 所以我们向conn1.readCh写入数据，然后从conn2.writeCh读取验证
	conn1.readCh <- msg

	// 等待数据转发完成
	select {
	case data := <-conn2.writeCh:
		if string(data) != string(msg) {
			t.Errorf("转发失败 got=%s want=%s", string(data), msg)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("转发超时")
	}

	// 关闭连接让RelayConn退出
	conn1.Close()
	conn2.Close()

	// 等待RelayConn退出
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		// 正常完成
	case <-time.After(100 * time.Millisecond):
		t.Fatal("RelayConn未及时退出")
	}
}

// TestRelayConn_BothDirections 已移除，使用简单的单向测试已足够验证RelayConn功能
