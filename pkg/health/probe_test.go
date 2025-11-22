package health

import (
	"net"
	"testing"
	"time"
)

func TestProbeTCPAlive_ActiveAndInactive(t *testing.T) {
	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().String()
	alive := ProbeTCPAlive(addr, time.Second)
	if !alive {
		t.Errorf("端口已监听应返回true")
	}
	ln.Close()
	ok := ProbeTCPAlive(addr, time.Millisecond*50)
	if ok {
		t.Errorf("端口已关闭应返回false")
	}
}

func TestPeriodicProbe_Calls(t *testing.T) {
	// 利用chan让probe只检测一次后主动结束
	deadCh := make(chan struct{}, 1)
	go func() {
		PeriodicProbe("127.0.0.1:65530", 50*time.Millisecond,
			func() { deadCh <- struct{}{} }, nil)
	}()
	select {
	case <-deadCh:
		// ok，收到回调
	case <-time.After(300 * time.Millisecond):
		t.Fatal("PeriodicProbe未及时触发onDead")
	}
}
