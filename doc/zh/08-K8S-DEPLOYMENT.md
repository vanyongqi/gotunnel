# gotunnel 在 Kubernetes 跨网络部署中的应用

**Language:** [English](../en/08-K8S-DEPLOYMENT.md) | [中文](./08-K8S-DEPLOYMENT.md)

本文档介绍如何使用 gotunnel 实现 Kubernetes 混合云部署场景，即本地内网 K8s 控制平面管理远程云服务器 worker 节点。

## 一、场景说明

### 架构拓扑

```
┌─────────────────────────────────────────────────────────┐
│  本地内网环境（Controller 节点）                          │
│                                                          │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐ │
│  │ kube-apiserver│  │ etcd         │  │ kube-controller││
│  │ :6443        │  │ :2379-2380   │  │ :10257        │ │
│  └──────────────┘  └──────────────┘  └──────────────┘ │
│                                                          │
│  ┌──────────────────────────────────────────────────┐   │
│  │ gotunnel-server (监听公网端口)                   │   │
│  │ :17000 (控制通道)                                │   │
│  │ :6443 (映射到 worker 的 kube-apiserver)          │   │
│  └──────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────┘
                        │
                        │ 公网
                        │
┌─────────────────────────────────────────────────────────┐
│  云服务器（Worker 节点）                                  │
│                                                          │
│  ┌──────────────────────────────────────────────────┐   │
│  │ gotunnel-client                                   │   │
│  │ 映射: local:6443 -> remote:6443                   │   │
│  └──────────────────────────────────────────────────┘   │
│                                                          │
│  ┌──────────────┐                                       │
│  │ kubelet       │  ──> 连接到 kube-apiserver:6443     │
│  │ kube-proxy    │  ──> 连接到 kube-apiserver:6443     │
│  │ CNI plugins   │  ──> 连接到 kube-apiserver:6443     │
│  └──────────────┘                                       │
└─────────────────────────────────────────────────────────┘
```

### 核心需求

1. **Worker 节点需要连接到 Controller 的 kube-apiserver**
   - Worker 节点的 kubelet、kube-proxy、CNI 等组件都需要访问 kube-apiserver
   - 默认端口：6443（HTTPS）

2. **Controller 需要访问 Worker 节点的 kubelet API**
   - 用于健康检查、日志收集、exec 等操作
   - 默认端口：10250（HTTPS）

3. **网络隔离问题**
   - Controller 在内网，Worker 在公网
   - 两者无法直接通信
   - 需要通过 gotunnel 建立隧道

## 二、部署方案

### 方案一：单向连接（推荐用于测试）

**场景：** Worker 节点主动连接 Controller

**配置步骤：**

#### 1. Controller 端（本地内网）

**启动 gotunnel-server：**

```yaml
# config.yaml (Controller 端)
server:
  addr: "0.0.0.0:17000"  # 监听所有网卡，允许公网访问
  log_level: "info"
  token: "your-very-secure-k8s-token-here"  # 使用强随机 token
```

**启动服务：**
```bash
./gotunnel-server
```

#### 2. Worker 端（云服务器）

**重要说明：** 当前 gotunnel 的工作方式是：Client 将本地端口映射到 Server 的远程端口。对于 K8s 场景，我们需要将 Controller 的 kube-apiserver (6443) 暴露给 Worker 访问。

**有两种实现方式：**

**方式 A：在 Controller 上运行 Client（推荐）**

在 Controller 节点上运行 gotunnel-client，将 Controller 的 6443 端口映射到 Server 的远程端口：

```yaml
# config.yaml (Controller 端，用于映射 kube-apiserver)
client:
  name: "k8s-controller-apiserver"
  token: "your-very-secure-k8s-token-here"
  server_addr: "your-controller-public-ip:17000"
  local_ports: [6443]  # Controller 本地的 kube-apiserver 端口
```

这样，Worker 就可以通过 `your-controller-public-ip:6443` 访问 Controller 的 kube-apiserver。

**方式 B：使用反向代理（如果方式 A 不可行）**

如果 Controller 上不能运行 gotunnel-client，可以在 Controller 上设置一个反向代理，将 6443 端口转发到 gotunnel-server 映射的端口。

### 方案二：双向连接（推荐用于生产）

**场景：** Controller 和 Worker 都需要互相访问

#### 架构设计

```
Controller (内网)                    Worker (公网)
     │                                    │
     │  ┌─────────────────────────────┐   │
     │  │ gotunnel-server            │   │
     │  │ :17000 (控制通道)           │   │
     │  │ :6443 (映射到 worker)       │   │
     │  └─────────────────────────────┘   │
     │           ▲                         │
     │           │                         │
     │           │ 控制通道                 │
     │           │                         │
     │           │                         │
     │  ┌─────────────────────────────┐   │
     │  │ gotunnel-client            │   │
     │  │ 映射: local:6443 -> remote:6443│
     │  └─────────────────────────────┘   │
     │                                    │
     │  kube-apiserver :6443              │  kubelet :10250
     │                                    │
```

