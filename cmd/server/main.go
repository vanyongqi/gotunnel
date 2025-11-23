package main

import (
	"context"
	"encoding/json"
	"fmt"
	"gotunnel/pkg/core"
	"gotunnel/pkg/log"
	"gotunnel/pkg/protocol"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/spf13/viper"
)

// Mapping represents a port mapping between a remote port and a local port.
type Mapping struct {
	ClientConn    net.Conn
	LocalPort     int
	LastHeartbeat time.Time     // Last heartbeat time received
	DataChan      chan net.Conn // Channel for pending data channel connections
	ListenDone    chan struct{} // Channel to stop listening
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

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Heartbeat check goroutine is scheduled by pkg/ha
	heartbeatDone := make(chan struct{})
	go func() {
		defer close(heartbeatDone)
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				checkClientHeartbeat()
			}
		}
	}()

	// Accept connections in a goroutine
	acceptDone := make(chan struct{})
	go func() {
		defer close(acceptDone)
		for {
			select {
			case <-ctx.Done():
				return
			default:
				conn, err := ln.Accept()
				if err != nil {
					// Check if it's due to shutdown
					select {
					case <-ctx.Done():
						return
					default:
						log.Errorf("server", "server.accept_error", err)
						continue
					}
				}
				go handleControlConn(conn, conf.Token)
			}
		}
	}()

	// Wait for shutdown signal
	<-sigChan
	log.Info("server", "server.shutdown_signal_received", nil)

	// Start graceful shutdown
	log.Info("server", "server.shutdown_started", nil)
	cancel()

	// Close listener to stop accepting new connections
	_ = ln.Close()

	// Wait for accept goroutine to finish
	<-acceptDone

	// Close all active connections
	mappingTableMu.Lock()
	for port, mapping := range mappingTable {
		log.Infof("server", "server.closing_mapping", port)
		// Close listener
		if mapping.ListenDone != nil {
			select {
			case <-mapping.ListenDone:
			default:
				close(mapping.ListenDone)
			}
		}
		// Close control connection
		_ = mapping.ClientConn.Close()
		// Close data channel queue
		select {
		case <-mapping.DataChan:
		default:
			close(mapping.DataChan)
		}
	}
	mappingTable = make(map[int]*Mapping)
	mappingTableMu.Unlock()

	// Wait a bit for connections to close gracefully
	time.Sleep(2 * time.Second)

	log.Info("server", "server.shutdown_complete", nil)
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
	var regdRemotePort, regdLocalPort int
	var listenDone chan struct{}
	// Read registration message
	firstPacket, err := protocol.ReadPacket(conn)
	if err != nil {
		log.Errorf("server", "server.read_register_packet_failed", err)
		_ = conn.Close()
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
		_ = conn.Close()
		return
	}
	// Handle data channel connection separately
	if reg.Type == "data_channel" {
		resp := protocol.RegisterResponse{Type: "register_resp", Status: "ok"}
		msg, _ := json.Marshal(resp)
		if err := protocol.WritePacket(conn, msg); err != nil {
			log.Errorf("server", "server.send_response_failed", err)
			_ = conn.Close()
			return
		}
		// Find the corresponding mapping and send data connection to channel
		mappingTableMu.Lock()
		mapping, exists := mappingTable[reg.RemotePort]
		mappingTableMu.Unlock()
		if !exists {
			log.Warnf("server", "server.data_channel_no_mapping", reg.RemotePort)
			_ = conn.Close()
			return
		}
		log.Infof("server", "server.data_channel_established", reg.RemotePort)
		// Send data connection to channel (non-blocking)
		select {
		case mapping.DataChan <- conn:
			// Connection is now in the channel, will be used by RelayConn
			// Don't close it here, RelayConn will close it when done
			// This function returns but connection remains alive for relay
		default:
			log.Warnf("server", "server.data_channel_queue_full", reg.RemotePort)
			_ = conn.Close()
		}
		// Don't close connection here - it's being used by RelayConn
		// The connection will be closed by RelayConn when the relay ends
		return
	}
	// For control channel, use defer to close connection when function exits
	defer func() { _ = conn.Close() }()
	mappingTableMu.Lock()
	// Check if port already exists and close old listener
	if oldMapping, exists := mappingTable[reg.RemotePort]; exists {
		// Close old listener (safely)
		if oldMapping.ListenDone != nil {
			select {
			case <-oldMapping.ListenDone:
				// Already closed
			default:
				close(oldMapping.ListenDone)
			}
		}
		// Close old data channel queue (safely)
		select {
		case <-oldMapping.DataChan:
			// Channel already closed or empty
		default:
			close(oldMapping.DataChan)
		}
		// Close old control connection
		_ = oldMapping.ClientConn.Close()
	}
	listenDone = make(chan struct{})
	mappingTable[reg.RemotePort] = &Mapping{
		ClientConn:    conn,
		LocalPort:     reg.LocalPort,
		LastHeartbeat: time.Now(),
		DataChan:      make(chan net.Conn, 10), // Buffer for pending data connections
		ListenDone:    listenDone,
	}
	mappingTableMu.Unlock()
	regdRemotePort, regdLocalPort = reg.RemotePort, reg.LocalPort
	log.Infof("server", "server.port_mapping_registered", regdLocalPort, regdRemotePort)
	resp := protocol.RegisterResponse{Type: "register_resp", Status: "ok"}
	msg, _ := json.Marshal(resp)
	if err := protocol.WritePacket(conn, msg); err != nil {
		log.Errorf("server", "server.send_response_failed", err)
		return
	}

	go listenAndForwardWithStop(regdRemotePort, conn, regdLocalPort, listenDone)

	for {
		packet, err := protocol.ReadPacket(conn)
		if err != nil {
			log.Warnf("server", "server.control_channel_disconnected", err)
			// Close the listening port when control channel disconnects
			mappingTableMu.Lock()
			if mapping, exists := mappingTable[regdRemotePort]; exists && mapping.ListenDone != nil {
				select {
				case <-mapping.ListenDone:
					// Already closed
				default:
					close(mapping.ListenDone)
				}
			}
			mappingTableMu.Unlock()
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
				// Close the listening port when control channel disconnects
				mappingTableMu.Lock()
				if mapping, exists := mappingTable[regdRemotePort]; exists && mapping.ListenDone != nil {
					select {
					case <-mapping.ListenDone:
						// Already closed
					default:
						close(mapping.ListenDone)
					}
				}
				mappingTableMu.Unlock()
				break
			}
			continue
		}
		// Handle offline_port request
		var off protocol.OfflinePortRequest
		if err := json.Unmarshal(packet, &off); err == nil && off.Type == "offline_port" {
			log.Infof("server", "server.client_offline_port", off.Port)
			// Actively stop listening and relay
			mappingTableMu.Lock()
			if mapping, exists := mappingTable[off.Port]; exists {
				if mapping.ListenDone != nil {
					select {
					case <-mapping.ListenDone:
						// Already closed
					default:
						close(mapping.ListenDone)
					}
				}
				delete(mappingTable, off.Port)
			}
			mappingTableMu.Unlock()
			continue
		}
		var on protocol.OnlinePortRequest
		if err := json.Unmarshal(packet, &on); err == nil && on.Type == "online_port" {
			log.Infof("server", "server.client_online_port", on.Port)
			// Re-listen on the port
			mappingTableMu.Lock()
			if mapping, exists := mappingTable[on.Port]; exists {
				if mapping.ListenDone != nil {
					select {
					case <-mapping.ListenDone:
						// Already closed
					default:
						close(mapping.ListenDone)
					}
				}
				listenDone = make(chan struct{})
				mapping.ListenDone = listenDone
			} else {
				listenDone = make(chan struct{})
			}
			mappingTableMu.Unlock()
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
				// Send open_data_channel command to client
				req := protocol.RegisterRequest{Type: "open_data_channel", LocalPort: localPort}
				reqBytes, _ := json.Marshal(req)
				if err := protocol.WritePacket(clientConn, reqBytes); err != nil {
					log.Errorf("server", "server.send_data_channel_cmd_failed", err)
					_ = userConn.Close()
					return
				}
				// Wait for data channel connection from client
				mappingTableMu.Lock()
				mapping, exists := mappingTable[remotePort]
				mappingTableMu.Unlock()
				if !exists {
					log.Warnf("server", "server.mapping_not_found", remotePort)
					_ = userConn.Close()
					return
				}
				// Wait for data channel connection with timeout (increased to 60 seconds)
				waitStart := time.Now()
				select {
				case dataConn := <-mapping.DataChan:
					waitDuration := time.Since(waitStart)
					log.Infof("server", "server.data_channel_connected", remotePort, waitDuration.Milliseconds())
					log.Debugf("server", "server.relay_starting", remotePort)
					// Relay user connection to data channel connection
					core.RelayConn(userConn, dataConn)
					log.Debugf("server", "server.relay_finished", remotePort)
				case <-time.After(60 * time.Second):
					log.Warnf("server", "server.data_channel_timeout", remotePort)
					_ = userConn.Close()
				}
			}()
		}
	}
}
