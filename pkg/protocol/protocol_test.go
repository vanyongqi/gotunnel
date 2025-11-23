package protocol

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"testing"
	"time"
)

func TestWriteAndReadPacket(t *testing.T) {
	msg := []byte("hello-gotunnel")
	buf := &bytes.Buffer{}
	// 写入/读取
	if err := WritePacket(buf, msg); err != nil {
		t.Fatalf("写入失败: %v", err)
	}
	data, err := ReadPacket(buf)
	if err != nil {
		t.Fatalf("读取失败: %v", err)
	}
	if string(data) != string(msg) {
		t.Errorf("内容不符: got %s want %s", string(data), string(msg))
	}
}

func TestMultiPacket(t *testing.T) {
	buf := &bytes.Buffer{}
	inputs := [][]byte{{'a'}, {'b'}, {'c', 'd'}}
	for _, in := range inputs {
		if err := WritePacket(buf, in); err != nil {
			t.Fatal(err)
		}
	}
	for _, expect := range inputs {
		out, err := ReadPacket(buf)
		if err != nil {
			t.Fatal(err)
		}
		if string(out) != string(expect) {
			t.Errorf("多包流不符: got %v, want %v", out, expect)
		}
	}
}

func TestEmptyPacket(t *testing.T) {
	buf := &bytes.Buffer{}
	if err := WritePacket(buf, []byte{}); err != nil {
		t.Fatal(err)
	}
	data, err := ReadPacket(buf)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) != 0 {
		t.Errorf("空包应返回空slice")
	}
}

func TestHugePacket(t *testing.T) {
	buf := &bytes.Buffer{}
	big := make([]byte, 0x80000000)
	if err := WritePacket(buf, big); err == nil {
		t.Error("超大消息应报错")
	}
}

func TestBrokenStream(t *testing.T) {
	buf := &bytes.Buffer{}
	msg := []byte("ABCDE")
	WritePacket(buf, msg)
	raw := buf.Bytes()
	partial := bytes.NewBuffer(raw[:len(raw)-2])
	_, err := ReadPacket(partial)
	if err == nil && !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Error("流截断应报io.ErrUnexpectedEOF")
	}
}

func TestJSONStructCompat(t *testing.T) {
	reg := RegisterRequest{Type: "register", LocalPort: 22, RemotePort: 10022, Protocol: "tcp", Token: "tk", Name: "cc"}
	b, err := json.Marshal(reg)
	if err != nil {
		t.Fatal(err)
	}
	var out RegisterRequest
	if err = json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	if reg.Type != out.Type || reg.LocalPort != out.LocalPort || reg.Protocol != out.Protocol {
		t.Error("序列化不对称出错")
	}
}

func TestHeartbeatPingStruct(t *testing.T) {
	now := time.Now().Unix()
	ping := HeartbeatPing{Type: "ping", Time: now}
	b, err := json.Marshal(ping)
	if err != nil {
		t.Fatal(err)
	}
	var out HeartbeatPing
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	if out.Type != "ping" || out.Time != now {
		t.Error("HeartbeatPing序列化异常")
	}
}

func TestWritePacketError(t *testing.T) {
	// Test write error
	ew := &errorWriter{}
	err := WritePacket(ew, []byte("test"))
	if err == nil {
		t.Error("WritePacket should return error on write failure")
	}
}

func TestReadPacketError(t *testing.T) {
	// Test read error
	er := &errorReader{}
	_, err := ReadPacket(er)
	if err == nil {
		t.Error("ReadPacket should return error on read failure")
	}
}

func TestReadPacketIncompleteLength(t *testing.T) {
	// Test incomplete length field
	buf := bytes.NewBuffer([]byte{0x00, 0x00}) // Only 2 bytes instead of 4
	_, err := ReadPacket(buf)
	if err == nil {
		t.Error("ReadPacket should return error on incomplete length field")
	}
}

func TestReadPacketIncompletePayload(t *testing.T) {
	// Test incomplete payload
	buf := bytes.NewBuffer([]byte{0x00, 0x00, 0x00, 0x05}) // Length = 5
	buf.Write([]byte{0x01, 0x02})                          // Only 2 bytes instead of 5
	_, err := ReadPacket(buf)
	if err == nil {
		t.Error("ReadPacket should return error on incomplete payload")
	}
}

type errorWriter struct{}

func (e *errorWriter) Write([]byte) (int, error) {
	return 0, errors.New("write error")
}

type errorReader struct{}

func (e *errorReader) Read([]byte) (int, error) {
	return 0, errors.New("read error")
}