#### 1. Controller 端配置

**启动 gotunnel-server：**
```yaml
# config.yaml (Controller 端)
server:
  addr: "0.0.0.0:17000"
  log_level: "info"
  token: "your-very-secure-k8s-token-here"
```

**启动服务：**
```bash
./gotunnel-server
```

#### 2. Controller 端配置（映射 kube-apiserver）

**在 Controller 节点上运行 gotunnel-client：**
```yaml
# config.yaml (Controller 端，用于映射 kube-apiserver)
client:
  name: "k8s-controller-apiserver"
  token: "your-very-secure-k8s-token-here"
  server_addr: "your-controller-public-ip:17000"
  local_ports: [6443]  # Controller 本地的 kube-apiserver 端口
```

**启动客户端：**
```bash
./gotunnel-client
```

**说明：** 这样配置后，gotunnel-server 会在远程端口（默认 10022，但可以通过配置修改）上监听，并将流量转发到 Controller 的 6443 端口。Worker 节点可以通过 `your-controller-public-ip:10022` 访问 kube-apiserver。

**注意：** 当前版本 gotunnel-client 的 `remote_port` 是硬编码为 10022。如果需要使用 6443 端口，需要修改代码或使用反向代理。

#### 3. 配置 K8s Worker 节点

**修改 kubelet 配置：**
```bash
# 编辑 /etc/kubernetes/kubelet.conf
# 将 kube-apiserver 地址改为 gotunnel-server 映射的地址
apiVersion: v1
clusters:
- cluster:
    server: https://your-controller-public-ip:6443  # 使用 gotunnel 映射的地址
    certificate-authority-data: <ca-data>
  name: default-cluster
```

**或者使用 kubeadm join 时指定：**
```bash
kubeadm join your-controller-public-ip:6443 \
  --token <token> \
  --discovery-token-ca-cert-hash sha256:<hash>
```

## 三、完整部署流程

### 步骤 1：准备 Controller 节点

1. **确保 Controller 节点已安装 K8s**
   ```bash
   # 使用 kubeadm 初始化（如果还未初始化）
   kubeadm init --pod-network-cidr=10.244.0.0/16
   ```

2. **部署 gotunnel-server**
   ```bash
   # 下载 gotunnel
   git clone https://github.com/vanyongqi/gotunnel.git
   cd gotunnel
   go build -o gotunnel-server ./cmd/server
   
   # 创建配置文件
   cat > config.yaml <<EOF
   server:
     addr: "0.0.0.0:17000"
     log_level: "info"
     token: "$(openssl rand -hex 32)"  # 生成随机 token
   EOF
   
   # 启动服务（建议使用 systemd）
   sudo cp gotunnel-server /usr/local/bin/
   sudo systemctl enable gotunnel-server
   sudo systemctl start gotunnel-server
   ```

3. **配置防火墙**
   ```bash
   # 开放 gotunnel 控制通道端口
   sudo ufw allow 17000/tcp
   # 开放 kube-apiserver 映射端口（如果需要）
   sudo ufw allow 6443/tcp
   ```

### 步骤 2：准备 Worker 节点（云服务器）

1. **安装 K8s 组件**
   ```bash
   # 安装 kubelet, kubeadm, kubectl
   sudo apt-get update
   sudo apt-get install -y kubelet kubeadm kubectl
   ```

2. **部署 gotunnel-client**
   ```bash
   # 下载 gotunnel
   git clone https://github.com/vanyongqi/gotunnel.git
   cd gotunnel
   go build -o gotunnel-client ./cmd/client
   
   # 创建配置文件
   cat > config.yaml <<EOF
   client:
     name: "$(hostname)"  # 使用主机名
     token: "your-very-secure-k8s-token-here"  # 与 server 一致
     server_addr: "your-controller-public-ip:17000"
     local_ports: [6443]
   EOF
   
   # 启动客户端（建议使用 systemd）
   sudo cp gotunnel-client /usr/local/bin/
   sudo systemctl enable gotunnel-client
   sudo systemctl start gotunnel-client
   ```

3. **验证连接**
   ```bash
   # 检查 gotunnel-client 日志
   sudo journalctl -u gotunnel-client -f
   
   # 应该看到成功注册的消息
   ```

### 步骤 3：加入 K8s 集群

1. **在 Controller 节点获取 join 命令**
   ```bash
   # 获取 token
   kubeadm token create --print-join-command
   ```

