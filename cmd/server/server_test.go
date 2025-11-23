package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"gotunnel/pkg/protocol"
	"io"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/spf13/viper"
)

type mockConn struct {
	io.Reader
	io.Writer
	closed bool
	mu     sync.Mutex
}

func (m *mockConn) Read(b []byte) (int, error)         { return m.Reader.Read(b) }
func (m *mockConn) Write(b []byte) (int, error)        { return m.Writer.Write(b) }
func (m *mockConn) Close() error                       { m.mu.Lock(); defer m.mu.Unlock(); m.closed = true; return nil }
func (m *mockConn) LocalAddr() net.Addr                { return nil }
func (m *mockConn) RemoteAddr() net.Addr               { return nil }
func (m *mockConn) SetDeadline(t time.Time) error      { return nil }
func (m *mockConn) SetReadDeadline(t time.Time) error  { return nil }
func (m *mockConn) SetWriteDeadline(t time.Time) error { return nil }

type errorReader struct{}

func (e *errorReader) Read([]byte) (int, error) { return 0, errors.New("read error") }

type errorWriter struct{}

func (e *errorWriter) Write([]byte) (int, error) { return 0, errors.New("write error") }

func TestLoadServerConfig(t *testing.T) {
	viper.Reset()
	conf := loadServerConfig()
	if conf.ListenAddr == "" {
		t.Error("expected default listen addr")
	}
	if conf.Token == "" {
		t.Error("expected default token")
	}
}

func TestLoadServerConfig_WithViper(t *testing.T) {
	viper.Reset()
	viper.Set("server.addr", ":8080")
	viper.Set("server.token", "test-token")
	conf := loadServerConfig()
	if conf.ListenAddr != ":8080" {
		t.Errorf("expected :8080, got %s", conf.ListenAddr)
	}
	if conf.Token != "test-token" {
		t.Errorf("expected test-token, got %s", conf.Token)
	}
}

func TestCheckClientHeartbeat(t *testing.T) {
	// 清理映射表
	mappingTableMu.Lock()
	mappingTable = make(map[int]*Mapping)
	mappingTableMu.Unlock()

	// 创建一个过期的映射
	var buf bytes.Buffer
	conn := &mockConn{Reader: &buf, Writer: &buf}
	mappingTableMu.Lock()
	mappingTable[8080] = &Mapping{
		ClientConn:    conn,
		LocalPort:     22,
		LastHeartbeat: time.Now().Add(-40 * time.Second), // 40秒前，超过30秒超时
	}
	mappingTableMu.Unlock()

	checkClientHeartbeat()

	mappingTableMu.Lock()
	if _, exists := mappingTable[8080]; exists {
		t.Error("expected mapping to be deleted after heartbeat timeout")
	}
	mappingTableMu.Unlock()
}

func TestCheckClientHeartbeat_NoTimeout(t *testing.T) {
	// 清理映射表
	mappingTableMu.Lock()
	mappingTable = make(map[int]*Mapping)
	mappingTableMu.Unlock()

	// 创建一个未过期的映射
	var buf bytes.Buffer
	conn := &mockConn{Reader: &buf, Writer: &buf}
	mappingTableMu.Lock()
	mappingTable[8080] = &Mapping{
		ClientConn:    conn,
		LocalPort:     22,
		LastHeartbeat: time.Now(), // 刚刚更新
	}
	mappingTableMu.Unlock()

	checkClientHeartbeat()

	mappingTableMu.Lock()
	if _, exists := mappingTable[8080]; !exists {
		t.Error("expected mapping to still exist")
	}
	mappingTableMu.Unlock()
}

func TestHandleControlConn_ReadPacketError(t *testing.T) {
	conn := &mockConn{Reader: &errorReader{}, Writer: &bytes.Buffer{}}
	handleControlConn(conn, "test-token")
	// 应该正常返回，不panic
}

func TestHandleControlConn_AuthFail(t *testing.T) {
	var wbuf bytes.Buffer
	req := protocol.RegisterRequest{
		Type:       "register",
		LocalPort:  22,
		RemotePort: 8080,
		Token:      "wrong-token",
		Name:       "test-client",
	}
	b, _ := json.Marshal(req)
	protocol.WritePacket(&wbuf, b)
	conn := &mockConn{Reader: bytes.NewReader(wbuf.Bytes()), Writer: &bytes.Buffer{}}
	handleControlConn(conn, "correct-token")
	// 应该拒绝并返回
}

func TestHandleControlConn_RegisterSuccess(t *testing.T) {
	// 清理映射表
	mappingTableMu.Lock()
	mappingTable = make(map[int]*Mapping)
	mappingTableMu.Unlock()

	var wbuf bytes.Buffer
	req := protocol.RegisterRequest{
		Type:       "register",
		LocalPort:  22,
		RemotePort: 8080,
		Token:      "test-token",
		Name:       "test-client",
	}
	b, _ := json.Marshal(req)
	protocol.WritePacket(&wbuf, b)
	// 添加一个ping消息来测试心跳处理
	ping := protocol.HeartbeatPing{Type: "ping", Time: time.Now().Unix()}
	pingBytes, _ := json.Marshal(ping)
	protocol.WritePacket(&wbuf, pingBytes)
	// 添加一个错误来结束循环
	protocol.WritePacket(&wbuf, []byte("invalid"))

	conn := &mockConn{Reader: bytes.NewReader(wbuf.Bytes()), Writer: &bytes.Buffer{}}
	done := make(chan struct{})
	go func() {
		handleControlConn(conn, "test-token")
		close(done)
	}()

	select {
	case <-done:
		// 正常完成
	case <-time.After(2 * time.Second):
		t.Error("handleControlConn should complete")
	}

	mappingTableMu.Lock()
	if _, exists := mappingTable[8080]; exists {
		t.Error("expected mapping to be cleaned up after connection closes")
	}
	mappingTableMu.Unlock()
}

