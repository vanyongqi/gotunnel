# gotunnel for Kubernetes Cross-Network Deployment

**Language:** [English](./08-K8S-DEPLOYMENT.md) | [中文](../zh/08-K8S-DEPLOYMENT.md)

This document describes how to use gotunnel to implement Kubernetes hybrid cloud deployment scenarios, where a local intranet K8s control plane manages remote cloud server worker nodes.

## I. Scenario Description

### Architecture Topology

```
┌─────────────────────────────────────────────────────────┐
│  Local Intranet (Controller Node)                       │
│                                                          │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐ │
│  │ kube-apiserver│  │ etcd         │  │ kube-controller││
│  │ :6443        │  │ :2379-2380   │  │ :10257        │ │
│  └──────────────┘  └──────────────┘  └──────────────┘ │
│                                                          │
│  ┌──────────────────────────────────────────────────┐   │
│  │ gotunnel-server (listening on public port)      │   │
│  │ :17000 (control channel)                        │   │
│  │ :6443 (mapped to worker's kube-apiserver)       │   │
│  └──────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────┘
                        │
                        │ Public Network
                        │
┌─────────────────────────────────────────────────────────┐
│  Cloud Server (Worker Node)                            │
│                                                          │
│  ┌──────────────────────────────────────────────────┐   │
│  │ gotunnel-client                                  │   │
│  │ Mapping: local:6443 -> remote:6443               │   │
│  └──────────────────────────────────────────────────┘   │
│                                                          │
│  ┌──────────────┐                                       │
│  │ kubelet       │  ──> Connect to kube-apiserver:6443 │
│  │ kube-proxy    │  ──> Connect to kube-apiserver:6443 │
│  │ CNI plugins   │  ──> Connect to kube-apiserver:6443 │
│  └──────────────┘                                       │
└─────────────────────────────────────────────────────────┘
```

### Core Requirements

1. **Worker nodes need to connect to Controller's kube-apiserver**
   - Worker node components (kubelet, kube-proxy, CNI) all need to access kube-apiserver
   - Default port: 6443 (HTTPS)

2. **Controller needs to access Worker node's kubelet API**
   - For health checks, log collection, exec operations
   - Default port: 10250 (HTTPS)

3. **Network Isolation Problem**
   - Controller is in intranet, Worker is on public network
   - They cannot communicate directly
   - Need to establish tunnel through gotunnel

## II. Deployment Solutions

### Solution 1: Unidirectional Connection (Recommended for Testing)

**Scenario:** Worker nodes actively connect to Controller

**Configuration Steps:**

#### 1. Controller Side (Local Intranet)

**Start gotunnel-server:**

```yaml
# config.yaml (Controller side)
server:
  addr: "0.0.0.0:17000"  # Listen on all interfaces, allow public access
  log_level: "info"
  token: "your-very-secure-k8s-token-here"  # Use strong random token
```

**Start service:**
```bash
./gotunnel-server
```

#### 2. Worker Side (Cloud Server)

**Install gotunnel-client:**
```bash
# Download and build
git clone https://github.com/vanyongqi/gotunnel.git
cd gotunnel
go build -o gotunnel-client ./cmd/client
```

**Configure gotunnel-client:**
```yaml
# config.yaml (Worker side)
client:
  name: "k8s-worker-01"  # Node name, recommend using hostname
  token: "your-very-secure-k8s-token-here"  # Must match server
  server_addr: "your-controller-public-ip:17000"  # Controller's public IP
  local_ports: [6443]  # Map kube-apiserver port
```

**Note:** The current version has limitations. For production use, see Solution 2.

### Solution 2: Bidirectional Connection (Recommended for Production)

**Scenario:** Both Controller and Worker need to access each other

#### Architecture Design

