package main

import (
	"encoding/json"
	"fmt"
	"gotunnel/pkg/core"
	"gotunnel/pkg/ha"
	"gotunnel/pkg/log"
	"gotunnel/pkg/protocol"
	"net"
	"sync"
	"time"

	"github.com/spf13/viper"
)

// Mapping represents a port mapping between a remote port and a local port.
type Mapping struct {
	ClientConn    net.Conn
	LocalPort     int
	LastHeartbeat time.Time // Last heartbeat time received
}

var mappingTable = make(map[int]*Mapping)
var mappingTableMu sync.Mutex

var heartbeatTimeout = 30 // seconds

// ServerConfig holds the server configuration parameters.
type ServerConfig struct {
	ListenAddr string
	Token      string
	LogLevel   string
	LogLang    string
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
	logLevel := viper.GetString("server.log_level")
	if logLevel == "" {
		logLevel = "info"
	}
	logLang := viper.GetString("server.log_lang")
	if logLang == "" {
		logLang = "zh"
	}

	return &ServerConfig{
		ListenAddr: addr,
		Token:      token,
		LogLevel:   logLevel,
		LogLang:    logLang,
	}
}

func main() {
	conf := loadServerConfig()

	// Initialize logger
	log.Init(log.ParseLevel(conf.LogLevel), log.ParseLanguage(conf.LogLang))

	ln, err := net.Listen("tcp", conf.ListenAddr)
	if err != nil {
		panic(err)
	}
	log.Infof("server", "server.control_channel_listening", conf.ListenAddr, conf.Token)

	// Heartbeat check goroutine is scheduled by pkg/ha
	go ha.HeartbeatCheckLoop(checkClientHeartbeat)

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Errorf("server", "server.accept_error", err)
			continue
		}
		go handleControlConn(conn, conf.Token)
	}
}

// checkClientHeartbeat periodically checks mappingTable and releases resources immediately on timeout
func checkClientHeartbeat() {
	mappingTableMu.Lock()
	defer mappingTableMu.Unlock()
	now := time.Now()
	for port, m := range mappingTable {
		if now.Sub(m.LastHeartbeat) > time.Duration(heartbeatTimeout)*time.Second {
			log.Warnf("server", "server.client_heartbeat_timeout", port)
			_ = m.ClientConn.Close()
			delete(mappingTable, port)
		}
	}
}

// handleControlConn handles the control channel for registration, heartbeat, etc.
func handleControlConn(conn net.Conn, serverToken string) {
	defer func() { _ = conn.Close() }()
	var regdRemotePort, regdLocalPort int
	var listenDone chan struct{}
	// Read registration message
	firstPacket, err := protocol.ReadPacket(conn)
	if err != nil {
		log.Errorf("server", "server.read_register_packet_failed", err)
		return
	}
	var reg protocol.RegisterRequest
	_ = json.Unmarshal(firstPacket, &reg)
	if reg.Token != serverToken {
		resp := protocol.RegisterResponse{Type: "register_resp", Status: "fail", Reason: "authentication failed"}
		msg, _ := json.Marshal(resp)
		if err := protocol.WritePacket(conn, msg); err != nil {
			log.Errorf("server", "server.send_response_failed", err)
		}
		log.Warnf("server", "server.token_auth_failed", reg.Name)
		return
	}
	mappingTableMu.Lock()
	mappingTable[reg.RemotePort] = &Mapping{ClientConn: conn, LocalPort: reg.LocalPort, LastHeartbeat: time.Now()}
	mappingTableMu.Unlock()
	regdRemotePort, regdLocalPort = reg.RemotePort, reg.LocalPort
	log.Infof("server", "server.port_mapping_registered", regdRemotePort, regdLocalPort)
	resp := protocol.RegisterResponse{Type: "register_resp", Status: "ok"}
	msg, _ := json.Marshal(resp)
	if err := protocol.WritePacket(conn, msg); err != nil {
		log.Errorf("server", "server.send_response_failed", err)
		return
	}

	listenDone = make(chan struct{})
	go listenAndForwardWithStop(regdRemotePort, conn, regdLocalPort, listenDone)

	for {
		packet, err := protocol.ReadPacket(conn)
		if err != nil {
			log.Warnf("server", "server.control_channel_disconnected", err)
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
			if err := protocol.WritePacket(conn, b); err != nil {
				log.Errorf("server", "server.send_heartbeat_failed", err)
				break
			}
			continue
		}
		// Handle offline_port request
		var off protocol.OfflinePortRequest
		if err := json.Unmarshal(packet, &off); err == nil && off.Type == "offline_port" {
			log.Infof("server", "server.client_offline_port", off.Port)
			// Actively stop listening and relay
			close(listenDone)
			mappingTableMu.Lock()
			delete(mappingTable, off.Port)
			mappingTableMu.Unlock()
			continue
		}
		var on protocol.OnlinePortRequest
		if err := json.Unmarshal(packet, &on); err == nil && on.Type == "online_port" {
			log.Infof("server", "server.client_online_port", on.Port)
			// Re-listen on the port
			listenDone = make(chan struct{})
			go listenAndForwardWithStop(on.Port, conn, regdLocalPort, listenDone)
			continue
		}
		// Handle open_data_channel and other protocols
		var ctrl protocol.RegisterRequest
		if err := json.Unmarshal(packet, &ctrl); err == nil && ctrl.Type == "open_data_channel" {
			continue
		}
	}
	mappingTableMu.Lock()
	delete(mappingTable, regdRemotePort)
	mappingTableMu.Unlock()
	log.Info("server", "server.control_channel_exit", nil)
}

// listenAndForwardWithStop listens with stop signal support, allowing health probe to stop port listening and relay when down
func listenAndForwardWithStop(remotePort int, clientConn net.Conn, localPort int, stop <-chan struct{}) {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", remotePort))
	if err != nil {
		log.Errorf("server", "server.listen_port_failed", err)
		return
	}
	log.Infof("server", "server.port_listening", remotePort)
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
			_ = ln.Close()
			log.Infof("server", "server.port_stopped", remotePort)
			return
		case userConn := <-acceptCh:
			go func() {
				req := protocol.RegisterRequest{Type: "open_data_channel", LocalPort: localPort}
				reqBytes, _ := json.Marshal(req)
				if err := protocol.WritePacket(clientConn, reqBytes); err != nil {
					log.Errorf("server", "server.send_data_channel_cmd_failed", err)
					_ = userConn.Close()
					return
				}
				core.RelayConn(userConn, clientConn)
			}()
		}
	}
}
