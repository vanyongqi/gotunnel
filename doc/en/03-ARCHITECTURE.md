# gotunnel Architecture Design and Key Considerations

**Language:** [English](./03-ARCHITECTURE.md) | [中文](../zh/03-ARCHITECTURE.md)

## Overall Architecture

- Server maintains long connections with multiple clients for unified control and data forwarding.
- Supports massive concurrent client registration, suitable for cloud-native/K8s scenarios.
- Control channel handles command management, heartbeat, and state synchronization; data channel handles large data traffic forwarding.
- Clear system layering for easy future feature expansion and protocol switching.

## Key Considerations

### 1. Control Channel Packet Sticking/Fragmentation and Message Boundary Solution (Solution B)

- TCP is a byte stream protocol. Direct transmission of JSON or binary messages will encounter packet sticking/fragmentation issues. Message boundaries must be resolved to correctly transmit and parse multiple messages.
- Adopt "header + length" protocol. Each control message is prefixed with a fixed-length length field (e.g., int32, 4 bytes), followed by the actual message data (e.g., JSON byte stream). The receiver reads the length first, then reads the complete message, ensuring accurate packet parsing.
- Strong compatibility, easy to upgrade to Protobuf or custom binary protocol later.

---

## Development Phases and Task Breakdown

### Phase 1: Core Tunnel Implementation (MVP Phase)

- Focus only on gotunnel backend core: Server/Client end-to-end communication, control protocol (JSON+header+length), data channel forwarding, heartbeat/disconnect/reconnect management
- Logging, basic parameter configuration, unit tests
- All daily operations/node status verified only through command line, logs, or simple scripts
- Web management interface, API, visualization not developed in this phase, only reserved as future feature interfaces

### Phase 2: Web Management UI and Advanced History Audit

- After completing and stabilizing the main tunnel pipeline, enter Web management backend development
- Enable RESTful API, frontend React management console, implement node online/historical behavior, status statistics, etc.
- Frontend management pages must have login authentication to prevent unauthorized access
- Consider session history records, data visualization, quick operations, and push support

### Phase 3: Cloud-Native Friendly and Ecosystem Extension

- Batch deployment and auto-discovery in cloud-native environments
- Multi-instance high availability, monitoring/traffic alerts, etc.
- Plugins, ACL, protocol extensions, etc.

---

## Web Management UI Design (React Version)

### 1. Login Authentication

- **Must implement frontend Web management console login** to prevent unauthorized access to backend pages.
- Recommended account/password authentication, user information supports backend configuration or database persistence.
- All frontend/backend API calls must include authentication information (e.g., JWT/Session/Cookie).
- Login failure automatically blocks all frontend/backend resource access.

### 2. Technology Stack

- Frontend: React + Ant Design (or similar table components)
- Backend: Go Gin/Chi provides RESTful API, supports future websocket/authentication upgrades

### 3. Frontend Directory and Page Design

- /login Login page
- /clients Online client monitoring list
- /sessions History record display page
- Common UI components and API encapsulation

### 4. Main Features and Interfaces

- Real-time display of online clients, data channels, quick operations
- Query and download historical logs
- Basic management capabilities like one-click disconnect, refresh, etc.

### 5. Development Phase Suggestions

- Phase 1: Login authentication page, mock data-driven list development
- Phase 2: API integration, add historical logs, gradually improve
- Phase 3: UI optimization, permissions and production security

> Login security is the primary requirement for production scenarios. It is recommended to implement it early in the project to prevent any unauthorized/bypass access risks.

## Port Registration Protocol and Traffic Relay Implementation

### 1. Port Registration Protocol

- After client startup, send RegisterRequest message through control channel for port mapping registration. Example:
  ```json
  {
    "type": "register",
    "local_port": 22,
    "remote_port": 10022,
    "protocol": "tcp",
    "token": "changeme",
    "name": "gotunnel-demo"
  }
  ```
- After server receives it, reply with RegisterResponse (status: ok/fail), and dynamically listen on remote_port, update mapping table.

### 2. Bidirectional Traffic Relay Design

- After data channel is established, use RelayConn(a, b) in core/relay.go for fully transparent TCP data forwarding.
- RelayConn ensures full-duplex byte stream forwarding, supports all high-level protocols (e.g., ssh, http). Core code:
  ```go
  func RelayConn(a, b net.Conn) {
      go func() { io.Copy(b, a); b.Close() }()
      io.Copy(a, b); a.Close()
  }
  ```
- This ensures that when accessing server:remote_port from public network, it's like directly connecting to client:local_port, meeting requirements for long connections and large data volumes of protocols like SSH.

For more specific protocol/relay flow details, see detailed comments in protocol.go and core/relay.go.

