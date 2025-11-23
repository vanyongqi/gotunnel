package main

import (
	"context"
	"encoding/json"
	"fmt"
	"gotunnel/pkg/core"
	"gotunnel/pkg/errors"
	"gotunnel/pkg/health"
	"gotunnel/pkg/log"
	"gotunnel/pkg/protocol"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/viper"
)

// ClientConfig holds the client configuration parameters.
type ClientConfig struct {
	Name                string
	Token               string
	ServerAddr          string
	LocalPort           int
	RemotePort          int
	LogLevel            string
	LogLang             string
	HeartbeatInterval   int           // Heartbeat interval in seconds
	HealthCheckInterval time.Duration // Health check interval
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
	remotePort := 10022 // Default value
	if viper.IsSet("client.remote_port") {
		remotePort = viper.GetInt("client.remote_port")
	}
	logLevel := viper.GetString("client.log_level")
	if logLevel == "" {
		logLevel = "info"
	}
	logLang := viper.GetString("client.log_lang")
	if logLang == "" {
		logLang = "zh"
	}
	heartbeatInterval := 10 // Default 10 seconds
	if viper.IsSet("client.heartbeat_interval") {
		heartbeatInterval = viper.GetInt("client.heartbeat_interval")
		if heartbeatInterval <= 0 {
			heartbeatInterval = 10 // Ensure greater than 0
		}
	}
	healthCheckInterval := 30 * time.Second // Default 30 seconds
	if viper.IsSet("client.health_check_interval") {
		intervalSeconds := viper.GetInt("client.health_check_interval")
		if intervalSeconds > 0 {
			healthCheckInterval = time.Duration(intervalSeconds) * time.Second
		}
	}
	return &ClientConfig{
		Name:                name,
		Token:               token,
		ServerAddr:          serverAddr,
		LocalPort:           localPort,
		RemotePort:          remotePort,
		LogLevel:            logLevel,
		LogLang:             logLang,
		HeartbeatInterval:   heartbeatInterval,
		HealthCheckInterval: healthCheckInterval,
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
		// Log error with i18n, but still return error for caller to handle
		log.Errorf("client", "error.register_failed", resp.Reason)
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
func StartHealthProbe(conf *ClientConfig, _ net.Conn, onOffline func(), onOnline func()) (stop func()) {
	doneHealth := make(chan struct{})
	go func() {
		target := fmt.Sprintf("127.0.0.1:%d", conf.LocalPort)
		health.PeriodicProbe(target, conf.HealthCheckInterval, onOffline, onOnline)
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
			log.Infof("client", "client.data_channel_received", ctrl.LocalPort)
			// Handle data channel establishment in a separate goroutine to avoid blocking control loop
			go func(localPort int) {
				startTime := time.Now()
				// Establish a separate data channel connection
				dataConn, err := net.Dial("tcp", conf.ServerAddr)
				if err != nil {
					log.Errorf("client", "client.connect_data_channel_failed", err)
					return
				}
				defer func() {
					// Close data connection if relay fails
					_ = dataConn.Close()
				}()
				connectDuration := time.Since(startTime)
				log.Debugf("client", "client.data_channel_dialed", connectDuration.Milliseconds())
				// Send data channel registration (reuse register format but with data_channel type)
				dataReq := protocol.RegisterRequest{
					Type:       "data_channel",
					LocalPort:  localPort,
					RemotePort: conf.RemotePort,
					Token:      conf.Token,
					Name:       conf.Name,
				}
				dataReqBytes, _ := json.Marshal(dataReq)
				if err := protocol.WritePacket(dataConn, dataReqBytes); err != nil {
					log.Errorf("client", "client.send_data_channel_reg_failed", err)
					return
				}
				// Read response
				respBytes, err := protocol.ReadPacket(dataConn)
				if err != nil {
					log.Errorf("client", "client.read_data_channel_resp_failed", err)
					return
				}
				var resp protocol.RegisterResponse
				_ = json.Unmarshal(respBytes, &resp)
				if resp.Status != "ok" {
					log.Errorf("client", "client.data_channel_reg_failed", resp.Reason)
					return
				}
				regDuration := time.Since(startTime)
				log.Debugf("client", "client.data_channel_registered", regDuration.Milliseconds())
				// Connect to local service
				localAddr := fmt.Sprintf("127.0.0.1:%d", localPort)
				log.Debugf("client", "client.connecting_local", localAddr)
				localConn, err := net.Dial("tcp", localAddr)
				if err != nil {
					log.Errorf("client", "client.connect_local_failed", err)
					_ = dataConn.Close()
					return
				}
				totalDuration := time.Since(startTime)
				log.Infof("client", "client.data_channel_ready", localPort, totalDuration.Milliseconds())
				log.Debugf("client", "client.relay_starting", localPort)
				// Relay on separate data channel connection
				core.RelayConn(localConn, dataConn)
				log.Debugf("client", "client.relay_finished", localPort)
			}(ctrl.LocalPort)
		}
	}
}

// handleConnection handles a single connection lifecycle.
func handleConnection(conn net.Conn, conf *ClientConfig) error {
	// Start heartbeat goroutine
	heartbeatStop := StartHeartbeat(conn, time.Duration(conf.HeartbeatInterval)*time.Second, func() {
		log.Warn("client", "client.heartbeat_timeout", nil)
		_ = conn.Close()
	})
	defer heartbeatStop()

	// Start health probe (using closure to capture state variable)
	var healthDown bool
	stopHealth := StartHealthProbe(conf, conn,
		func() {
			if !healthDown {
				log.Warnf("client", "client.local_port_health_lost", conf.LocalPort)
				req := protocol.OfflinePortRequest{Type: "offline_port", Port: conf.RemotePort}
				b, _ := json.Marshal(req)
				if err := protocol.WritePacket(conn, b); err != nil {
					log.Errorf("client", "client.send_offline_port_failed", err)
				}
				healthDown = true
			}
		},
		func() {
			if healthDown {
				log.Infof("client", "client.local_port_recovered", conf.LocalPort)
				req := protocol.OnlinePortRequest{Type: "online_port", Port: conf.RemotePort}
				b, _ := json.Marshal(req)
				if err := protocol.WritePacket(conn, b); err != nil {
					log.Errorf("client", "client.send_online_port_failed", err)
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

	// Initialize logger
	log.Init(log.ParseLevel(conf.LogLevel), log.ParseLanguage(conf.LogLang))

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Start reconnection loop in a goroutine
	reconnectDone := make(chan struct{})
	go func() {
		defer close(reconnectDone)
		log.Infof("client", "client.port_registered", conf.LocalPort, conf.RemotePort)
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			conn, err := DialServer(conf)
			if err != nil {
				select {
				case <-ctx.Done():
					return
				default:
					errors.PrintError(errors.ErrConnectFailed, err)
					time.Sleep(3 * time.Second)
					continue
				}
			}

			if err := RegisterPort(conn, conf); err != nil {
				_ = conn.Close()
				select {
				case <-ctx.Done():
					return
				default:
					log.Errorf("client", "client.port_register_failed", err)
					time.Sleep(3 * time.Second)
					continue
				}
			}

			log.Info("client", "client.port_register_success", nil)

			// Handle connection in a goroutine so we can check for shutdown
			connDone := make(chan struct{})
			go func() {
				defer close(connDone)
				_ = handleConnection(conn, conf)
			}()

			// Wait for connection to close or shutdown signal
			select {
			case <-ctx.Done():
				_ = conn.Close()
				<-connDone
				return
			case <-connDone:
				log.Warn("client", "client.control_channel_disconnected", nil)
				_ = conn.Close()
				select {
				case <-ctx.Done():
					return
				default:
					time.Sleep(3 * time.Second)
				}
			}
		}
	}()

	// Wait for shutdown signal
	<-sigChan
	log.Info("client", "client.shutdown_signal_received", nil)

	// Start graceful shutdown
	log.Info("client", "client.shutdown_started", nil)
	cancel()

	// Wait for reconnection loop to finish
	<-reconnectDone

	log.Info("client", "client.shutdown_complete", nil)
}
