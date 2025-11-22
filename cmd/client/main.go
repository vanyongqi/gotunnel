package main

import (
	"encoding/json"
	"fmt"
	"gotunnel/pkg/core"
	"gotunnel/pkg/errors"
	"gotunnel/pkg/ha"
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

// ClientConfig 保存客户端配置
type ClientConfig struct {
	Name       string
	Token      string
	ServerAddr string
	LocalPort  int
	RemotePort int
}

// loadClientConfig 加载客户端配置
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

func main() {
	conf := loadClientConfig()
	fmt.Printf("[gotunnel][client] 注册端口: 本地 %d => 公网 %d\n", conf.LocalPort, conf.RemotePort)
	for {
		var healthDown = false
		var conn net.Conn = nil
		dial := func() bool {
			c, err := net.Dial("tcp", conf.ServerAddr)
			if err != nil {
				errors.PrintError(errors.ErrConnectFailed, err)
				return false
			}
			conn = c
			return true
		}
		if !ha.ReconnectLoop(dial, 3, 60, 0) {
			fmt.Println("[gotunnel][client] 自动重连中止")
			return
		}

		// 注册端口逻辑
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
			conn.Close()
			time.Sleep(3 * time.Second)
			continue
		}
		respBytes, err := protocol.ReadPacket(conn)
		if err != nil {
			conn.Close()
			time.Sleep(3 * time.Second)
			continue
		}
		var resp protocol.RegisterResponse
		_ = json.Unmarshal(respBytes, &resp)
		if resp.Status != "ok" {
			conn.Close()
			time.Sleep(5 * time.Second)
			continue
		}
		fmt.Println("[gotunnel][client] 端口注册成功，启动心跳和健康探针...")

		// 启动心跳包 goroutine
		hbMgr := &ha.HeartbeatManager{
			Conn:     conn,
			Interval: time.Duration(heartbeatInterval) * time.Second,
			OnTimeout: func() {
				fmt.Println("[gotunnel][client] 心跳超时，触发重连...")
				conn.Close()
			},
		}
		hbMgr.StartHeartbeat()
		defer hbMgr.StopHeartbeat()

		// 启动健康探针（副协程）
		doneHealth := make(chan struct{})
		go func() {
			target := fmt.Sprintf("127.0.0.1:%d", conf.LocalPort)
			health.PeriodicProbe(target, healthCheckInterval,
				// onDead: 探针down时通知server下线端口
				func() {
					if !healthDown {
						fmt.Printf("[gotunnel][client] 本地端口%d健康丢失，发起offline_port\n", conf.LocalPort)
						req := protocol.OfflinePortRequest{Type: "offline_port", Port: conf.RemotePort}
						b, _ := json.Marshal(req)
						protocol.WritePacket(conn, b)
						healthDown = true
					}
				},
				// onAlive: 恢复时重注册online_port
				func() {
					if healthDown {
						fmt.Printf("[gotunnel][client] 本地端口%d恢复，发起online_port\n", conf.LocalPort)
						req := protocol.OnlinePortRequest{Type: "online_port", Port: conf.RemotePort}
						b, _ := json.Marshal(req)
						protocol.WritePacket(conn, b)
						healthDown = false
					}
				},
			)
			close(doneHealth)
		}()

		for {
			packet, err := protocol.ReadPacket(conn)
			if err != nil {
				fmt.Println("[gotunnel][client] 控制通道断开，自动重连: ", err)
				break // 跳出自动重连
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
		conn.Close()
		time.Sleep(3 * time.Second)
	}
}
