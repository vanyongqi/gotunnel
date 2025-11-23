# gotunnel Quick Start Guide

**Language:** [English](./01-QUICKSTART.md) | [中文](../zh/01-QUICKSTART.md)

## Requirements

- Go 1.22 or higher
- Linux/macOS/Windows
- Server requires a public IP

## Build & Run

```bash
# Clone repo
git clone https://github.com/vanyongqi/gotunnel.git
cd gotunnel
# Build
go build -o gotunnel-server ./cmd/server
go build -o gotunnel-client ./cmd/client
```

## Start Server

Edit `config.yaml`:
```
server:
  addr: "0.0.0.0:17000"
  token: "your-token"
```
Run:
```bash
./gotunnel-server
```

## Start Client

Edit `config.yaml`:
```
client:
  name: "my-client"
  token: "your-token"        # same as server
  server_addr: "your-server-ip:17000"
  local_ports: [22]
```
Run:
```bash
./gotunnel-client
```

## Test SSH Tunnel

```bash
ssh user@your-server-ip -p 10022
```

---

For more details, see [02-Configuration Guide](./02-CONFIG.md), [04-Protocol](./04-PROTOCOL.md), [06-Troubleshooting](./06-TROUBLESHOOTING.md).