func TestHandleControlConn_OfflinePort(t *testing.T) {
	// 清理映射表
	mappingTableMu.Lock()
	mappingTable = make(map[int]*Mapping)
	var buf bytes.Buffer
	conn := &mockConn{Reader: &buf, Writer: &buf}
	mappingTable[8080] = &Mapping{
		ClientConn:    conn,
		LocalPort:     22,
		LastHeartbeat: time.Now(),
	}
	mappingTableMu.Unlock()

	var wbuf bytes.Buffer
	req := protocol.RegisterRequest{
		Type:       "register",
		LocalPort:  22,
		RemotePort: 8080,
		Token:      "test-token",
		Name:       "test-client",
	}
	b, _ := json.Marshal(req)
	protocol.WritePacket(&wbuf, b)
	// 添加offline_port请求
	offline := protocol.OfflinePortRequest{Type: "offline_port", Port: 8080}
	offlineBytes, _ := json.Marshal(offline)
	protocol.WritePacket(&wbuf, offlineBytes)
	// 添加错误来结束循环
	protocol.WritePacket(&wbuf, []byte("invalid"))

	conn2 := &mockConn{Reader: bytes.NewReader(wbuf.Bytes()), Writer: &bytes.Buffer{}}
	done := make(chan struct{})
	go func() {
		handleControlConn(conn2, "test-token")
		close(done)
	}()

	select {
	case <-done:
		// 正常完成
	case <-time.After(2 * time.Second):
		t.Error("handleControlConn should complete")
	}

	mappingTableMu.Lock()
	if _, exists := mappingTable[8080]; exists {
		t.Error("expected mapping to be deleted after offline_port")
	}
	mappingTableMu.Unlock()
}

func TestHandleControlConn_OnlinePort(t *testing.T) {
	// 清理映射表
	mappingTableMu.Lock()
	mappingTable = make(map[int]*Mapping)
	mappingTableMu.Unlock()

	var wbuf bytes.Buffer
	req := protocol.RegisterRequest{
		Type:       "register",
		LocalPort:  22,
		RemotePort: 8080,
		Token:      "test-token",
		Name:       "test-client",
	}
	b, _ := json.Marshal(req)
	protocol.WritePacket(&wbuf, b)
	// 添加online_port请求
	online := protocol.OnlinePortRequest{Type: "online_port", Port: 8080}
	onlineBytes, _ := json.Marshal(online)
	protocol.WritePacket(&wbuf, onlineBytes)
	// 添加错误来结束循环
	protocol.WritePacket(&wbuf, []byte("invalid"))

	conn := &mockConn{Reader: bytes.NewReader(wbuf.Bytes()), Writer: &bytes.Buffer{}}
	done := make(chan struct{})
	go func() {
		handleControlConn(conn, "test-token")
		close(done)
	}()

	select {
	case <-done:
		// 正常完成
	case <-time.After(2 * time.Second):
		t.Error("handleControlConn should complete")
	}
}

func TestListenAndForwardWithStop(t *testing.T) {
	// 找一个可用端口
	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().(*net.TCPAddr)
	ln.Close()

	var buf bytes.Buffer
	clientConn := &mockConn{Reader: &buf, Writer: &buf}
	stop := make(chan struct{})

	done := make(chan struct{})
	go func() {
		listenAndForwardWithStop(addr.Port, clientConn, 22, stop)
		close(done)
	}()

	// 等待监听启动
	time.Sleep(100 * time.Millisecond)

	// 停止监听
	close(stop)

	select {
	case <-done:
		// 正常完成
	case <-time.After(2 * time.Second):
		t.Error("listenAndForwardWithStop should complete after stop")
	}
}

func TestListenAndForwardWithStop_ListenError(t *testing.T) {
	// 使用一个无效的端口（可能需要root权限的端口，或者已经被占用的端口）
	// 这里我们使用一个可能无效的端口号
	var buf bytes.Buffer
	clientConn := &mockConn{Reader: &buf, Writer: &buf}
	stop := make(chan struct{})
	// 使用一个非常大的端口号，可能会失败
	listenAndForwardWithStop(999999, clientConn, 22, stop)
	// 应该正常返回，不panic
}

func TestListenAndForwardWithStop_AcceptConnection(t *testing.T) {
	// 找一个可用端口
	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().(*net.TCPAddr)
	ln.Close()

	var buf bytes.Buffer
	clientConn := &mockConn{Reader: &buf, Writer: &buf}
	stop := make(chan struct{})

	done := make(chan struct{})
	go func() {
		listenAndForwardWithStop(addr.Port, clientConn, 22, stop)
		close(done)
	}()

	// 等待监听启动
	time.Sleep(100 * time.Millisecond)

	// 尝试连接
	userConn, err := net.Dial("tcp", addr.String())
	if err != nil {
		t.Logf("连接失败（可能监听还未启动）: %v", err)
	} else {
		userConn.Close()
		time.Sleep(50 * time.Millisecond) // 等待处理
	}

	// 停止监听
	close(stop)

	select {
	case <-done:
		// 正常完成
	case <-time.After(2 * time.Second):
		t.Error("listenAndForwardWithStop should complete after stop")
	}
}