```
Controller (Intranet)                    Worker (Public Network)
     │                                    │
     │  ┌─────────────────────────────┐   │
     │  │ gotunnel-server            │   │
     │  │ :17000 (control channel)   │   │
     │  │ :6443 (mapped to worker)   │   │
     │  └─────────────────────────────┘   │
     │           ▲                         │
     │           │                         │
     │           │ Control Channel         │
     │           │                         │
     │           │                         │
     │  ┌─────────────────────────────┐   │
     │  │ gotunnel-client            │   │
     │  │ Mapping: local:6443 -> remote:6443│
     │  └─────────────────────────────┘   │
     │                                    │
     │  kube-apiserver :6443              │  kubelet :10250
     │                                    │
```

#### 1. Controller Side Configuration

**Start gotunnel-server:**
```yaml
# config.yaml (Controller side)
server:
  addr: "0.0.0.0:17000"
  log_level: "info"
  token: "your-very-secure-k8s-token-here"
```

**Start service:**
```bash
./gotunnel-server
```

#### 2. Worker Side Configuration

**Configure gotunnel-client:**
```yaml
# config.yaml (Worker side)
client:
  name: "k8s-worker-01"  # Use hostname or node identifier
  token: "your-very-secure-k8s-token-here"
  server_addr: "your-controller-public-ip:17000"
  local_ports: [6443]  # Map Controller's 6443 to address accessible by Worker
```

**Start client:**
```bash
./gotunnel-client
```

#### 3. Configure K8s Worker Node

**Modify kubelet configuration:**
```bash
# Edit /etc/kubernetes/kubelet.conf
# Change kube-apiserver address to gotunnel-server mapped address
apiVersion: v1
clusters:
- cluster:
    server: https://your-controller-public-ip:6443  # Use gotunnel mapped address
    certificate-authority-data: <ca-data>
  name: default-cluster
```

**Or specify when using kubeadm join:**
```bash
kubeadm join your-controller-public-ip:6443 \
  --token <token> \
  --discovery-token-ca-cert-hash sha256:<hash>
```

## III. Complete Deployment Process

### Step 1: Prepare Controller Node

1. **Ensure Controller node has K8s installed**
   ```bash
   # Initialize with kubeadm (if not already initialized)
   kubeadm init --pod-network-cidr=10.244.0.0/16
   ```

2. **Deploy gotunnel-server**
   ```bash
   # Download gotunnel
   git clone https://github.com/vanyongqi/gotunnel.git
   cd gotunnel
   go build -o gotunnel-server ./cmd/server
   
   # Create config file
   cat > config.yaml <<EOF
   server:
     addr: "0.0.0.0:17000"
     log_level: "info"
     token: "$(openssl rand -hex 32)"  # Generate random token
   EOF
   
   # Start service (recommend using systemd)
   sudo cp gotunnel-server /usr/local/bin/
   sudo systemctl enable gotunnel-server
   sudo systemctl start gotunnel-server
   ```

3. **Configure firewall**
   ```bash
   # Open gotunnel control channel port
   sudo ufw allow 17000/tcp
   # Open kube-apiserver mapped port (if needed)
   sudo ufw allow 6443/tcp
   ```

### Step 2: Prepare Worker Node (Cloud Server)

1. **Install K8s components**
   ```bash
   # Install kubelet, kubeadm, kubectl
   sudo apt-get update
   sudo apt-get install -y kubelet kubeadm kubectl
   ```

2. **Deploy gotunnel-client**
   ```bash
   # Download gotunnel
   git clone https://github.com/vanyongqi/gotunnel.git
   cd gotunnel
   go build -o gotunnel-client ./cmd/client
   
   # Create config file
   cat > config.yaml <<EOF
   client:
     name: "$(hostname)"  # Use hostname
     token: "your-very-secure-k8s-token-here"  # Must match server
     server_addr: "your-controller-public-ip:17000"
     local_ports: [6443]
   EOF
   
   # Start client (recommend using systemd)
   sudo cp gotunnel-client /usr/local/bin/
   sudo systemctl enable gotunnel-client
   sudo systemctl start gotunnel-client
   ```

3. **Verify connection**
   ```bash
   # Check gotunnel-client logs
   sudo journalctl -u gotunnel-client -f
   
   # Should see successful registration message
   ```

### Step 3: Join K8s Cluster

1. **Get join command on Controller node**
   ```bash
   # Get token
   kubeadm token create --print-join-command
   ```

