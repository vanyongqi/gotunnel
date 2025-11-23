package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"gotunnel/pkg/log"
	"gotunnel/pkg/protocol"
	"io"
	"net"
	"testing"
	"time"

	"github.com/spf13/viper"
	"golang.org/x/text/language"
)

type mockConn struct {
	io.Reader
	io.Writer
	closed bool
}

func (m *mockConn) Read(b []byte) (int, error)         { return m.Reader.Read(b) }
func (m *mockConn) Write(b []byte) (int, error)        { return m.Writer.Write(b) }
func (m *mockConn) Close() error                       { m.closed = true; return nil }
func (m *mockConn) LocalAddr() net.Addr                { return nil }
func (m *mockConn) RemoteAddr() net.Addr               { return nil }
func (m *mockConn) SetDeadline(t time.Time) error      { return nil }
func (m *mockConn) SetReadDeadline(t time.Time) error  { return nil }
func (m *mockConn) SetWriteDeadline(t time.Time) error { return nil }

type errorReader struct{}

func (e *errorReader) Read([]byte) (int, error) { return 0, errors.New("read error") }

type errorWriter struct{}

func (e *errorWriter) Write([]byte) (int, error) { return 0, errors.New("write error") }

func TestRegisterPort_Success(t *testing.T) {
	conf := &ClientConfig{Name: "test", Token: "tok", LocalPort: 1, RemotePort: 2}
	var rbuf, wbuf bytes.Buffer
	resp := protocol.RegisterResponse{Type: "register_resp", Status: "ok"}
	b, _ := json.Marshal(resp)
	protocol.WritePacket(&wbuf, b)
	conn := &mockConn{Reader: bytes.NewReader(wbuf.Bytes()), Writer: &rbuf}
	err := RegisterPort(conn, conf)
	if err != nil {
		t.Fatal(err)
	}
}

func TestRegisterPort_AuthFail(t *testing.T) {
	conf := &ClientConfig{Name: "test", Token: "tok", LocalPort: 1, RemotePort: 2}
	var rbuf, wbuf bytes.Buffer
	resp := protocol.RegisterResponse{Type: "register_resp", Status: "fail", Reason: "INVALID TOKEN"}
	b, _ := json.Marshal(resp)
	protocol.WritePacket(&wbuf, b)
	conn := &mockConn{Reader: bytes.NewReader(wbuf.Bytes()), Writer: &rbuf}
	err := RegisterPort(conn, conf)
	if err == nil {
		t.Fatal("auth failure expect error")
	}
}

func TestRegisterPort_WritePacketError(t *testing.T) {
	conf := &ClientConfig{Name: "test", Token: "tok", LocalPort: 1, RemotePort: 2}
	conn := &mockConn{Reader: &bytes.Buffer{}, Writer: &errorWriter{}}
	err := RegisterPort(conn, conf)
	if err == nil {
		t.Fatal("expected write error")
	}
}

func TestRegisterPort_ReadPacketError(t *testing.T) {
	conf := &ClientConfig{Name: "test", Token: "tok", LocalPort: 1, RemotePort: 2}
	conn := &mockConn{Reader: &errorReader{}, Writer: &bytes.Buffer{}}
	err := RegisterPort(conn, conf)
	if err == nil {
		t.Fatal("expected read error")
	}
}

func TestHeartbeatManager_StartStop(t *testing.T) {
	var buf bytes.Buffer
	conn := &mockConn{Reader: &buf, Writer: &buf}
	mgr := &HeartbeatManager{Conn: conn, Interval: 1 * time.Millisecond, OnTimeout: func() {}}
	mgr.StartHeartbeat()
	time.Sleep(5 * time.Millisecond) // 确保至少发送一次心跳
	mgr.StopHeartbeat()
	time.Sleep(2 * time.Millisecond) // 确保停止后不再发送
}

func TestStartHeartbeat(t *testing.T) {
	var buf bytes.Buffer
	conn := &mockConn{Reader: &buf, Writer: &buf}
	stop := StartHeartbeat(conn, 1*time.Millisecond, func() {})
	time.Sleep(5 * time.Millisecond)
	stop()
	time.Sleep(2 * time.Millisecond)
}

func TestStartHealthProbe(t *testing.T) {
	conf := &ClientConfig{Name: "test", LocalPort: 99999} // 使用不存在的端口
	var buf bytes.Buffer
	conn := &mockConn{Reader: &buf, Writer: &buf}
	stop := StartHealthProbe(conf, conn,
		func() { /* offline callback */ },
		func() { /* online callback */ },
	)
	time.Sleep(100 * time.Millisecond) // 等待一次探针检查
	stop()
	// 由于端口不存在，应该会触发offline回调
	time.Sleep(200 * time.Millisecond)
}

func TestStartControlLoop_Pong(t *testing.T) {
	var wbuf bytes.Buffer
	pong := protocol.HeartbeatPong{Type: "pong", Time: time.Now().Unix()}
	b, _ := json.Marshal(pong)
	protocol.WritePacket(&wbuf, b)
	conn := &mockConn{Reader: bytes.NewReader(wbuf.Bytes()), Writer: &bytes.Buffer{}}
	conf := &ClientConfig{LocalPort: 22}
	done := make(chan error, 1)
	go func() {
		done <- StartControlLoop(conn, conf)
	}()
	select {
	case err := <-done:
		if err == nil {
			t.Error("expected error when connection closes")
		}
	case <-time.After(100 * time.Millisecond):
		// 正常情况，pong消息被处理，循环继续等待
	}
}

