package main

import (
	"encoding/json"
	"fmt"
	"gotunnel/pkg/core"
	"gotunnel/pkg/errors"
	"gotunnel/pkg/health"
	"gotunnel/pkg/protocol"
	"net"
	"time"

	"github.com/spf13/viper"
)

var (
	heartbeatInterval   = 10               // 秒,建议配置化
	healthCheckInterval = 30 * time.Second // probe 间隔
)

// ClientConfig holds the client configuration parameters.
type ClientConfig struct {
	Name       string
	Token      string
	ServerAddr string
	LocalPort  int
	RemotePort int
}

func loadClientConfig() *ClientConfig {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	_ = viper.ReadInConfig()

	name := viper.GetString("client.name")
	if name == "" {
		name = "gotunnel-client-demo"
	}
	token := viper.GetString("client.token")
	if token == "" {
		token = "changeme"
	}
	serverAddr := viper.GetString("client.server_addr")
	if serverAddr == "" {
		serverAddr = "127.0.0.1:17000"
	}
	localPort := 22
	if viper.IsSet("client.local_ports") {
		arr := viper.Get("client.local_ports").([]interface{})
		if len(arr) > 0 {
			lp, ok := arr[0].(int)
			if ok {
				localPort = lp
			}
		}
	}
	remotePort := 10022 // 可以通过配置扩展
	return &ClientConfig{
		Name:       name,
		Token:      token,
		ServerAddr: serverAddr,
		LocalPort:  localPort,
		RemotePort: remotePort,
	}
}

// DialServer establishes a TCP connection to the server.
func DialServer(conf *ClientConfig) (net.Conn, error) {
	return net.Dial("tcp", conf.ServerAddr)
}

// RegisterPort sends a port registration request to the server.
func RegisterPort(conn net.Conn, conf *ClientConfig) error {
	registerReq := protocol.RegisterRequest{
		Type:       "register",
		LocalPort:  conf.LocalPort,
		RemotePort: conf.RemotePort,
		Protocol:   "tcp",
		Token:      conf.Token,
		Name:       conf.Name,
	}
	reqBytes, _ := json.Marshal(registerReq)
	if err := protocol.WritePacket(conn, reqBytes); err != nil {
		return err
	}
	respBytes, err := protocol.ReadPacket(conn)
	if err != nil {
		return err
	}
	var resp protocol.RegisterResponse
	_ = json.Unmarshal(respBytes, &resp)
	if resp.Status != "ok" {
		return fmt.Errorf("register failed: %s", resp.Reason)
	}
	return nil
}

// HeartbeatManager manages heartbeat sending and monitoring for control channel health.
type HeartbeatManager struct {
	Conn      net.Conn
	Interval  time.Duration
	OnTimeout func()
	stop      chan struct{}
}

// StartHeartbeat starts the heartbeat goroutine that periodically sends ping packets.
func (h *HeartbeatManager) StartHeartbeat() {
	h.stop = make(chan struct{})
	go func() {
		for {
			select {
			case <-h.stop:
				return
			case <-time.After(h.Interval):
				ping := protocol.HeartbeatPing{Type: "ping", Time: time.Now().Unix()}
				b, _ := json.Marshal(ping)
				_ = protocol.WritePacket(h.Conn, b)
			}
		}
	}()
}

// StopHeartbeat stops the heartbeat goroutine.
func (h *HeartbeatManager) StopHeartbeat() { close(h.stop) }

// StartHeartbeat creates and starts a heartbeat manager for the given connection.
func StartHeartbeat(conn net.Conn, interval time.Duration, onTimeout func()) (stop func()) {
	mgr := &HeartbeatManager{Conn: conn, Interval: interval, OnTimeout: onTimeout}
	mgr.StartHeartbeat()
	return mgr.StopHeartbeat
}

