# gotunnel 开发指南

**Language:** [English](../en/05-DEVELOPMENT.md) | [中文](./05-DEVELOPMENT.md)

## 一、项目结构

```
gotunnel/
├── cmd/                    # 可执行程序入口
│   ├── server/            # 服务端主程序
│   │   └── main.go
│   └── client/            # 客户端主程序
│       └── main.go
├── pkg/                    # 核心功能包
│   ├── core/              # 核心转发功能
│   │   ├── relay.go       # 数据中继实现
│   │   └── relay_test.go
│   ├── protocol/          # 协议定义和编解码
│   │   ├── protocol.go
│   │   └── protocol_test.go
│   ├── ha/                # 高可用机制
│   │   ├── heartbeat.go   # 心跳包管理
│   │   ├── reconnect.go   # 自动重连
│   │   └── *_test.go
│   ├── health/            # 健康检查
│   │   ├── probe.go       # 端口健康探针
│   │   └── probe_test.go
│   └── errors/            # 统一错误处理
│       ├── errors.go
│       └── errors_test.go
├── config.yaml            # 配置文件
├── go.mod                 # Go模块定义
├── README.md              # 项目说明
└── doc/                   # 文档目录
    ├── 00-README.md        # 文档中心首页
    ├── 01-QUICKSTART.md   # 快速开始
    ├── 02-CONFIG.md       # 配置说明
    ├── 03-ARCHITECTURE.md # 架构设计文档
    ├── 04-PROTOCOL.md     # 协议文档
    ├── 05-DEVELOPMENT.md  # 开发指南（本文档）
    └── 06-TROUBLESHOOTING.md # 故障排查
```

## 二、开发环境搭建

### 1. 安装依赖

```bash
go mod download
```

### 2. 运行测试

```bash
# 运行所有测试
go test ./...

# 运行测试并查看覆盖率
go test ./pkg/... -cover

# 运行特定包的测试
go test ./pkg/protocol -v
```

### 3. 代码规范

- 遵循 Go 官方代码规范
- 使用 `gofmt` 格式化代码
- 所有导出函数必须有注释
- 单元测试覆盖率目标：>85%

## 三、核心模块说明

### 1. 协议层（pkg/protocol）

**职责：**
- 定义所有控制消息结构体
- 实现消息编解码（WritePacket/ReadPacket）
- 解决 TCP 粘包/分包问题

**关键函数：**
- `WritePacket(w io.Writer, payload []byte) error`: 写入一条完整消息
- `ReadPacket(r io.Reader) ([]byte, error)`: 读取一条完整消息

### 2. 核心转发（pkg/core）

**职责：**
- 实现 TCP 数据流的双向转发
- 支持长连接和大量数据传输

**关键函数：**
- `RelayConn(a, b net.Conn)`: 在两个连接之间进行全双工数据转发

### 3. 高可用机制（pkg/ha）

**职责：**
- 心跳包发送和检测
- 自动重连机制（指数回退 + jitter）
- 连接健康监控

**关键组件：**
- `HeartbeatManager`: 心跳管理器
- `ReconnectLoop`: 自动重连循环

### 4. 健康检查（pkg/health）

**职责：**
- 检测本地端口可用性
- 自动下线/上线端口映射

**关键函数：**
- `ProbeTCPAlive(addr string, timeout time.Duration) bool`: TCP 端口探活
- `PeriodicProbe(...)`: 周期性健康检查

## 四、开发流程

### 1. 添加新功能

1. 在对应的 `pkg/` 目录下实现功能
2. 编写单元测试（覆盖率 >85%）
3. 更新相关文档
4. 提交代码并推送到仓库

### 2. 修改协议

1. 在 `pkg/protocol/protocol.go` 中添加新的消息结构体
2. 更新 `cmd/client/main.go` 和 `cmd/server/main.go` 中的处理逻辑
3. 更新 `doc/04-PROTOCOL.md` 文档
4. 确保向后兼容（如需要）

### 3. 调试技巧

**启用详细日志：**
```yaml
server:
  log_level: "debug"
```

**使用 Go 调试器：**
```bash
dlv debug ./cmd/server/main.go
```

**网络抓包：**
```bash
# 使用 tcpdump 抓包
sudo tcpdump -i any -w capture.pcap port 17000
```

## 五、测试指南

### 1. 单元测试

所有 `pkg/` 下的包都有对应的 `*_test.go` 文件。

**运行测试：**
```bash
go test ./pkg/... -v
```

**查看覆盖率：**
```bash
go test ./pkg/... -coverprofile=cover.out
go tool cover -func=cover.out
```

### 2. 集成测试

**本地测试流程：**

1. 启动服务端：
   ```bash
   go run cmd/server/main.go
   ```

2. 启动客户端：
   ```bash
   go run cmd/client/main.go
   ```

3. 测试 SSH 穿透：
   ```bash
   ssh user@127.0.0.1 -p 10022
   ```

### 3. 性能测试

**并发连接测试：**
```bash
# 使用 Apache Bench 测试
ab -n 1000 -c 100 http://server-ip:10022/
```

## 六、代码贡献

### 1. 提交规范

使用 [Conventional Commits](https://www.conventionalcommits.org/) 规范：

- `feat`: 新功能
- `fix`: 修复bug
- `docs`: 文档更新
- `refactor`: 代码重构
- `test`: 测试相关
- `chore`: 构建/工具相关

**示例：**
```
feat: 添加多端口映射支持
fix: 修复心跳超时问题
docs: 更新快速开始文档
```

### 2. Pull Request 流程

1. Fork 仓库
2. 创建功能分支
3. 提交代码并编写测试
4. 确保所有测试通过
5. 提交 Pull Request

## 七、常见开发问题

### 1. 依赖问题

```bash
# 更新依赖
go get -u ./...

# 清理依赖
go mod tidy
```

### 2. 编译问题

```bash
# 清理构建缓存
go clean -cache

# 重新编译
go build ./cmd/server
```

### 3. 测试失败

- 检查是否有端口冲突
- 检查网络连接
- 查看测试日志输出

## 八、下一步开发计划

参考 [03-架构文档](./03-ARCHITECTURE.md) 中的开发阶段规划：

- **阶段一**：核心功能实现（已完成）
- **阶段二**：Web 管理 UI（计划中）
- **阶段三**：云原生扩展（计划中）

## 九、参考资源

- [Go 官方文档](https://golang.org/doc/)
- [frp 源码](https://github.com/fatedier/frp)
- [ngrok 架构](https://ngrok.com/product)

