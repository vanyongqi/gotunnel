package ha

import (
	"errors"
	"net"
	"sync/atomic"
	"testing"
	"time"
)

type failWriteConn struct{ net.Conn }

func (f *failWriteConn) Write(b []byte) (int, error) { return 0, errors.New("fail") }

func TestHeartbeatManager_NoTimeoutOnNormalWrite(t *testing.T) {
	trigger := int32(0)
	mgr := &HeartbeatManager{
		Conn:      &goodWriteConn{},
		Interval:  50 * time.Millisecond,
		OnTimeout: func() { atomic.StoreInt32(&trigger, 1) },
	}
	mgr.StartHeartbeat()
	time.Sleep(120 * time.Millisecond)
	mgr.StopHeartbeat()
	if atomic.LoadInt32(&trigger) != 0 {
		t.Errorf("正常Write不应触发超时回调")
	}
}

type goodWriteConn struct{ net.Conn }

func (g *goodWriteConn) Write(b []byte) (int, error) { return len(b), nil }

func TestHeartbeatManager_TimeoutOnWriteFail(t *testing.T) {
	// 用错误写触发超时
	triggered := make(chan struct{}, 1)
	mgr := &HeartbeatManager{
		Conn:      &failWriteConn{},
		Interval:  30 * time.Millisecond,
		OnTimeout: func() { triggered <- struct{}{} },
	}
	mgr.StartHeartbeat()
	select {
	case <-triggered:
		// ok
	case <-time.After(80 * time.Millisecond):
		t.Fatal("heartbeat timeout callback not triggered on write fail")
	}
}
