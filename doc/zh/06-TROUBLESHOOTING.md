# gotunnel 故障排查指南

**Language:** [English](../en/06-TROUBLESHOOTING.md) | [中文](./06-TROUBLESHOOTING.md)

## 一、连接问题

### 1. 客户端无法连接服务端

**症状：**
```
[ERROR][1001] 无法连接服务端: dial tcp: connection refused
```

**可能原因：**
- 服务端未启动
- 服务端地址配置错误
- 防火墙阻止连接
- 网络不通

**解决方案：**
1. 检查服务端是否运行：`ps aux | grep gotunnel-server`
2. 检查服务端监听端口：`netstat -an | grep 17000`
3. 检查防火墙规则
4. 使用 `telnet` 或 `nc` 测试连接：
   ```bash
   telnet server-ip 17000
   ```

### 2. 认证失败

**症状：**
```
[ERROR][1002] 认证失败，请检查token
```

**可能原因：**
- 客户端和服务端的 token 不一致
- token 配置错误

**解决方案：**
1. 检查 `config.yaml` 中 client 和 server 的 token 是否一致
2. 确认 token 没有多余空格或换行
3. 重新生成并配置 token

### 3. 心跳超时

**症状：**
```
[gotunnel][server] 客户端端口 XXX 心跳超时，主动关掉映射
```

**可能原因：**
- 网络不稳定
- 防火墙/NAT 设备断开长连接
- 客户端进程异常

**解决方案：**
1. 检查网络连接稳定性
2. 检查防火墙/NAT 的 keepalive 设置
3. 检查客户端进程是否正常运行
4. 增加心跳超时时间（代码中修改 `heartbeatTimeout`）

## 二、端口映射问题

### 1. 端口已被占用

**症状：**
```
[server] 监听端口失败: bind: address already in use
```

**解决方案：**
1. 查找占用端口的进程：
   ```bash
   lsof -i:17000
   # 或
   netstat -anp | grep 17000
   ```
2. 修改配置文件中的端口号
3. 或停止占用端口的进程

### 2. 映射端口无法访问

**症状：**
- 通过服务端端口无法访问内网服务
- 连接超时或连接被拒绝

**可能原因：**
- 客户端本地服务未启动
- 本地端口配置错误
- 数据通道建立失败

**解决方案：**
1. 检查客户端本地服务是否运行：
   ```bash
   # 检查SSH服务
   systemctl status sshd
   # 或
   netstat -an | grep :22
   ```
2. 检查 `config.yaml` 中的 `local_ports` 配置
3. 查看客户端日志，确认数据通道是否建立成功

### 3. 健康探针误报

**症状：**
- 端口实际可用，但被健康探针标记为下线

**解决方案：**
1. 检查本地服务是否真的可用
2. 调整健康检查间隔（代码中修改 `healthCheckInterval`）
3. 检查防火墙是否阻止本地连接

## 三、性能问题

### 1. 数据传输慢

**可能原因：**
- 网络带宽限制
- 服务端/客户端资源不足
- 并发连接过多

**解决方案：**
1. 检查网络带宽和延迟
2. 监控 CPU 和内存使用情况
3. 限制并发连接数（计划中）

### 2. 内存占用高

**可能原因：**
- 连接未正确关闭
- 数据缓冲区过大
- 内存泄漏

**解决方案：**
1. 检查是否有连接泄漏
2. 使用 `pprof` 分析内存使用：
   ```bash
   go tool pprof http://localhost:6060/debug/pprof/heap
   ```

## 四、日志分析

### 1. 启用详细日志

修改 `config.yaml`：
```yaml
server:
  log_level: "debug"
```

### 2. 关键日志信息

**服务端：**
- `[gotunnel][server] 控制通道监听`: 服务端启动成功
- `[gotunnel][server] 注册成功`: 客户端注册成功
- `[gotunnel][server] 公网端口监听开启`: 端口映射生效

**客户端：**
- `[gotunnel][client] 端口注册成功`: 注册成功
- `[gotunnel][client] 收到数据通道指令`: 数据通道建立
- `[health] 端口 XXX 不可达`: 健康检查失败

## 五、常见错误码

| 错误码 | 说明 | 解决方案 |
|--------|------|----------|
| 1001 | 连接服务端失败 | 检查网络和服务端状态 |
| 1002 | 认证失败 | 检查 token 配置 |

## 六、调试工具

### 1. 网络工具

```bash
# 测试端口连通性
nc -zv server-ip 17000

# 抓包分析
sudo tcpdump -i any port 17000 -w capture.pcap

# 查看连接状态
netstat -an | grep ESTABLISHED
```

### 2. Go 调试工具

```bash
# 使用 delve 调试
dlv debug ./cmd/server/main.go

# 性能分析
go tool pprof http://localhost:6060/debug/pprof/profile
```

## 七、获取帮助

如果以上方法无法解决问题：

1. 查看项目 Issues：https://github.com/vanyongqi/gotunnel/issues
2. 提交新的 Issue，包含：
   - 错误日志
   - 配置文件（隐藏敏感信息）
   - 系统环境信息
   - 复现步骤

## 八、预防措施

1. **定期检查日志**：及时发现潜在问题
2. **监控资源使用**：CPU、内存、网络带宽
3. **备份配置**：定期备份 `config.yaml`
4. **更新版本**：及时更新到最新版本
5. **安全加固**：使用强 token，启用 TLS（计划中）

