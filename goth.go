package goth

import (
	"context"
	"fmt"
	"runtime/debug"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	statusStopped int32 = iota
	statusRunning
	statusWaiting
	statusCrashed
)

type ThreadFunc func(ctx context.Context)

type ThreadConf struct {
	Restart      int32         `json:"restart"`
	RestartDelay time.Duration `json:"restart_delay"`
}

type ThreadStat struct {
	Restarts int32 `json:"restarts"`
	Crashes  int32 `json:"crashes"`
	Status   int32 `json:"status"`
}

type Thread struct {
	ctx  context.Context
	cls  context.CancelFunc
	stop chan struct{}
	conf ThreadConf
	stat ThreadStat
	fn   ThreadFunc
	clct *StatCollector
}

func NewThread(conf ThreadConf, fn ThreadFunc) *Thread {
	ctx, cls := context.WithCancel(context.Background())
	return &Thread{
		ctx:  ctx,
		cls:  cls,
		stop: nil,
		conf: conf,
		stat: ThreadStat{},
		fn:   fn,
		clct: nil,
	}
}

func (t *Thread) WithContext(ctx context.Context) *Thread {
	t.cls()
	t.ctx, t.cls = context.WithCancel(ctx)
	return t
}

// creates prometheus metric collector and register it.
// panics in case of error.
// watch for label uniqueness!
func (t *Thread) WithCollector(label string) *Thread {
	t.clct = NewStatCollector(label, &t.stat)
	prometheus.MustRegister(t.clct)
	return t
}

// signals thread to stop.
func (t *Thread) Stop() <-chan struct{} {
	t.cls()
	return t.stop
}

// waiting for stop.
func (t *Thread) Wait() <-chan struct{} {
	return t.stop
}

// cleanup
func (t *Thread) cleanup() {
	// unregister prometheus collector
	if t.clct != nil {
		prometheus.Unregister(t.clct)
		t.clct.stat = nil
		t.clct = nil
	}
}

// terminates thread and cleans up all referenced objects.
func (t *Thread) Terminate() {
	t.cls()
	<-t.stop
	t.cleanup()
}

// run thread loop
func (t *Thread) Run() *Thread {
	go t.loop()
	return t
}

// thread loop
func (t *Thread) loop() {
	t.stop = make(chan struct{}, 1)
	defer close(t.stop)
	for {
		// check context
		select {
		case <-t.ctx.Done():
			return
		default:
		}
		// execute
		t.execute()
		// restart?
		if !t.restart() {
			return
		}
	}
}

func (t *Thread) execute() {
	atomic.StoreInt32(&t.stat.Status, statusRunning)
	defer func() {
		if v := recover(); v != nil {
			atomic.AddInt32(&t.stat.Crashes, 1)
			atomic.StoreInt32(&t.stat.Status, statusCrashed)
			fmt.Printf("thread crashed: %+v\n%s\n", v, string(debug.Stack()))
		} else {
			atomic.StoreInt32(&t.stat.Status, statusStopped)
		}
	}()
	t.fn(t.ctx)
}

func (t *Thread) restart() bool {
	// restart?
	switch {
	case t.conf.Restart == 0:
		fallthrough
	case t.conf.Restart > 0 && t.stat.Restarts >= t.conf.Restart:
		return false
	}
	// restart!
	if t.conf.RestartDelay > 0 {
		atomic.StoreInt32(&t.stat.Status, statusWaiting)
		select {
		case <-t.ctx.Done():
			return false
		case <-time.After(t.conf.RestartDelay):
		}
	} else {
		select {
		case <-t.ctx.Done():
			return false
		default:
		}
	}
	atomic.AddInt32(&t.stat.Restarts, 1)
	return true
}
