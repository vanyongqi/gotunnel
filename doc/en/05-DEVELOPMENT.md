# gotunnel Development Guide

**Language:** [English](./05-DEVELOPMENT.md) | [中文](../zh/05-DEVELOPMENT.md)

## Project Structure

```
gotunnel/
├── cmd/                    # Executable entry points
│   ├── server/            # Server main program
│   │   └── main.go
│   └── client/            # Client main program
│       └── main.go
├── pkg/                    # Core functionality packages
│   ├── core/              # Core forwarding functionality
│   │   ├── relay.go       # Data relay implementation
│   │   └── relay_test.go
│   ├── protocol/          # Protocol definition and encoding/decoding
│   │   ├── protocol.go
│   │   └── protocol_test.go
│   ├── ha/                # High availability mechanisms
│   │   ├── heartbeat.go   # Heartbeat packet management
│   │   ├── reconnect.go   # Auto-reconnect
│   │   └── *_test.go
│   ├── health/            # Health checks
│   │   ├── probe.go       # Port health probe
│   │   └── probe_test.go
│   └── errors/            # Unified error handling
│       ├── errors.go
│       └── errors_test.go
├── config.yaml            # Configuration file
├── go.mod                 # Go module definition
├── README.md              # Project description
└── doc/                   # Documentation directory
    ├── en/                # English documentation
    └── zh/                # Chinese documentation
```

## Development Environment Setup

### 1. Install Dependencies

```bash
go mod download
```

### 2. Run Tests

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test ./pkg/... -cover

# Run specific package tests
go test ./pkg/protocol -v
```

### 3. Code Standards

- Follow Go official code standards
- Use `gofmt` to format code
- All exported functions must have comments
- Unit test coverage target: >85%

## Core Modules

### 1. Protocol Layer (pkg/protocol)

**Responsibilities:**
- Define all control message structs
- Implement message encoding/decoding (WritePacket/ReadPacket)
- Solve TCP packet sticking/fragmentation issues

**Key Functions:**
- `WritePacket(w io.Writer, payload []byte) error`: Write a complete message
- `ReadPacket(r io.Reader) ([]byte, error)`: Read a complete message

### 2. Core Forwarding (pkg/core)

**Responsibilities:**
- Implement bidirectional TCP data stream forwarding
- Support long connections and large data transfers

**Key Functions:**
- `RelayConn(a, b net.Conn)`: Full-duplex data forwarding between two connections

### 3. High Availability (pkg/ha)

**Responsibilities:**
- Heartbeat packet sending and detection
- Auto-reconnect mechanism (exponential backoff + jitter)
- Connection health monitoring

**Key Components:**
- `HeartbeatManager`: Heartbeat manager
- `ReconnectLoop`: Auto-reconnect loop

### 4. Health Check (pkg/health)

**Responsibilities:**
- Detect local port availability
- Auto offline/online port mappings

**Key Functions:**
- `ProbeTCPAlive(addr string, timeout time.Duration) bool`: TCP port probe
- `PeriodicProbe(...)`: Periodic health check

## Development Workflow

### 1. Adding New Features

1. Implement functionality in corresponding `pkg/` directory
2. Write unit tests (coverage >85%)
3. Update related documentation
4. Commit code and push to repository

### 2. Modifying Protocol

1. Add new message structs in `pkg/protocol/protocol.go`
2. Update handling logic in `cmd/client/main.go` and `cmd/server/main.go`
3. Update `doc/04-PROTOCOL.md` documentation
4. Ensure backward compatibility (if needed)

### 3. Debugging Tips

**Enable verbose logging:**
```yaml
server:
  log_level: "debug"
```

**Use Go debugger:**
```bash
dlv debug ./cmd/server/main.go
```

**Network packet capture:**
```bash
# Use tcpdump to capture packets
sudo tcpdump -i any -w capture.pcap port 17000
```

## Testing Guide

### 1. Unit Tests

All packages under `pkg/` have corresponding `*_test.go` files.

**Run tests:**
```bash
go test ./pkg/... -v
```

**View coverage:**
```bash
go test ./pkg/... -coverprofile=cover.out
go tool cover -func=cover.out
```

### 2. Integration Tests

**Local testing flow:**

1. Start server:
   ```bash
   go run cmd/server/main.go
   ```

2. Start client:
   ```bash
   go run cmd/client/main.go
   ```

3. Test SSH tunnel:
   ```bash
   ssh user@127.0.0.1 -p 10022
   ```

## Contributing

### 1. Commit Standards

Use [Conventional Commits](https://www.conventionalcommits.org/) format:

- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation update
- `refactor`: Code refactoring
- `test`: Test related
- `chore`: Build/tool related

**Examples:**
```
feat: Add multi-port mapping support
fix: Fix heartbeat timeout issue
docs: Update quick start guide
```

### 2. Pull Request Process

1. Fork repository
2. Create feature branch
3. Commit code and write tests
4. Ensure all tests pass
5. Submit Pull Request

## Next Development Plans

Refer to [03-Architecture Design](./03-ARCHITECTURE.md) for development phase planning:

- **Phase 1**: Core functionality implementation (Completed)
- **Phase 2**: Web Management UI (Planned)
- **Phase 3**: Cloud-native extensions (Planned)

## Reference Resources

- [Go Official Documentation](https://golang.org/doc/)
- [frp Source Code](https://github.com/fatedier/frp)
- [ngrok Architecture](https://ngrok.com/product)