2. **Execute join on Worker node**
   ```bash
   # Use gotunnel mapped address
   kubeadm join your-controller-public-ip:6443 \
     --token <token> \
     --discovery-token-ca-cert-hash sha256:<hash>
   ```

3. **Verify node status**
   ```bash
   # Execute on Controller node
   kubectl get nodes
   
   # Should see Worker node in Ready status
   ```

## IV. Port Mapping Description

### K8s Component Communication Ports

| Component | Port | Direction | Description |
|-----------|------|-----------|-------------|
| kube-apiserver | 6443 | Controller → Worker | Worker connects to Controller |
| kubelet API | 10250 | Controller → Worker | Controller accesses Worker (optional) |
| kube-scheduler | 10259 | Controller internal | No mapping needed |
| kube-controller-manager | 10257 | Controller internal | No mapping needed |
| etcd | 2379-2380 | Controller internal | No mapping needed |

### gotunnel Mapping Configuration

**Current Version Limitations:**
- Each client can only map one port
- If multiple ports are needed, need to run multiple gotunnel-client instances

**Recommended Configuration:**
```yaml
# Worker side: Only map kube-apiserver port
client:
  local_ports: [6443]  # Map Controller's 6443 port
```

## V. Using systemd to Manage Services

### Controller Side systemd Configuration

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

### Worker Side systemd Configuration

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

## VI. Security Recommendations

### 1. Token Security

```bash
# Generate strong random token
openssl rand -hex 32

# Use environment variables or key management service
export GOTUNNEL_TOKEN="your-token"
```

### 2. Network Security

- Use firewall to limit access sources
- Consider using TLS encryption (future version support)
- Regularly change tokens

### 3. K8s Security

- Use RBAC to control access permissions
- Enable Pod Security Policies
- Regularly update K8s version

## VII. Troubleshooting

### 1. Worker Node Cannot Connect to Controller

**Check Items:**
- Is gotunnel-client running normally?
- Is Controller public IP and port correct?
- Is firewall open for port 17000?
- Do tokens match?

**Debug Commands:**
```bash
# Check gotunnel-client status
sudo systemctl status gotunnel-client

# View logs
sudo journalctl -u gotunnel-client -f

# Test network connection
telnet your-controller-public-ip 17000
```

### 2. K8s Node Status Abnormal

**Check Items:**
- Can kubelet connect to kube-apiserver?
- Is gotunnel mapping normal?
- Is network latency too high?

**Debug Commands:**
```bash
# Check kubelet status
sudo systemctl status kubelet

# View kubelet logs
sudo journalctl -u kubelet -f

# Test kube-apiserver connection
curl -k https://your-controller-public-ip:6443
```

## VIII. Performance Optimization

### 1. Network Optimization

- Use low-latency cloud servers
- Consider using dedicated lines or VPN
- Optimize gotunnel heartbeat interval

### 2. Resource Limits

- Limit gotunnel memory and CPU usage
- Monitor network bandwidth usage

## IX. Extended Scenarios

### 1. Multiple Worker Nodes

Each Worker node needs:
- Independent gotunnel-client instance
- Unique client name
- Same token and server_addr

### 2. High Availability Controller

If there are multiple Controller nodes:
- Each Controller runs gotunnel-server
- Workers can connect to any Controller
- Use load balancer to distribute requests

### 3. Edge Computing Scenarios

- Edge nodes as Workers
- Controller in cloud
- Use gotunnel to establish connection

## X. Important Notes

1. **Current Version Limitations**
   - Each client can only map one port
   - If multiple ports are needed, need to run multiple instances or modify code

2. **Network Latency**
   - Public network connections will have latency
   - May affect K8s operation response time

3. **Single Point of Failure**
   - gotunnel-server is a single point
   - Recommend using high availability solution

4. **Security**
   - Current version is not encrypted
   - Production environment recommend using VPN or dedicated line

---

**Document Version:** v1.0  
**Last Updated:** 2024-01-XX  
**Applicable Version:** gotunnel v0.1.0+

