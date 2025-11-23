# gotunnel 演示示例

本目录包含 gotunnel 的演示场景配置和说明。

## 📁 目录结构

```
example/
├── README.md           # 本文件
├── http/              # HTTP 服务演示场景
│   ├── README.md      # HTTP 演示说明
│   ├── server.yaml    # 服务端配置
│   └── client.yaml    # 客户端配置
└── ssh/               # SSH 服务演示场景
    ├── README.md      # SSH 演示说明
    ├── server.yaml    # 服务端配置
    └── client.yaml    # 客户端配置
```

## 🚀 快速开始

### 场景 1：HTTP 服务演示

将本地 HTTP 服务通过 gotunnel 暴露到公网。

```bash
cd example/http
# 查看 README.md 了解详细步骤
```

### 场景 2：SSH 服务演示

通过 gotunnel 远程访问内网 SSH 服务。

```bash
cd example/ssh
# 查看 README.md 了解详细步骤
```

## 📝 使用说明

1. 选择要演示的场景（HTTP 或 SSH）
2. 复制对应的配置文件到项目根目录作为 `config.yaml`
3. 根据实际情况修改配置（如服务器 IP、端口等）
4. 按照场景 README 中的步骤操作

## ⚠️ 注意事项

- 确保服务端防火墙开放相应端口
- 客户端和服务端的 token 必须一致
- 服务端地址应使用公网 IP（或可访问的 IP）

