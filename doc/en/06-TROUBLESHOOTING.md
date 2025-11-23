# gotunnel Troubleshooting Guide

**Language:** [English](./06-TROUBLESHOOTING.md) | [中文](../zh/06-TROUBLESHOOTING.md)

## Connection Issues

### 1. Client Cannot Connect to Server

**Symptoms:**
```
[ERROR][1001] Failed to connect to server: dial tcp: connection refused
```

**Possible Causes:**
- Server not started
- Server address misconfigured
- Firewall blocking connection
- Network unreachable

**Solutions:**
1. Check if server is running: `ps aux | grep gotunnel-server`
2. Check server listening port: `netstat -an | grep 17000`
3. Check firewall rules
4. Test connection using `telnet` or `nc`:
   ```bash
   telnet server-ip 17000
   ```

### 2. Authentication Failed

**Symptoms:**
```
[ERROR][1002] Authentication failed, please check token
```

**Possible Causes:**
- Client and server tokens don't match
- Token misconfigured

**Solutions:**
1. Check if tokens in `config.yaml` for client and server match
2. Ensure token has no extra spaces or newlines
3. Regenerate and configure token

### 3. Heartbeat Timeout

**Symptoms:**
```
[gotunnel][server] Client port XXX heartbeat timeout, closing mapping
```

**Possible Causes:**
- Unstable network
- Firewall/NAT device disconnecting long connections
- Client process abnormal

**Solutions:**
1. Check network connection stability
2. Check firewall/NAT keepalive settings
3. Check if client process is running normally
4. Increase heartbeat timeout (modify `heartbeatTimeout` in code)

## Port Mapping Issues

### 1. Port Already in Use

**Symptoms:**
```
[server] Failed to listen on port: bind: address already in use
```

**Solutions:**
1. Find process using the port:
   ```bash
   lsof -i:17000
   # or
   netstat -anp | grep 17000
   ```
2. Modify port in configuration file
3. Or stop the process using the port

### 2. Mapped Port Cannot Be Accessed

**Symptoms:**
- Cannot access internal service via server port
- Connection timeout or connection refused

**Possible Causes:**
- Local service on client not started
- Local port misconfigured
- Data channel establishment failed

**Solutions:**
1. Check if local service on client is running:
   ```bash
   # Check SSH service
   systemctl status sshd
   # or
   netstat -an | grep :22
   ```
2. Check `local_ports` configuration in `config.yaml`
3. Check client logs to confirm data channel establishment

### 3. Health Probe False Positive

**Symptoms:**
- Port is actually available but marked offline by health probe

**Solutions:**
1. Check if local service is really available
2. Adjust health check interval (modify `healthCheckInterval` in code)
3. Check if firewall is blocking local connections

## Performance Issues

### 1. Slow Data Transfer

**Possible Causes:**
- Network bandwidth limitations
- Insufficient server/client resources
- Too many concurrent connections

**Solutions:**
1. Check network bandwidth and latency
2. Monitor CPU and memory usage
3. Limit concurrent connections (planned)

### 2. High Memory Usage

**Possible Causes:**
- Connections not properly closed
- Data buffer too large
- Memory leak

**Solutions:**
1. Check for connection leaks
2. Use `pprof` to analyze memory usage:
   ```bash
   go tool pprof http://localhost:6060/debug/pprof/heap
   ```

## Log Analysis

### 1. Enable Verbose Logging

Modify `config.yaml`:
```yaml
server:
  log_level: "debug"
```

### 2. Key Log Information

**Server:**
- `[gotunnel][server] Control channel listening`: Server started successfully
- `[gotunnel][server] Registration successful`: Client registered successfully
- `[gotunnel][server] Public port listening enabled`: Port mapping active

**Client:**
- `[gotunnel][client] Port registration successful`: Registration successful
- `[gotunnel][client] Received data channel instruction`: Data channel established
- `[health] Port XXX unreachable`: Health check failed

## Common Error Codes

| Error Code | Description | Solution |
|------------|-------------|----------|
| 1001 | Failed to connect to server | Check network and server status |
| 1002 | Authentication failed | Check token configuration |

## Debugging Tools

### 1. Network Tools

```bash
# Test port connectivity
nc -zv server-ip 17000

# Packet capture analysis
sudo tcpdump -i any port 17000 -w capture.pcap

# View connection status
netstat -an | grep ESTABLISHED
```

### 2. Go Debugging Tools

```bash
# Use delve debugger
dlv debug ./cmd/server/main.go

# Performance profiling
go tool pprof http://localhost:6060/debug/pprof/profile
```

## Getting Help

If the above methods cannot solve the problem:

1. Check project Issues: https://github.com/vanyongqi/gotunnel/issues
2. Submit a new Issue, including:
   - Error logs
   - Configuration file (hide sensitive information)
   - System environment information
   - Reproduction steps

## Prevention Measures

1. **Regular log checks**: Discover potential issues early
2. **Monitor resource usage**: CPU, memory, network bandwidth
3. **Backup configuration**: Regularly backup `config.yaml`
4. **Update versions**: Update to latest version in time
5. **Security hardening**: Use strong tokens, enable TLS (planned)

