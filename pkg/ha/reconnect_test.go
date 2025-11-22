package ha

import (
	"sync/atomic"
	"testing"
	"time"
)

func TestReconnectLoop_Success(t *testing.T) {
	sleepHook = func(time.Duration) {} // mock sleep，测试秒过
	defer func() { sleepHook = time.Sleep }()
	done := make(chan struct{})
	var tries int32
	go func() {
		ok := ReconnectLoop(func() bool {
			atomic.AddInt32(&tries, 1)
			return tries == 3 // 第三次连接才成功
		}, 1, 5, 10)
		if !ok {
			t.Errorf("3次后应成功返回true")
		}
		done <- struct{}{}
	}()
	<-done
	if tries != 3 {
		t.Errorf("期望3次后连通，实际%v", tries)
	}
}

func TestReconnectLoop_Timeout(t *testing.T) {
	sleepHook = func(time.Duration) {}
	defer func() { sleepHook = time.Sleep }()
	done := make(chan struct{})
	var count int32
	go func() {
		ok := ReconnectLoop(func() bool {
			atomic.AddInt32(&count, 1)
			return false
		}, 1, 2, 3)
		if ok {
			t.Errorf("失败时应返回false")
		}
		done <- struct{}{}
	}()
	<-done
	if count != 3 {
		t.Errorf("最大尝试应为3次, got=%v", count)
	}
}
