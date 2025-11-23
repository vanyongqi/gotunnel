[![CI](https://github.com/vanyongqi/gotunnel/actions/workflows/test.yaml/badge.svg)](https://github.com/vanyongqi/gotunnel/actions)
[![codecov](https://codecov.io/gh/vanyongqi/gotunnel/branch/main/graph/badge.svg)](https://codecov.io/gh/vanyongqi/gotunnel)
[![Go Report Card](https://goreportcard.com/badge/github.com/vanyongqi/gotunnel)](https://goreportcard.com/report/github.com/vanyongqi/gotunnel)
[![Go Version](https://img.shields.io/badge/go-1.21%2B-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![GitHub release](https://img.shields.io/github/release/vanyongqi/gotunnel.svg)](https://github.com/vanyongqi/gotunnel/releases)
[![GitHub stars](https://img.shields.io/github/stars/vanyongqi/gotunnel.svg?style=social&label=Star)](https://github.com/vanyongqi/gotunnel)
# gotunnel

gotunnel is a high-performance intranet penetration (Tunnel/Proxy) tool implemented in Go, inspired by frp and ngrok. It enables secure and efficient remote access and management of any number of internal network service nodes through cloud servers.

## Quick Start

```bash
# Build
go build -o gotunnel-server ./cmd/server
go build -o gotunnel-client ./cmd/client

# Start server
./gotunnel-server

# Start client
./gotunnel-client
```

## 📚 Documentation

Complete documentation is available in [doc/](./doc/) directory:

**Language:** [English](./doc/en/00-README.md) | [中文](./doc/zh/00-README.md)

- **[Quick Start](./doc/en/01-QUICKSTART.md)** - Get started in 5 minutes
- **[Configuration Guide](./doc/en/02-CONFIG.md)** - Detailed configuration options
- **[Architecture Design](./doc/en/03-ARCHITECTURE.md)** - System architecture documentation
- **[Protocol Documentation](./doc/en/04-PROTOCOL.md)** - Communication protocol details
- **[Development Guide](./doc/en/05-DEVELOPMENT.md)** - Developer documentation
- **[Troubleshooting Guide](./doc/en/06-TROUBLESHOOTING.md)** - Common issues and solutions

## Core Features

- ✅ TCP/HTTP protocol tunneling support
- ✅ Control channel + data channel separated architecture
- ✅ Heartbeat keepalive + auto-reconnect mechanism
- ✅ Port health probe + auto offline/online
- ✅ Multi-client concurrent support
- ✅ Complete unit tests (89% coverage)

## Project Status

- ✅ Phase 1: Core functionality implementation (Completed)
- 🚧 Phase 2: Web Management UI (Planned)
- 📋 Phase 3: Cloud-native extensions (Planned)

## Reference Projects

- [frp](https://github.com/fatedier/frp) - High-performance reverse proxy application
- [ngrok](https://ngrok.com) - Intranet penetration service
- [lanproxy](https://github.com/ffay/lanproxy) - Java-based intranet penetration tool

## License

MIT License - see [LICENSE](LICENSE) file for details
