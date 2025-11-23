package ha

import (
	"gotunnel/pkg/log"
	"math/rand"
	"time"
)

// 单元测试可替换sleepHook控制耗时/效率
var sleepHook = time.Sleep

// ReconnectLoop 按指数回退+随机jitter方式自动重连，
// dialFunc 是尝试建立连接并返回是否成功的函数。
// baseInterval, maxInterval：基础与最大回退间隔（秒）
// maxTries：最大尝试次数，为0表示无限重试。
// 成功建立连接则返回true，否则最终失败返回false。
func ReconnectLoop(dialFunc func() bool, baseInterval, maxInterval int, maxTries int) bool {
	tries := 0
	interval := baseInterval
	for {
		if maxTries > 0 && tries >= maxTries {
			log.Errorf("ha", "ha.reconnect_max_tries", maxTries)
			return false
		}
		if dialFunc() {
			return true
		}
		d := time.Duration(interval) * time.Second
		jitter := time.Duration(rand.Intn(1000)) * time.Millisecond
		log.Warnf("ha", "ha.reconnect_retry", tries+1, d+jitter)
		sleepHook(d + jitter)
		interval = interval * 2
		if interval > maxInterval {
			interval = maxInterval
		}
		tries++
	}
}

// 示例用法见测试。
