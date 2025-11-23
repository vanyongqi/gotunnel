# gotunnel v1.0.0 首次发布功能清单

## 📦 核心功能

### 1. TCP 隧道代理
- ✅ **TCP 协议支持**：支持任意 TCP 服务的端口映射
- ✅ **控制通道 + 数据通道分离架构**：控制通道负责注册、心跳、健康检查；数据通道负责实际数据传输
- ✅ **端口映射**：客户端本地端口 ↔ 服务端远程端口的一对一映射
- ✅ **多客户端并发支持**：服务端可同时管理多个客户端连接

### 2. 通信协议
- ✅ **自定义二进制协议**：4字节长度头 + JSON payload 格式
- ✅ **消息类型支持**：
  - `register` / `register_resp`：端口注册请求/响应
  - `ping` / `pong`：心跳包
  - `offline_port`：端口下线通知
  - `online_port`：端口上线通知
  - `open_data_channel`：数据通道建立指令

### 3. 高可用机制
- ✅ **心跳保活机制**：
  - 客户端每 10 秒发送 ping 包
  - 服务端收到 ping 立即回复 pong
  - 服务端检测客户端心跳超时（30秒）自动断开连接
- ✅ **自动重连机制**：
  - 指数退避算法（baseInterval → maxInterval）
  - 随机 jitter 避免惊群效应
  - 支持最大重试次数限制或无限重试

### 4. 健康检查
- ✅ **本地端口健康探针**：
  - 定时 TCP 连接探测（默认 30 秒间隔）
  - 检测本地服务是否存活
- ✅ **自动上下线**：
  - 检测到端口 down → 自动发送 `offline_port` 通知服务端停止监听
  - 检测到端口恢复 → 自动发送 `online_port` 通知服务端重新监听

### 5. 认证与安全
- ✅ **Token 认证**：客户端注册时验证 token，认证失败拒绝连接
- ✅ **连接管理**：服务端维护端口映射表，支持动态添加/删除映射

### 6. 配置管理
- ✅ **YAML 配置文件**：使用 viper 支持 YAML 格式配置
- ✅ **服务端配置项**：
  - `server.addr`：监听地址（默认 `0.0.0.0:17000`）
  - `server.token`：认证 token（默认 `changeme`）
  - `server.log_level`：日志级别（debug/info/warn/error）
  - `server.log_lang`：日志语言（zh/en）
- ✅ **客户端配置项**：
  - `client.name`：客户端名称（默认 `gotunnel-client-demo`）
  - `client.token`：认证 token（必须与服务端一致）
  - `client.server_addr`：服务端地址（默认 `127.0.0.1:17000`）
  - `client.local_ports`：本地端口列表（数组，当前仅支持单个端口）
  - `client.remote_port`：远程端口（默认 `10022`）
  - `client.log_level`：日志级别
  - `client.log_lang`：日志语言

### 7. 日志系统
- ✅ **结构化日志**：支持 Debug/Info/Warn/Error 四个级别
- ✅ **国际化支持（i18n）**：
  - 支持中文（zh）和英文（en）切换
  - 使用 go-i18n/v2 库，TOML 格式翻译文件
  - 所有日志和错误消息均支持多语言
- ✅ **日志输出**：标准输出（stdout/stderr）

### 8. 错误处理
- ✅ **统一错误码**：
  - `ErrConnectFailed (1001)`：连接失败
  - `ErrAuthFailed (1002)`：认证失败
- ✅ **错误消息国际化**：所有错误消息支持中英文切换

## 🏗️ 架构特性

### 1. 模块化设计
- ✅ **pkg/core**：核心转发逻辑（RelayConn）
- ✅ **pkg/protocol**：协议编解码（WritePacket/ReadPacket）
- ✅ **pkg/ha**：高可用模块（心跳、重连）
- ✅ **pkg/health**：健康检查模块（TCP 探针）
- ✅ **pkg/log**：日志模块（结构化日志 + i18n）
- ✅ **pkg/errors**：错误处理模块

### 2. 并发安全
- ✅ **互斥锁保护**：端口映射表使用 `sync.Mutex` 保护
- ✅ **Goroutine 管理**：心跳、健康检查、数据转发均使用独立 goroutine

