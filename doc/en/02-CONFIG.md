# gotunnel Configuration Guide

**Language:** [English](./02-CONFIG.md) | [中文](../zh/02-CONFIG.md)

## File Format

gotunnel uses `config.yaml` in YAML format, in the project root.

## Server Example

```yaml
server:
  addr: "0.0.0.0:17000"
  token: "your-strong-token"
  log_level: "debug"
```

## Client Example

```yaml
client:
  name: "gotunnel-client-demo"
  token: "your-strong-token"
  server_addr: "your-server-ip:17000"
  local_ports: [22, 8080]
```

## Key Parameters

| Key           | Required | Description                         |
|---------------|----------|-------------------------------------|
| server.addr   | yes      | Listen IP and port                  |
| server.token  | yes      | Server token (auth, same as client) |
| client.token  | yes      | Client token (auth, same as server) |
| client.server_addr | yes  | Server endpoint                     |
| client.local_ports | yes  | Ports to expose (list)              |

**Tip:** Token security is crucial! Use strong random strings.

