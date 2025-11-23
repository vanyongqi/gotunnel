# gotunnel Design Notes and Best Practices

**Language:** [English](./07-DESIGN-NOTES.md) | [中文](../zh/07-DESIGN-NOTES.md)

This document summarizes key design considerations, technical decisions, and best practices during the design and implementation of the gotunnel project, for reference by developers and maintainers.

## I. Core Design Principles

### 1. Dual-Channel Architecture Design

**Design Philosophy:**
- **Control Channel**: For registration, heartbeat, state synchronization, and other control messages (long connection)
- **Data Channel**: For actual business traffic forwarding (established on demand)

**Why This Design?**
- Control channel maintains long connection for real-time management and monitoring
- Data channel established on demand to avoid resource waste
- Separation design facilitates future extensions (e.g., multiplexing, load balancing)

**Considerations:**
- When control channel disconnects, all data channels should be closed immediately
- Data channel exceptions should not affect control channel stability

### 2. Message Boundary Handling

**Problem Background:**
TCP is a byte stream protocol. Direct transmission of JSON or binary messages will encounter packet sticking/fragmentation issues.

**Solution:**
Adopt "length prefix" protocol:
```
[4-byte length (big-endian)] + [message body (JSON/binary)]
```

**Implementation Points:**
```go
// Send: write length first, then content
binary.BigEndian.PutUint32(lenBuf[:], uint32(len(payload)))
w.Write(lenBuf[:])
w.Write(payload)

// Receive: read length first, then complete content
io.ReadFull(r, lenBuf[:])
length := binary.BigEndian.Uint32(lenBuf[:])
io.ReadFull(r, buf[:length])
```

**Key Considerations:**
- Use `io.ReadFull` to ensure complete data reading
- Length field uses big-endian (BigEndian) for cross-platform compatibility
- Set maximum message length limit (e.g., 0x7fffffff) to prevent malicious large packet attacks

### 3. Fully Transparent Data Forwarding

**Design Goal:**
Support any TCP protocol (SSH, HTTP, MySQL, Redis, etc.) without any protocol parsing.

**Implementation:**
```go
func RelayConn(a, b net.Conn) {
    // Bidirectional forwarding: a->b and b->a simultaneously
    go func() {
        io.Copy(b, a)
        b.Close()
    }()
    io.Copy(a, b)
    a.Close()
}
```

**Key Considerations:**
- Use `io.Copy` for efficient data copying
- Must be bidirectional forwarding (full-duplex)
- When either end closes, immediately close the other end
- Use goroutine for concurrent forwarding

## II. High Availability Mechanisms

### 1. Heartbeat Keepalive Mechanism

**Design Purpose:**
- Detect if connection is alive
- Timely detect network anomalies
- Automatically clean up zombie connections

**Implementation Strategy:**

**Client:**
- Send ping every 10 seconds
- Update status after receiving pong
- Trigger reconnection on send failure

**Server:**
- Check all client heartbeats every 5 seconds
- Disconnect if no heartbeat received for more than 30 seconds
- Automatically clean up mapping table

**Considerations:**
- Heartbeat interval should be balanced: too short wastes resources, too long causes detection delay
- Timeout should be reasonable: too short causes false positives, too long causes resource leaks
- Heartbeat failure should immediately trigger reconnection, don't wait

### 2. Automatic Reconnection Mechanism

**Design Goal:**
Automatically recover connection on network anomalies to ensure service availability.

**Implementation Strategy:**
- Exponential backoff: 1s → 2s → 4s → 8s → 16s (max)
- Random jitter: avoid multiple clients reconnecting simultaneously causing storms
- Maximum retry count: prevent infinite retries

**Considerations:**
- After successful reconnection, need to re-register ports
- Health probe should be paused during reconnection
- Reconnection failures should be logged for troubleshooting

### 3. Health Probe Mechanism

**Design Purpose:**
- Detect if local service is alive
- Avoid "black hole proxy" (local service dead but mapping still exists)
- Automatically online/offline port mappings

**Implementation Strategy:**
- Check local port every 30 seconds
- Port down → notify server to offline
- Port recovered → notify server to re-online

**Considerations:**
- Probe interval should be reasonable, avoid frequent checks
- Probe failure should not immediately offline, can set consecutive failure count threshold
- After probe recovery, should immediately notify server to online

## III. Concurrency Safety

### 1. Shared Resource Protection

**Problem:**
Multiple goroutines accessing `mappingTable` simultaneously causes data races.

**Solution:**
```go
var mappingTable = make(map[int]*Mapping)
var mappingTableMu sync.Mutex

// Lock when accessing
mappingTableMu.Lock()
defer mappingTableMu.Unlock()
// Operate on mappingTable
```

**Considerations:**
- Lock granularity should be small, avoid holding lock for long time
- Use `defer` to ensure lock is always released
- Avoid network I/O operations while holding lock

### 2. Connection Management

**Design Principles:**
- Each client connection handled by independent goroutine
- Clean up all related resources when connection closes
- Use `defer` to ensure resource release

**Considerations:**
- Don't use `defer` in loops (causes resource leaks)
- After connection closes, immediately delete from mapping table
- Ignore errors when closing connections (`_ = conn.Close()`)