2. **在 Worker 节点执行 join**
   ```bash
   # 使用 gotunnel 映射的地址
   kubeadm join your-controller-public-ip:6443 \
     --token <token> \
     --discovery-token-ca-cert-hash sha256:<hash>
   ```

3. **验证节点状态**
   ```bash
   # 在 Controller 节点执行
   kubectl get nodes
   
   # 应该看到 Worker 节点为 Ready 状态
   ```

## 四、端口映射说明

### K8s 组件通信端口

| 组件 | 端口 | 方向 | 说明 |
|------|------|------|------|
| kube-apiserver | 6443 | Controller → Worker | Worker 连接 Controller |
| kubelet API | 10250 | Controller → Worker | Controller 访问 Worker（可选）|
| kube-scheduler | 10259 | Controller 内部 | 不需要映射 |
| kube-controller-manager | 10257 | Controller 内部 | 不需要映射 |
| etcd | 2379-2380 | Controller 内部 | 不需要映射 |

### gotunnel 映射配置

**当前版本限制：**
- 每个客户端只能映射一个端口
- 如果需要多端口，需要运行多个 gotunnel-client 实例

**推荐配置：**
```yaml
# Worker 端：只映射 kube-apiserver 端口
client:
  local_ports: [6443]  # 映射 Controller 的 6443 端口
```

## 五、使用 systemd 管理服务

### Controller 端 systemd 配置

```ini
# /etc/systemd/system/gotunnel-server.service
[Unit]
Description=gotunnel Server
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=/opt/gotunnel
ExecStart=/usr/local/bin/gotunnel-server
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

### Worker 端 systemd 配置

```ini
# /etc/systemd/system/gotunnel-client.service
[Unit]
Description=gotunnel Client
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=/opt/gotunnel
ExecStart=/usr/local/bin/gotunnel-client
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

## 六、安全建议

### 1. Token 安全

```bash
# 生成强随机 token
openssl rand -hex 32

# 使用环境变量或密钥管理服务
export GOTUNNEL_TOKEN="your-token"
```

### 2. 网络安全

- 使用防火墙限制访问来源
- 考虑使用 TLS 加密（未来版本支持）
- 定期更换 token

### 3. K8s 安全

- 使用 RBAC 控制访问权限
- 启用 Pod Security Policies
- 定期更新 K8s 版本

## 七、故障排查

### 1. Worker 节点无法连接 Controller

**检查项：**
- gotunnel-client 是否正常运行
- Controller 公网 IP 和端口是否正确
- 防火墙是否开放 17000 端口
- token 是否匹配

**调试命令：**
```bash
# 检查 gotunnel-client 状态
sudo systemctl status gotunnel-client

# 查看日志
sudo journalctl -u gotunnel-client -f

# 测试网络连接
telnet your-controller-public-ip 17000
```

### 2. K8s 节点状态异常

**检查项：**
- kubelet 是否能连接到 kube-apiserver
- gotunnel 映射是否正常
- 网络延迟是否过高

**调试命令：**
```bash
# 检查 kubelet 状态
sudo systemctl status kubelet

# 查看 kubelet 日志
sudo journalctl -u kubelet -f

# 测试 kube-apiserver 连接
curl -k https://your-controller-public-ip:6443
```

## 八、性能优化

### 1. 网络优化

- 使用低延迟的云服务器
- 考虑使用专线或 VPN
- 优化 gotunnel 心跳间隔

### 2. 资源限制

- 限制 gotunnel 的内存和 CPU 使用
- 监控网络带宽使用情况

## 九、扩展场景

### 1. 多 Worker 节点

每个 Worker 节点都需要：
- 独立的 gotunnel-client 实例
- 唯一的客户端名称
- 相同的 token 和 server_addr

### 2. 高可用 Controller

如果有多个 Controller 节点：
- 每个 Controller 运行 gotunnel-server
- Worker 可以连接到任意 Controller
- 使用负载均衡器分发请求

### 3. 边缘计算场景

- 边缘节点作为 Worker
- Controller 在云端
- 使用 gotunnel 建立连接

## 十、注意事项

1. **当前版本限制**
   - 每个客户端只能映射一个端口
   - 如果需要多端口，需要运行多个实例或修改代码

2. **网络延迟**
   - 公网连接会有延迟
   - 可能影响 K8s 操作响应时间

3. **单点故障**
   - gotunnel-server 是单点
   - 建议使用高可用方案

4. **安全性**
   - 当前版本未加密
   - 生产环境建议使用 VPN 或专线

---

**文档版本：** v1.0  
**最后更新：** 2024-01-XX  
**适用版本：** gotunnel v0.1.0+