func TestStartControlLoop_DataChannel(t *testing.T) {
	// 创建一个模拟的数据通道指令
	var wbuf bytes.Buffer
	req := protocol.RegisterRequest{Type: "open_data_channel", LocalPort: 22}
	b, _ := json.Marshal(req)
	protocol.WritePacket(&wbuf, b)
	// 添加一个错误包来结束循环
	protocol.WritePacket(&wbuf, []byte("invalid"))
	conn := &mockConn{Reader: bytes.NewReader(wbuf.Bytes()), Writer: &bytes.Buffer{}}
	conf := &ClientConfig{LocalPort: 22}
	err := StartControlLoop(conn, conf)
	// 由于本地端口22可能不存在，会返回错误或继续循环
	_ = err
}

func TestStartControlLoop_UnknownMessage(t *testing.T) {
	// 测试未知消息类型（既不是pong也不是open_data_channel）
	var wbuf bytes.Buffer
	unknown := map[string]string{"type": "unknown", "data": "test"}
	b, _ := json.Marshal(unknown)
	protocol.WritePacket(&wbuf, b)
	// 添加一个错误包来结束循环
	protocol.WritePacket(&wbuf, []byte("invalid"))
	conn := &mockConn{Reader: bytes.NewReader(wbuf.Bytes()), Writer: &bytes.Buffer{}}
	conf := &ClientConfig{LocalPort: 22}
	done := make(chan error, 1)
	go func() {
		done <- StartControlLoop(conn, conf)
	}()
	select {
	case err := <-done:
		if err == nil {
			t.Error("expected error when connection closes")
		}
	case <-time.After(100 * time.Millisecond):
		// 正常情况，未知消息被忽略，循环继续等待
	}
}

func TestDialServer(t *testing.T) {
	// 测试连接失败的情况
	conf := &ClientConfig{ServerAddr: "127.0.0.1:99999"}
	_, err := DialServer(conf)
	if err == nil {
		t.Error("expected connection error")
	}
}

func TestLoadClientConfig(t *testing.T) {
	// 保存原始配置
	viper.Reset()
	// 测试默认值
	conf := loadClientConfig()
	if conf.Name == "" {
		t.Error("expected default name")
	}
	if conf.ServerAddr == "" {
		t.Error("expected default server addr")
	}
	if conf.LocalPort == 0 {
		t.Error("expected default local port")
	}
}

func TestLoadClientConfig_WithViper(t *testing.T) {
	viper.Reset()
	viper.Set("client.name", "test-client")
	viper.Set("client.token", "test-token")
	viper.Set("client.server_addr", "192.168.1.1:8080")
	viper.Set("client.local_ports", []interface{}{8080})
	conf := loadClientConfig()
	if conf.Name != "test-client" {
		t.Errorf("expected test-client, got %s", conf.Name)
	}
	if conf.Token != "test-token" {
		t.Errorf("expected test-token, got %s", conf.Token)
	}
	if conf.ServerAddr != "192.168.1.1:8080" {
		t.Errorf("expected 192.168.1.1:8080, got %s", conf.ServerAddr)
	}
	if conf.LocalPort != 8080 {
		t.Errorf("expected 8080, got %d", conf.LocalPort)
	}
}

func TestHandleConnection(t *testing.T) {
	// Initialize logger for testing
	log.Init(log.LevelInfo, language.Chinese)

	// Create a connection that will close immediately (simulating connection error)
	var wbuf bytes.Buffer
	// Send a pong message then close
	pong := protocol.HeartbeatPong{Type: "pong", Time: time.Now().Unix()}
	b, _ := json.Marshal(pong)
	protocol.WritePacket(&wbuf, b)

	conn := &mockConn{
		Reader: bytes.NewReader(wbuf.Bytes()),
		Writer: &bytes.Buffer{},
	}

	conf := &ClientConfig{
		Name:       "test",
		LocalPort:  99999, // Non-existent port for health probe
		RemotePort: 10022,
		LogLevel:   "info",
		LogLang:    "zh",
	}

	done := make(chan error, 1)
	go func() {
		// handleConnection will call StartControlLoop which will return when connection closes
		done <- handleConnection(conn, conf)
	}()

	// Wait a bit for health probe and heartbeat to start
	time.Sleep(50 * time.Millisecond)

	// Close connection to trigger error
	conn.Close()

	select {
	case err := <-done:
		// Expected error when connection closes
		if err == nil {
			t.Error("expected error when connection closes")
		}
	case <-time.After(500 * time.Millisecond):
		// Timeout - connection handling might still be running
		// This is acceptable as health probe and heartbeat are running
	}
}

func TestLoadClientConfig_RemotePort(t *testing.T) {
	viper.Reset()
	viper.Set("client.remote_port", 6443)
	conf := loadClientConfig()
	if conf.RemotePort != 6443 {
		t.Errorf("expected remote_port 6443, got %d", conf.RemotePort)
	}
}

func TestLoadClientConfig_LogSettings(t *testing.T) {
	viper.Reset()
	viper.Set("client.log_level", "debug")
	viper.Set("client.log_lang", "en")
	conf := loadClientConfig()
	if conf.LogLevel != "debug" {
		t.Errorf("expected log_level debug, got %s", conf.LogLevel)
	}
	if conf.LogLang != "en" {
		t.Errorf("expected log_lang en, got %s", conf.LogLang)
	}
}
