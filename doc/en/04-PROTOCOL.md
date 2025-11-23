# gotunnel Protocol Documentation

**Language:** [English](./04-PROTOCOL.md) | [中文](../zh/04-PROTOCOL.md)

## Protocol Overview

gotunnel adopts a **control channel + data channel** dual-channel architecture:

- **Control Channel**: For client registration, heartbeat, state synchronization, and other control messages
- **Data Channel**: For transparent forwarding of actual business traffic

## Message Format

### 1. Transport Layer Protocol

All control messages use **"header + length + message body"** format:

```
[4-byte length (big-endian)] + [message body (JSON/binary)]
```

- **Length Field**: 4 bytes, big-endian (BigEndian), represents the byte count of the message body
- **Message Body**: JSON format control messages or binary data

### 2. Message Boundary Handling

Use `WritePacket` and `ReadPacket` functions from `pkg/protocol` package to ensure message boundaries:

```go
// Send message
protocol.WritePacket(conn, jsonBytes)

// Receive message
packet, err := protocol.ReadPacket(conn)
```

This completely avoids TCP packet sticking/fragmentation issues.

## Control Message Types

### 1. Port Registration Request (RegisterRequest)

Client registers ports that need to be mapped with the server.

**Message Format:**
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

**Field Description:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `type` | string | Yes | Fixed value `"register"` |
| `local_port` | int | Yes | Local port on client to map |
| `remote_port` | int | Yes | Public port exposed by server |
| `protocol` | string | Yes | Protocol type, currently supports `"tcp"` |
| `token` | string | Yes | Authentication token |
| `name` | string | Yes | Client name |

### 2. Port Registration Response (RegisterResponse)

Server responds to registration request.

**Message Format:**
```json
{
  "type": "register_resp",
  "status": "ok",
  "reason": ""
}
```

**Field Description:**

| Field | Type | Description |
|-------|------|-------------|
| `type` | string | Fixed value `"register_resp"` |
| `status` | string | `"ok"` for success, `"fail"` for failure |
| `reason` | string | Reason description on failure (optional) |

### 3. Heartbeat Packet (HeartbeatPing/HeartbeatPong)

Used to keep control channel connection alive.

**Ping Message:**
```json
{
  "type": "ping",
  "time": 1703123456
}
```

**Pong Response:**
```json
{
  "type": "pong",
  "time": 1703123456
}
```

### 4. Data Channel Establishment Request (OpenDataChannel)

Server notifies client to establish data channel.

**Message Format:**
```json
{
  "type": "open_data_channel",
  "local_port": 22
}
```

### 5. Port Offline Request (OfflinePortRequest)

Client notifies server that port is offline.

**Message Format:**
```json
{
  "type": "offline_port",
  "port": 10022
}
```

### 6. Port Online Request (OnlinePortRequest)

Client notifies server that port is back online.

**Message Format:**
```json
{
  "type": "online_port",
  "port": 10022
}
```

## Data Channel Protocol

Data channel uses **fully transparent TCP forwarding**, no protocol parsing:

- All byte streams are directly forwarded bidirectionally
- Supports any TCP protocol (SSH, HTTP, MySQL, Redis, etc.)
- Maintains long connection characteristics

## Communication Flow

### 1. Client Registration Flow

```
Client                    Server
  |                         |
  |--- RegisterRequest ---->|
  |                         | (verify token)
  |<-- RegisterResponse ----|
  |                         | (listen on remote_port)
```

### 2. Data Forwarding Flow

```
User                    Server                    Client                    LocalService
  |                        |                         |                          |
  |--- TCP Connect ------->|                         |                          |
  |                        |--- OpenDataChannel ---->|                          |
  |                        |                         |--- TCP Connect --------->|
  |                        |<-- Data Channel -------|                          |
  |<-- Data Channel -------|                         |                          |
  |                        |                         |                          |
  |<========== Bidirectional Data Forwarding ===========>|                          |
```

### 3. Heartbeat Keepalive Flow

```
Client                    Server
  |                         |
  |--- Ping (every 10s) ---->|
  |<-- Pong ----------------|
  |                         | (update LastHeartbeat)
```

## Error Handling

### Error Code Definition

Currently defined error codes (`pkg/errors/errors.go`):

- `1001`: Failed to connect to server
- `1002`: Authentication failed

### Error Message Format

Errors are output through standard error output, format:

```
[ERROR][error_code] Error description: detailed error message
```

## Protocol Extensions

### Future Plans

- Support Protobuf serialization (performance optimization)
- Support TLS encryption for control channel
- Support multiplexing (one control channel manages multiple data channels)
- Support UDP protocol

## Reference Implementation

Detailed implementation code reference:

- `pkg/protocol/protocol.go`: Protocol definition and encoding/decoding
- `cmd/client/main.go`: Client protocol usage example
- `cmd/server/main.go`: Server protocol handling example

