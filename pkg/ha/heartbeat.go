package ha

import (
	"encoding/json"
	"fmt"
	"gotunnel/pkg/protocol"
	"net"
	"time"
)

// HeartbeatManager 管理心跳发送和监测。用于客户端和服务端控制通道健康。
type HeartbeatManager struct {
	Conn      net.Conn      // 关联的网络连接
	Interval  time.Duration // 心跳包发送间隔
	OnTimeout func()        // 超时断开等回调
	stopChan  chan struct{}
}

// StartHeartbeat 启动定时发送ping包的goroutine。适用于客户端。
func (h *HeartbeatManager) StartHeartbeat() {
	h.stopChan = make(chan struct{})
	go func() {
		ticker := time.NewTicker(h.Interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				ping := protocol.HeartbeatPing{Type: "ping", Time: time.Now().Unix()}
				b, _ := json.Marshal(ping)
				err := protocol.WritePacket(h.Conn, b)
				if err != nil {
					fmt.Println("[ha] 心跳包发送失败", err)
					if h.OnTimeout != nil {
						h.OnTimeout()
					}
					return
				}
			case <-h.stopChan:
				return
			}
		}
	}()
}

// StopHeartbeat 停止心跳。
func (h *HeartbeatManager) StopHeartbeat() {
	close(h.stopChan)
}

// 服务端用：检查所有client最后心跳时间，超时回调。需要配合映射表调用。
func HeartbeatCheckLoop(checkFunc func()) {
	for {
		checkFunc()
		time.Sleep(5 * time.Second)
	}
}

// 示例用法
// cMgr := &ha.HeartbeatManager{Conn: conn, Interval: 10 * time.Second, OnTimeout: func(){...}}
// cMgr.StartHeartbeat() // 客户端心跳
// cMgr.StopHeartbeat() // 停止心跳
// ha.HeartbeatCheckLoop(自定义check函数) // 服务端心跳轮询
