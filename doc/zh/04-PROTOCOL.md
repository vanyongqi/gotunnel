# gotunnel 协议文档

**Language:** [English](../en/04-PROTOCOL.md) | [中文](./04-PROTOCOL.md)

## 一、协议概述

gotunnel 采用**控制通道 + 数据通道**的双通道架构：

- **控制通道**：用于客户端注册、心跳、状态同步等控制消息
- **数据通道**：用于实际业务流量的透明转发

## 二、消息格式

### 1. 传输层协议

所有控制消息采用 **"包头 + 长度 + 消息体"** 格式：

```
[4字节长度(大端)] + [消息体(JSON/二进制)]
```

- **长度字段**：4字节，大端序（BigEndian），表示消息体的字节数
- **消息体**：JSON 格式的控制消息或二进制数据

### 2. 消息边界处理

使用 `pkg/protocol` 包的 `WritePacket` 和 `ReadPacket` 函数确保消息边界：

```go
// 发送消息
protocol.WritePacket(conn, jsonBytes)

// 接收消息
packet, err := protocol.ReadPacket(conn)
```

这样可以完全避免 TCP 粘包/分包问题。

## 三、控制消息类型

### 1. 端口注册请求（RegisterRequest）

客户端向服务端注册需要映射的端口。

**消息格式：**
```json
{
  "type": "register",
  "local_port": 22,
  "remote_port": 10022,
  "protocol": "tcp",
  "token": "your-token",
  "name": "client-name"
}
```

**字段说明：**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `type` | string | 是 | 固定值 `"register"` |
| `local_port` | int | 是 | 客户端本地要映射的端口 |
| `remote_port` | int | 是 | 服务端对外暴露的公网端口 |
| `protocol` | string | 是 | 协议类型，当前支持 `"tcp"` |
| `token` | string | 是 | 认证token |
| `name` | string | 是 | 客户端名称 |

### 2. 端口注册响应（RegisterResponse）

服务端响应注册请求。

**消息格式：**
```json
{
  "type": "register_resp",
  "status": "ok",
  "reason": ""
}
```

**字段说明：**

| 字段 | 类型 | 说明 |
|------|------|------|
| `type` | string | 固定值 `"register_resp"` |
| `status` | string | `"ok"` 表示成功，`"fail"` 表示失败 |
| `reason` | string | 失败时的原因说明（可选） |

### 3. 心跳包（HeartbeatPing/HeartbeatPong）

用于保持控制通道连接活跃。

**Ping 消息：**
```json
{
  "type": "ping",
  "time": 1703123456
}
```

**Pong 响应：**
```json
{
  "type": "pong",
  "time": 1703123456
}
```

### 4. 数据通道建立请求（OpenDataChannel）

服务端通知客户端建立数据通道。

**消息格式：**
```json
{
  "type": "open_data_channel",
  "local_port": 22
}
```

### 5. 端口下线请求（OfflinePortRequest）

客户端通知服务端端口下线。

**消息格式：**
```json
{
  "type": "offline_port",
  "port": 10022
}
```

### 6. 端口上线请求（OnlinePortRequest）

客户端通知服务端端口恢复上线。

**消息格式：**
```json
{
  "type": "online_port",
  "port": 10022
}
```

## 四、数据通道协议

数据通道采用**全透明 TCP 转发**，不进行任何协议解析：

- 所有字节流直接双向转发
- 支持任意 TCP 协议（SSH、HTTP、MySQL、Redis 等）
- 保持长连接特性

## 五、通信流程

### 1. 客户端注册流程

```
Client                    Server
  |                         |
  |--- RegisterRequest ---->|
  |                         | (验证token)
  |<-- RegisterResponse ----|
  |                         | (监听remote_port)
```

### 2. 数据转发流程

```
User                    Server                    Client                    LocalService
  |                        |                         |                          |
  |--- TCP Connect ------->|                         |                          |
  |                        |--- OpenDataChannel ---->|                          |
  |                        |                         |--- TCP Connect --------->|
  |                        |<-- Data Channel -------|                          |
  |<-- Data Channel -------|                         |                          |
  |                        |                         |                          |
  |<========== 双向数据转发 ===========>|                          |
```

### 3. 心跳保活流程

```
Client                    Server
  |                         |
  |--- Ping (每10秒) ------->|
  |<-- Pong ----------------|
  |                         | (更新LastHeartbeat)
```

## 六、错误处理

### 错误码定义

当前定义的错误码（`pkg/errors/errors.go`）：

- `1001`: 连接服务端失败
- `1002`: 认证失败

### 错误消息格式

错误通过标准错误输出，格式：

```
[ERROR][错误码] 错误描述: 详细错误信息
```

## 七、协议扩展

### 未来计划

- 支持 Protobuf 序列化（性能优化）
- 支持 TLS 加密控制通道
- 支持多路复用（一个控制通道管理多个数据通道）
- 支持 UDP 协议

## 八、参考实现

详细实现代码参考：

- `pkg/protocol/protocol.go`: 协议定义和编解码
- `cmd/client/main.go`: 客户端协议使用示例
- `cmd/server/main.go`: 服务端协议处理示例

