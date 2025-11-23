# gotunnel 配置说明文档

**Language:** [English](../en/02-CONFIG.md) | [中文](./02-CONFIG.md)

## 一、配置文件格式

gotunnel 使用 YAML 格式的配置文件，默认文件名为 `config.yaml`，放在项目根目录。

## 二、服务端配置

### 配置项说明

```yaml
server:
  addr: "0.0.0.0:17000"     # 服务端监听地址和端口
  log_level: "debug"         # 日志级别: debug/info/warn/error
  token: "changeme"          # 认证token，必须与客户端一致
```

### 参数详解

| 参数 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| `addr` | string | 否 | `:17000` | 监听地址，`0.0.0.0` 表示监听所有网卡 |
| `log_level` | string | 否 | `debug` | 日志级别，影响输出详细程度 |
| `token` | string | **是** | 无 | 认证token，用于验证客户端身份 |

### 配置示例

**生产环境配置：**
```yaml
server:
  addr: "0.0.0.0:17000"
  log_level: "info"
  token: "your-very-secure-random-token-here"
```

**本地测试配置：**
```yaml
server:
  addr: "127.0.0.1:17000"
  log_level: "debug"
  token: "test-token"
```

## 三、客户端配置

### 配置项说明

```yaml
client:
  name: "gotunnel-client-demo"  # 客户端自定义名称
  token: "changeme"              # 认证token，必须与服务端一致
  server_addr: "127.0.0.1:17000" # 服务端地址
  local_ports: [22]              # 要映射的本地端口列表
```

### 参数详解

| 参数 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| `name` | string | 否 | `gotunnel-client-demo` | 客户端名称，用于标识和日志 |
| `token` | string | **是** | 无 | 认证token，必须与服务端一致 |
| `server_addr` | string | **是** | 无 | 服务端地址，格式：`IP:端口` |
| `local_ports` | array | **是** | 无 | 要映射的本地端口列表，如 `[22, 8080]` |

### 配置示例

**单端口映射：**
```yaml
client:
  name: "web-server-01"
  token: "your-secret-token"
  server_addr: "120.120.120.120:17000"
  local_ports: [8080]
```

**多端口映射：**
```yaml
client:
  name: "multi-service-client"
  token: "your-secret-token"
  server_addr: "120.120.120.120:17000"
  local_ports: [22, 3306, 8080, 9090]
```

## 四、高级配置

### 环境变量支持（计划中）

未来版本将支持通过环境变量覆盖配置：

```bash
export GOTUNNEL_SERVER_ADDR="0.0.0.0:17000"
export GOTUNNEL_SERVER_TOKEN="your-token"
```

### 配置文件位置

gotunnel 按以下顺序查找配置文件：

1. 当前工作目录的 `config.yaml`
2. 用户主目录的 `~/.gotunnel/config.yaml`（计划中）
3. `/etc/gotunnel/config.yaml`（计划中）

## 五、安全建议

1. **Token 安全**
   - 使用强随机字符串作为 token
   - 定期更换 token
   - 不要将 token 提交到代码仓库

2. **网络安全**
   - 生产环境建议使用 TLS 加密（计划中）
   - 限制服务端监听地址，避免暴露到公网
   - 使用防火墙限制访问来源

3. **权限控制**
   - 以非 root 用户运行服务
   - 限制文件系统权限

## 六、配置验证

启动时会自动验证配置：

- Token 不能为空
- 服务端地址格式正确
- 端口号在有效范围内（1-65535）

配置错误会在启动时输出错误信息并退出。