### 3. 资源管理
- ✅ **连接关闭**：所有网络连接正确关闭，避免资源泄漏
- ✅ **优雅退出**：支持通过 stop channel 控制 goroutine 退出

## 📊 代码质量

### 1. 测试覆盖
- ✅ **单元测试覆盖率 84.4%**：
  - `pkg/core`: 100%
  - `pkg/errors`: 100%
  - `pkg/health`: 100%
  - `pkg/protocol`: 100%
  - `pkg/ha`: 91.4%
  - `pkg/log`: 90.5%
  - `cmd/server`: 83.1%
  - `cmd/client`: 72.6%

### 2. 代码规范
- ✅ **Go 代码规范**：通过 golangci-lint 检查
- ✅ **代码格式化**：所有代码通过 gofmt 格式化
- ✅ **Linter 规则**：
  - `revive`：代码风格检查
  - `govet`：静态分析
  - `errcheck`：错误检查
  - `gofmt`：格式检查

### 3. CI/CD
- ✅ **GitHub Actions**：
  - Lint 检查（golangci-lint）
  - 单元测试（Go 1.21.x, 1.22.x）
  - 代码覆盖率上传（Codecov）
- ✅ **徽章支持**：
  - CI 状态
  - 代码覆盖率
  - Go Report Card
  - Go 版本
  - License
  - GitHub Release
  - GitHub Stars

## 📚 文档

### 1. 用户文档
- ✅ **README.md**：项目概述和快速开始
- ✅ **快速开始指南**（中英文）：
  - `doc/zh/01-QUICKSTART.md`
  - `doc/en/01-QUICKSTART.md`
- ✅ **配置指南**（中英文）：
  - `doc/zh/02-CONFIG.md`
  - `doc/en/02-CONFIG.md`
- ✅ **架构文档**（中英文）：
  - `doc/zh/03-ARCHITECTURE.md`
  - `doc/en/03-ARCHITECTURE.md`
- ✅ **协议文档**（中英文）：
  - `doc/zh/04-PROTOCOL.md`
  - `doc/en/04-PROTOCOL.md`
- ✅ **开发指南**（中英文）：
  - `doc/zh/05-DEVELOPMENT.md`
  - `doc/en/05-DEVELOPMENT.md`
- ✅ **故障排查**（中英文）：
  - `doc/zh/06-TROUBLESHOOTING.md`
  - `doc/en/06-TROUBLESHOOTING.md`
- ✅ **设计说明**（中英文）：
  - `doc/zh/07-DESIGN-NOTES.md`
  - `doc/en/07-DESIGN-NOTES.md`
- ✅ **Kubernetes 部署指南**（中英文）：
  - `doc/zh/08-K8S-DEPLOYMENT.md`
  - `doc/en/08-K8S-DEPLOYMENT.md`

### 2. 代码文档
- ✅ **函数注释**：所有公共函数均有注释
- ✅ **包文档**：每个包都有 package 级别文档

## 🚀 构建与部署

### 1. 构建
- ✅ **Go 版本要求**：Go 1.21+
- ✅ **构建命令**：
  ```bash
  go build -o gotunnel-server ./cmd/server
  go build -o gotunnel-client ./cmd/client
  ```

### 2. 运行
- ✅ **服务端**：`./gotunnel-server`
- ✅ **客户端**：`./gotunnel-client`

### 3. 依赖管理
- ✅ **go.mod**：Go modules 依赖管理
- ✅ **主要依赖**：
  - `github.com/spf13/viper`：配置管理
  - `github.com/nicksnyder/go-i18n/v2`：国际化
  - `golang.org/x/text`：语言标签

## 📝 已知限制

1. **单端口映射**：当前版本仅支持单个端口映射，多端口需要扩展代码
2. **仅 TCP 协议**：当前仅支持 TCP 协议，HTTP/HTTPS 需要额外实现
3. **无 Web UI**：当前无管理界面，后续版本计划添加
4. **无持久化存储**：端口映射信息仅保存在内存中，服务重启后需重新注册

## 🔄 后续计划

- 🚧 **Phase 2**：Web 管理界面
- 📋 **Phase 3**：云原生扩展（Kubernetes Operator、Service Mesh 集成等）

## 📄 License

MIT License - see [LICENSE](LICENSE) file for details

