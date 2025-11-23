package health

import (
	"gotunnel/pkg/log"
	"net"
	"time"
)

// ProbeTCPAlive 检查本地某端口存活性，timeout可配置
func ProbeTCPAlive(addr string, timeout time.Duration) bool {
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err == nil {
		_ = conn.Close()
		return true
	}
	return false
}

// PeriodicProbe 调用，定时探测本地端口，发现down后可通知主流程下线
func PeriodicProbe(target string, interval time.Duration, onDead func(), onAlive func()) {
	aliveLast := true
	for {
		ok := ProbeTCPAlive(target, time.Second)
		if ok {
			if !aliveLast && onAlive != nil {
				onAlive()
			}
			aliveLast = true
			log.Debugf("health", "health.port_healthy", target)
		} else {
			if aliveLast && onDead != nil {
				onDead()
			}
			aliveLast = false
			log.Warnf("health", "health.port_unreachable", target)
		}
		time.Sleep(interval)
	}
}

// 示例用法：
// health.PeriodicProbe("127.0.0.1:2222", 30*time.Second, func(){...}, func(){...})