// StartHealthProbe starts a periodic health probe for the local port.
func StartHealthProbe(conf *ClientConfig, conn net.Conn, onOffline func(), onOnline func()) (stop func()) {
	doneHealth := make(chan struct{})
	go func() {
		target := fmt.Sprintf("127.0.0.1:%d", conf.LocalPort)
		health.PeriodicProbe(target, healthCheckInterval, onOffline, onOnline)
		close(doneHealth)
	}()
	return func() { close(doneHealth) }
}

// StartControlLoop starts the main control loop that handles server messages.
func StartControlLoop(conn net.Conn, conf *ClientConfig) error {
	for {
		packet, err := protocol.ReadPacket(conn)
		if err != nil {
			return err
		}
		var ping protocol.HeartbeatPong
		if err := json.Unmarshal(packet, &ping); err == nil && ping.Type == "pong" {
			continue
		}
		var ctrl protocol.RegisterRequest
		_ = json.Unmarshal(packet, &ctrl)
		if ctrl.Type == "open_data_channel" {
			fmt.Printf("[gotunnel][client] 收到数据通道指令，准备转发本地 %d\n", ctrl.LocalPort)
			localConn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", ctrl.LocalPort))
			if err != nil {
				fmt.Println("[ERROR] 连接本地端口失败:", err)
				continue
			}
			fmt.Println("[gotunnel][client] relay 开始 ...")
			core.RelayConn(conn, localConn)
		}
	}
}

// handleConnection handles a single connection lifecycle.
func handleConnection(conn net.Conn, conf *ClientConfig) error {
	// 启动心跳包 goroutine
	heartbeatStop := StartHeartbeat(conn, time.Duration(heartbeatInterval)*time.Second, func() {
		fmt.Println("[gotunnel][client] 心跳超时，触发重连...")
		_ = conn.Close()
	})
	defer heartbeatStop()

	// 启动健康探针（使用闭包捕获状态变量）
	var healthDown bool
	stopHealth := StartHealthProbe(conf, conn,
		func() {
			if !healthDown {
				fmt.Printf("[gotunnel][client] 本地端口%d健康丢失，发起offline_port\n", conf.LocalPort)
				req := protocol.OfflinePortRequest{Type: "offline_port", Port: conf.RemotePort}
				b, _ := json.Marshal(req)
				if err := protocol.WritePacket(conn, b); err != nil {
					fmt.Printf("[gotunnel][client] 发送offline_port失败: %v\n", err)
				}
				healthDown = true
			}
		},
		func() {
			if healthDown {
				fmt.Printf("[gotunnel][client] 本地端口%d恢复，发起online_port\n", conf.LocalPort)
				req := protocol.OnlinePortRequest{Type: "online_port", Port: conf.RemotePort}
				b, _ := json.Marshal(req)
				if err := protocol.WritePacket(conn, b); err != nil {
					fmt.Printf("[gotunnel][client] 发送online_port失败: %v\n", err)
				}
				healthDown = false
			}
		},
	)
	defer stopHealth()

	return StartControlLoop(conn, conf)
}

func main() {
	conf := loadClientConfig()
	fmt.Printf("[gotunnel][client] 注册端口: 本地 %d => 公网 %d\n", conf.LocalPort, conf.RemotePort)
	for {
		conn, err := DialServer(conf)
		if err != nil {
			errors.PrintError(errors.ErrConnectFailed, err)
			time.Sleep(3 * time.Second)
			continue
		}
		if err := RegisterPort(conn, conf); err != nil {
			_ = conn.Close()
			fmt.Println("[gotunnel][client] 端口注册失败:", err)
			time.Sleep(3 * time.Second)
			continue
		}
		fmt.Println("[gotunnel][client] 端口注册成功，启动心跳和健康探针...")

		if err := handleConnection(conn, conf); err != nil {
			fmt.Println("[gotunnel][client] 控制通道断开，自动重连:", err)
			_ = conn.Close()
			time.Sleep(3 * time.Second)
			continue
		}
		_ = conn.Close()
		time.Sleep(3 * time.Second)
	}
}
