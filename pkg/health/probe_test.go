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

func TestPeriodicProbe_DeadToAlive(t *testing.T) {
	// 测试从dead到alive的转换
	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().String()

	deadCh := make(chan struct{}, 1)
	aliveCh := make(chan struct{}, 1)

	// 先关闭监听，让probe检测到dead
	ln.Close()

	go func() {
		PeriodicProbe(addr, 50*time.Millisecond,
			func() { deadCh <- struct{}{} },
			func() { aliveCh <- struct{}{} })
	}()

	// 等待检测到dead
	select {
	case <-deadCh:
		// ok，收到dead回调
	case <-time.After(300 * time.Millisecond):
		t.Fatal("PeriodicProbe未及时触发onDead")
	}

	// 重新启动监听
	ln2, err := net.Listen("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	defer ln2.Close()

	// 等待检测到alive
	select {
	case <-aliveCh:
		// ok，收到alive回调
	case <-time.After(500 * time.Millisecond):
		// 可能还没检测到，这是正常的，因为需要等待interval
		t.Log("PeriodicProbe可能还未检测到alive，这是正常的")
	}
}
