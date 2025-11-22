package health

import (
	"fmt"
	"net"
	"time"
)

// ProbeTCPAlive 检查本地某端口存活性，timeout可配置
func ProbeTCPAlive(addr string, timeout time.Duration) bool {
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err == nil {
		conn.Close()
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
			fmt.Printf("[health] 端口 %s 健康\n", target)
		} else {
			if aliveLast && onDead != nil {
				onDead()
			}
			aliveLast = false
			fmt.Printf("[health] 端口 %s 不可达\n", target)
		}
		time.Sleep(interval)
	}
}

// 示例用法：
// health.PeriodicProbe("127.0.0.1:2222", 30*time.Second, func(){...}, func(){...})
