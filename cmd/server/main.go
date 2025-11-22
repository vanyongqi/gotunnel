package main

import (
	"encoding/json"
	"fmt"
	"gotunnel/pkg/core"
	"gotunnel/pkg/ha"
	"gotunnel/pkg/protocol"
	"net"
	"sync"
	"time"

	"github.com/spf13/viper"
)

type Mapping struct {
	ClientConn    net.Conn
	LocalPort     int
	LastHeartbeat time.Time // 记录上次收到心跳的时间
}

var mappingTable = make(map[int]*Mapping)
var mappingTableMu sync.Mutex

var heartbeatTimeout = 30 // 秒

// ServerConfig 保存服务端运行参数
type ServerConfig struct {
	ListenAddr string
	Token      string
}

func loadServerConfig() *ServerConfig {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	_ = viper.ReadInConfig()

	addr := viper.GetString("server.addr")
	if addr == "" {
		addr = ":17000"
	}
	token := viper.GetString("server.token")
	if token == "" {
		token = "changeme"
	}

	return &ServerConfig{
		ListenAddr: addr,
		Token:      token,
	}
}

func main() {
	conf := loadServerConfig()
	ln, err := net.Listen("tcp", conf.ListenAddr)
	if err != nil {
		panic(err)
	}
	fmt.Printf("[gotunnel][server] 控制通道监听: %s (token: %s)\n", conf.ListenAddr, conf.Token)

	// 心跳检查协程由pkg/ha统一调度
	go ha.HeartbeatCheckLoop(checkClientHeartbeat)

	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Println("accept error:", err)
			continue
		}
		go handleControlConn(conn, conf.Token)
	}
}

// checkClientHeartbeat 按mappingTable定期检查，发现超时立即释放资源
func checkClientHeartbeat() {
	mappingTableMu.Lock()
	defer mappingTableMu.Unlock()
	now := time.Now()
	for port, m := range mappingTable {
		if now.Sub(m.LastHeartbeat) > time.Duration(heartbeatTimeout)*time.Second {
			fmt.Printf("[gotunnel][server] 客户端端口 %d 心跳超时，主动关掉映射\n", port)
			m.ClientConn.Close()
			delete(mappingTable, port)
		}
	}
}

// 控制通道，注册、心跳包等
func handleControlConn(conn net.Conn, serverToken string) {
	defer conn.Close()
	var regdRemotePort, regdLocalPort int
	var listenDone chan struct{} = nil
	// 读取注册消息
	firstPacket, err := protocol.ReadPacket(conn)
	if err != nil {
		fmt.Println("[server] 读取注册包失败:", err)
		return
	}
	var reg protocol.RegisterRequest
	_ = json.Unmarshal(firstPacket, &reg)
	if reg.Token != serverToken {
		resp := protocol.RegisterResponse{Type: "register_resp", Status: "fail", Reason: "鉴权失败"}
		msg, _ := json.Marshal(resp)
		protocol.WritePacket(conn, msg)
		fmt.Println("[server] token校验失败，已拒绝", reg.Name)
		return
	}
	mappingTableMu.Lock()
	mappingTable[reg.RemotePort] = &Mapping{ClientConn: conn, LocalPort: reg.LocalPort, LastHeartbeat: time.Now()}
	mappingTableMu.Unlock()
	regdRemotePort, regdLocalPort = reg.RemotePort, reg.LocalPort
	fmt.Printf("[gotunnel][server] 注册成功，公网端口 %d => 内网 %d\n", regdRemotePort, regdLocalPort)
	resp := protocol.RegisterResponse{Type: "register_resp", Status: "ok"}
	msg, _ := json.Marshal(resp)
	protocol.WritePacket(conn, msg)

	listenDone = make(chan struct{})
	go listenAndForwardWithStop(regdRemotePort, conn, regdLocalPort, listenDone)

	for {
		packet, err := protocol.ReadPacket(conn)
		if err != nil {
			fmt.Printf("[server] 控制通道断开: %v\n", err)
			break
		}

		var ping protocol.HeartbeatPing
		if err := json.Unmarshal(packet, &ping); err == nil && ping.Type == "ping" {
			mappingTableMu.Lock()
			if m, ok := mappingTable[regdRemotePort]; ok {
				m.LastHeartbeat = time.Now()
			}
			mappingTableMu.Unlock()
			pong := protocol.HeartbeatPong{Type: "pong", Time: time.Now().Unix()}
			b, _ := json.Marshal(pong)
			protocol.WritePacket(conn, b)
			continue
		}
		// offline_port 处理
		var off protocol.OfflinePortRequest
		if err := json.Unmarshal(packet, &off); err == nil && off.Type == "offline_port" {
			fmt.Printf("[gotunnel][server] client请求下线端口 %d\n", off.Port)
			// 主动结束监听、relay
			close(listenDone)
			mappingTableMu.Lock()
			delete(mappingTable, off.Port)
			mappingTableMu.Unlock()
			continue
		}
		var on protocol.OnlinePortRequest
		if err := json.Unmarshal(packet, &on); err == nil && on.Type == "online_port" {
			fmt.Printf("[gotunnel][server] client恢复端口 %d，重新注册port监听\n", on.Port)
			// 重新监听端口
			listenDone = make(chan struct{})
			go listenAndForwardWithStop(on.Port, conn, regdLocalPort, listenDone)
			continue
		}
		// open_data_channel ... 其它协议不变
		var ctrl protocol.RegisterRequest
		if err := json.Unmarshal(packet, &ctrl); err == nil && ctrl.Type == "open_data_channel" {
			continue
		}
	}
	mappingTableMu.Lock()
	delete(mappingTable, regdRemotePort)
	mappingTableMu.Unlock()
	fmt.Println("[server] 控制通道退出，清理端口映射")
}

// 新增支持stop信号的监听，供健康探针down时停止端口监听和relay
func listenAndForwardWithStop(remotePort int, clientConn net.Conn, localPort int, stop <-chan struct{}) {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", remotePort))
	if err != nil {
		fmt.Printf("[server] 监听端口失败: %v\n", err)
		return
	}
	fmt.Printf("[gotunnel][server] 公网端口监听开启: %d\n", remotePort)
	acceptCh := make(chan net.Conn)
	go func() {
		for {
			userConn, err := ln.Accept()
			if err != nil {
				continue
			}
			acceptCh <- userConn
		}
	}()
	for {
		select {
		case <-stop:
			ln.Close()
			fmt.Printf("[server] 停止公网端口监听:%d(健康探针下线)\n", remotePort)
			return
		case userConn := <-acceptCh:
			go func() {
				req := protocol.RegisterRequest{Type: "open_data_channel", LocalPort: localPort}
				reqBytes, _ := json.Marshal(req)
				protocol.WritePacket(clientConn, reqBytes)
				core.RelayConn(userConn, clientConn)
			}()
		}
	}
}
