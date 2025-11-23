package protocol

import (
	"encoding/binary"
	"errors"
	"gotunnel/pkg/log"
	"io"
)

// HeartbeatPing 表示控制通道的心跳ping包，用于保持客户端-服务端连接活跃状态
// 服务端收到应立即回复 HeartbeatPong
// Type 固定值: "ping"
type HeartbeatPing struct {
	Type string `json:"type"` // "ping"
	Time int64  `json:"time"` // 可选，当前时间戳，可用于监控日志
}

// HeartbeatPong 表示心跳pong响应包
// Type 固定值: "pong"
type HeartbeatPong struct {
	Type string `json:"type"` // "pong"
	Time int64  `json:"time"` // 可选
}

// RegisterRequest 表示客户端注册端口的控制消息结构（用于注册需要被服务端代理的端口）
type RegisterRequest struct {
	Type       string `json:"type"`        // 固定为"register"
	LocalPort  int    `json:"local_port"`  // 客户端本地需要映射的端口
	RemotePort int    `json:"remote_port"` // 服务端为该映射开放的公网端口
	Protocol   string `json:"protocol"`    // 协议 "tcp"/"http" 等
	Token      string `json:"token"`       // 认证 token
	Name       string `json:"name"`        // 客户端自定义名称
}

// RegisterResponse 表示服务端响应注册的控制消息，用于确认/拒绝
type RegisterResponse struct {
	Type   string `json:"type"`             // 固定为"register_resp"
	Status string `json:"status"`           // "ok" / "fail"
	Reason string `json:"reason,omitempty"` // 失败时说明原因
}

// OfflinePortRequest 通知 server 某端口（remote_port）应下线，停止公网监听和映射
// Type: "offline_port"
type OfflinePortRequest struct {
	Type string `json:"type"` // "offline_port"
	Port int    `json:"port"` // remote_port 被摘除
}

// OnlinePortRequest 通知 server 某端口（remote_port）健康恢复，重新上线注册relay
// Type: "online_port"
type OnlinePortRequest struct {
	Type string `json:"type"` // "online_port"
	Port int    `json:"port"` // remote_port 被恢复
}

// WritePacket 将一条完整的消息写入连接，格式为：4字节包体长度（大端） + 原始消息内容（payload）。
// 参数说明：
//
//	w: 目标 io.Writer，对应网络连接等
//	payload: 需发送的消息内容（一般为json/protobuf编码结果）
//
// 返回值：
//
//	返回写入过程中的错误（如有），写入成功返回nil
func WritePacket(w io.Writer, payload []byte) error {
	// 最大消息长度限制，防止恶意大包攻击
	if len(payload) > 0x7fffffff {
		log.Errorf("protocol", "error.payload_too_large", len(payload))
		return errors.New("payload too large")
	}
	// 4字节大端存储包体长度
	var lenBuf [4]byte
	binary.BigEndian.PutUint32(lenBuf[:], uint32(len(payload)))
	// 先写包长，再写正文内容
	if _, err := w.Write(lenBuf[:]); err != nil {
		return err
	}
	_, err := w.Write(payload)
	return err
}

// ReadPacket 从连接读取一条完整消息，格式要求同上（4字节包体长度 + 实际内容）。
// 参数说明：
//
//	r: 来源 io.Reader，一般是网络连接
//
// 返回值：
//
//	[]byte: 消息内容
//	error: 读取或解包过程中遇到的错误
func ReadPacket(r io.Reader) ([]byte, error) {
	// 先读4字节包长度
	var lenBuf [4]byte
	if _, err := io.ReadFull(r, lenBuf[:]); err != nil {
		return nil, err
	}
	length := binary.BigEndian.Uint32(lenBuf[:])
	if length == 0 {
		return nil, nil // 空包体情况
	}
	// 按长度读出实际消息内容
	buf := make([]byte, length)
	if _, err := io.ReadFull(r, buf); err != nil {
		return nil, err
	}
	return buf, nil
}

// 本文件核心作用：
//   - 明确每条消息边界，彻底解决tcp粘包、分包问题
//   - 便于上层用json/protobuf自定义消息，实现安全、可扩展通信格式
//   - 此方案兼容生产环境与本地开发调试，后续协议升级亦可直接复用本传输层逻辑
