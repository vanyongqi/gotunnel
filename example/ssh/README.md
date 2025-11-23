# SSH 服务演示场景

本场景演示如何通过 gotunnel 远程访问内网 SSH 服务。

## 🎯 演示目标

- 在本地内网机器上运行 SSH 服务（端口 22）
- 通过 gotunnel 将 SSH 服务映射到公网端口 10022
- 通过公网 IP 访问内网 SSH 服务

## 📋 前置准备

1. **服务端**：有公网 IP 的云服务器
2. **客户端**：本地内网机器（运行 SSH 服务）
3. **SSH 服务**：确保本地 SSH 服务正常运行

## 🚀 演示步骤

### 步骤 1：验证本地 SSH 服务

在客户端机器上验证 SSH 服务是否运行：

```bash
# 检查 SSH 服务状态
sudo systemctl status ssh
# 或
sudo systemctl status sshd

# 测试本地 SSH 连接
ssh localhost
```

如果 SSH 服务未运行，启动它：

```bash
# Ubuntu/Debian
sudo systemctl start ssh

# CentOS/RHEL
sudo systemctl start sshd
```

### 步骤 2：配置服务端

将 `server.yaml` 复制到项目根目录：

```bash
cp example/ssh/server.yaml config.yaml
```

修改配置（如需要）：

```yaml
server:
  addr: "0.0.0.0:17000"     # 监听所有网卡
  log_level: "info"
  log_lang: "zh"
  token: "ssh-demo-token"    # 自定义 token
```

### 步骤 3：配置客户端

将 `client.yaml` 复制到项目根目录（覆盖之前的配置）：

```bash
cp example/ssh/client.yaml config.yaml
```

**重要**：修改 `server_addr` 为实际的服务端公网 IP：

```yaml
client:
  server_addr: "YOUR_SERVER_IP:17000"  # 替换为实际 IP
  local_ports: [22]                   # SSH 默认端口
  remote_port: 10022                  # 公网暴露端口
  token: "ssh-demo-token"             # 必须与服务端一致
```

### 步骤 4：启动服务端

在服务端机器上：

```bash
# 编译（如果还没有）
go build -o gotunnel-server ./cmd/server

# 启动服务端
./gotunnel-server
```

看到以下日志表示启动成功：

```
[INFO][server] 控制通道监听中: 0.0.0.0:17000, Token: ssh-demo-token
```

### 步骤 5：启动客户端

在客户端机器上：

```bash
# 编译（如果还没有）
go build -o gotunnel-client ./cmd/client

# 启动客户端
./gotunnel-client
```

看到以下日志表示连接成功：

```
[INFO][client] 端口已注册: 22 -> 10022
[INFO][client] 端口注册成功
```

### 步骤 6：验证演示

从任意机器（或服务端）通过公网 IP 访问内网 SSH：

```bash
# 通过公网 IP 和映射端口访问内网 SSH
ssh -p 10022 username@YOUR_SERVER_IP

# 例如：
ssh -p 10022 user@123.45.67.89
```

应该能够成功连接到内网的 SSH 服务。

## ⚙️ 端口说明

### 控制通道端口（默认 17000，可配置）
- **作用**：用于客户端注册、心跳、状态同步等控制消息
- **服务端配置**：`server.addr`（默认 `:17000`，可通过配置修改）
- **客户端配置**：`client.server_addr`（必须与服务端 `server.addr` 一致）
- **说明**：控制端口可以修改，但客户端和服务端必须使用相同的端口

### 数据通道端口（完全可自定义）
- **作用**：用于实际业务流量的透明转发
- **客户端配置**：`client.remote_port`（可任意指定可用端口）
- **服务端**：自动监听 `client.remote_port` 指定的端口（接收用户 SSH 连接）
- **转发**：将连接转发到客户端本地 `127.0.0.1:22`
- **说明**：数据端口完全可自定义，只需确保端口未被占用且防火墙已开放

## 🔍 演示检查清单

- [ ] 本地 SSH 服务正常运行（端口 22）
- [ ] 服务端成功启动并监听 17000 端口（控制通道）
- [ ] 客户端成功连接服务端
- [ ] 端口映射注册成功（22 -> 10022）
- [ ] 可以通过公网 IP:10022 访问内网 SSH
- [ ] 服务端防火墙已开放：
  - 17000（控制通道，客户端连接）
  - 10022（数据通道，用户 SSH 连接）

## 🎬 演示技巧

1. **展示 SSH 连接**：
   - 从服务端机器 SSH 到客户端内网机器
   - 展示文件传输、命令执行等功能

2. **展示健康检查**：
   - 停止本地 SSH 服务，观察日志显示端口下线
   - 重新启动 SSH 服务，观察日志显示端口上线

3. **展示心跳机制**：
   - 观察日志中的 ping/pong 消息
   - 展示连接稳定性

4. **展示多客户端**：
   - 可以启动多个客户端连接同一个服务端
   - 每个客户端映射不同的端口

## ⚠️ 常见问题

### 问题1：无法通过 10022 端口连接 SSH

**原因**：服务端防火墙未开放端口

**解决**：
- 在云服务器安全组中开放 10022 端口
- 检查服务端防火墙规则：`sudo ufw allow 10022`

### 问题2：SSH 连接被拒绝

**原因**：客户端 SSH 服务未运行或配置错误

**解决**：
- 检查本地 SSH 服务是否运行：`sudo systemctl status ssh`
- 检查 SSH 配置：`/etc/ssh/sshd_config`
- 确保 SSH 监听在 0.0.0.0 或 127.0.0.1

### 问题3：客户端连接失败

**原因**：服务端地址配置错误

**解决**：
- 检查 `server_addr` 是否正确
- 确保服务端监听 `0.0.0.0` 而不是 `127.0.0.1`
- 检查服务端防火墙是否开放 17000 端口

### 问题4：Token 认证失败

**原因**：客户端和服务端 token 不一致

**解决**：
- 确保 `client.token` 和 `server.token` 完全一致

## 🔒 安全建议

1. **使用强密码 token**：生产环境不要使用 "ssh-demo-token"
2. **限制 SSH 访问**：配置 SSH 密钥认证，禁用密码认证
3. **防火墙规则**：只允许必要的 IP 访问 10022 端口
4. **定期更新**：保持 gotunnel 和服务端系统更新

## 📊 演示效果

演示成功后，你应该能够：

1. ✅ 通过公网 IP 访问内网 SSH 服务
2. ✅ 正常使用 SSH 的所有功能（登录、文件传输等）
3. ✅ 看到清晰的日志输出（支持中英文）
4. ✅ 观察到健康检查和心跳机制的工作
5. ✅ 理解 gotunnel 在远程访问场景中的应用

