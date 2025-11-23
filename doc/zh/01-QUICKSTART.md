# gotunnel 快速开始指南

**Language:** [English](../en/01-QUICKSTART.md) | [中文](./01-QUICKSTART.md)

## 一、环境要求

- Go 1.22 或更高版本
- Linux/macOS/Windows 系统
- 服务端需要公网 IP（或可访问的云服务器）

## 二、快速安装

### 方式1：从源码编译

```bash
# 克隆仓库
git clone https://github.com/vanyongqi/gotunnel.git
cd gotunnel

# 编译服务端
go build -o gotunnel-server ./cmd/server

# 编译客户端
go build -o gotunnel-client ./cmd/client
```

### 方式2：使用预编译二进制（待发布）

下载对应平台的二进制文件，解压后直接使用。

## 三、快速启动

### 1. 配置服务端

编辑 `config.yaml`：

```yaml
server:
  addr: "0.0.0.0:17000"     # 服务端监听地址
  log_level: "debug"
  token: "your-secret-token" # 修改为强密码
```

### 2. 启动服务端

```bash
./gotunnel-server
# 或
go run cmd/server/main.go
```

看到以下输出表示启动成功：
```
[gotunnel][server] 控制通道监听: 0.0.0.0:17000 (token: your-secret-token)
```

### 3. 配置客户端

编辑 `config.yaml`：

```yaml
client:
  name: "my-client"                    # 客户端名称
  token: "your-secret-token"            # 必须与服务端一致
  server_addr: "your-server-ip:17000"  # 服务端地址
  local_ports: [22]                     # 要映射的本地端口
```

### 4. 启动客户端

```bash
./gotunnel-client
# 或
go run cmd/client/main.go
```

看到以下输出表示连接成功：
```
[gotunnel][client] 注册端口: 本地 22 => 公网 10022
[gotunnel][client] 端口注册成功，启动心跳和健康探针...
```

## 四、测试连接

### SSH 穿透测试

```bash
# 通过服务端公网IP访问内网SSH
ssh user@your-server-ip -p 10022
```

### HTTP 服务穿透测试

如果映射的是 Web 服务（如 8080 端口）：

```bash
# 访问内网Web服务
curl http://your-server-ip:10022
```

## 五、常见问题

### 1. 连接失败

- 检查服务端和客户端的 token 是否一致
- 检查防火墙是否开放对应端口
- 检查服务端地址是否正确

### 2. 端口被占用

- 修改 `config.yaml` 中的端口配置
- 或使用 `lsof -i:端口号` 查看占用进程

### 3. 心跳超时

- 检查网络连接是否稳定
- 检查防火墙/NAT 是否允许长连接

## 六、下一步

- 查看 [02-配置文档](./02-CONFIG.md) 了解详细配置选项
- 查看 [03-架构文档](./03-ARCHITECTURE.md) 了解系统设计
- 查看 [05-开发指南](./05-DEVELOPMENT.md) 参与开发