## IV. Error Handling

### 1. Unified Error Codes

**Design Purpose:**
Facilitate error location and problem troubleshooting.

**Implementation:**
```go
const (
    ErrConnectServer = 1001  // Failed to connect to server
    ErrAuthFailed    = 1002  // Authentication failed
)
```

**Considerations:**
- Error codes must be unique
- Error messages should be clear and explicit
- Record detailed error context

### 2. Error Handling Strategy

**Network Errors:**
- Connection failure → automatic reconnection
- Read timeout → close connection
- Write failure → log and close

**Business Errors:**
- Authentication failure → disconnect immediately, no reconnection
- Registration failure → log, can retry
- Protocol error → log, close connection

## V. Performance Optimization

### 1. Connection Pool Management

**Current Implementation:**
- Control channel: one long connection per client
- Data channel: established on demand, closed after use

**Optimization Suggestions:**
- Consider data channel connection pool in the future
- Control channel can support multiplexing (one control channel manages multiple data channels)

### 2. Memory Management

**Considerations:**
- Release unused connections and buffers in time
- Avoid long-term holding of large objects
- Use object pool to reduce GC pressure (if needed)

### 3. Network Optimization

**Considerations:**
- Use `io.Copy` for efficient data copying
- Avoid unnecessary data copying
- Reasonably set buffer size

## VI. Security Considerations

### 1. Authentication Mechanism

**Current Implementation:**
- Token authentication (simple but effective)

**Considerations:**
- Token should be complex enough to avoid guessing
- Must modify default token in production environment
- Consider TLS encryption for control channel in the future

### 2. Resource Limits

**Considerations:**
- Limit maximum message length (prevent large packet attacks)
- Limit number of port mappings per client
- Limit concurrent connections

### 3. Log Security

**Considerations:**
- Don't log sensitive information (token, password, etc.)
- Use appropriate log level in production environment
- Regularly clean up log files

## VII. Code Quality

### 1. Test Coverage

**Goal:**
- Unit test coverage > 80%
- Core functions 100% coverage

**Considerations:**
- Tests should cover normal and abnormal flows
- Use mock objects to isolate external dependencies
- Tests should be independent, not dependent on execution order

### 2. Code Standards

**Follow Standards:**
- Google Go programming standards
- Use `golangci-lint` for code checking
- Use `gofmt` to format code

**Considerations:**
- Exported functions must have comments
- Errors must be explicitly handled
- Avoid using `_` to ignore important errors

### 3. Maintainability

**Design Principles:**
- Single responsibility for functions
- Clear code structure
- Easy to extend and modify

**Considerations:**
- Avoid overly long functions (recommend < 100 lines)
- Use meaningful variable names
- Add necessary comments

## VIII. Extensibility Design

### 1. Protocol Extension

**Current:**
- JSON format control messages

**Future:**
- Support Protobuf (performance optimization)
- Support custom binary protocol

**Design Considerations:**
- Separate protocol layer from business layer
- Support protocol version negotiation
- Backward compatibility

### 2. Feature Extension

**Planned:**
- Web management UI
- Multiplexing
- UDP protocol support
- TLS encryption

**Design Principles:**
- New features should not affect existing features
- Control features through configuration switches
- Maintain API stability

## IX. Common Pitfalls and Considerations

### 1. Using defer in Loops

**Wrong Example:**
```go
for {
    conn := ln.Accept()
    defer conn.Close()  // ❌ Wrong: defer in loop won't execute
}
```

**Correct Approach:**
```go
for {
    conn := ln.Accept()
    go handleConnection(conn)  // Use defer in function
}

func handleConnection(conn net.Conn) {
    defer conn.Close()  // ✅ Correct
    // ...
}
```

### 2. Error Checking

**Wrong Example:**
```go
conn.Close()  // ❌ Ignored error
```

**Correct Approach:**
```go
_ = conn.Close()  // ✅ Explicitly ignore (if error is not important)
// Or
if err := conn.Close(); err != nil {
    log.Printf("Failed to close connection: %v", err)
}
```

### 3. Concurrency Safety

**Wrong Example:**
```go
mappingTable[port] = mapping  // ❌ Not concurrency safe
```

**Correct Approach:**
```go
mappingTableMu.Lock()
defer mappingTableMu.Unlock()
mappingTable[port] = mapping  // ✅ Protected by lock
```

## X. Best Practices Summary

1. **Message Boundaries**: Must use length prefix protocol to solve TCP packet sticking issues
2. **Resource Management**: Use `defer` to ensure resource release, but don't use in loops
3. **Concurrency Safety**: Shared resources must be protected by locks
4. **Error Handling**: Explicitly handle all errors, don't ignore important errors
5. **Heartbeat Keepalive**: Reasonably set heartbeat interval and timeout
6. **Automatic Reconnection**: Use exponential backoff to avoid reconnection storms
7. **Health Probe**: Regularly check local service status, avoid black hole proxy
8. **Code Quality**: Maintain high test coverage, follow code standards
9. **Security**: Use strong tokens, limit resource usage
10. **Extensibility**: Consider future extensions in design, maintain interface stability

---

**Document Version:** v1.0  
**Last Updated:** 2024-01-XX  
**Maintainer:** gotunnel development team

