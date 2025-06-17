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

// Thread configuration
type ThreadConf struct {
	// Number of process restarts
	//  -1 -- infinite restart
	//   0 -- execute once
	//   n -- execute and then restart n-times
	Restart int32 `json:"restart"`
	// Delay before process restarting
	RestartDelay time.Duration `json:"restart_delay"`
}

type ThreadStat struct {
	Restarts int32 `json:"restarts"`
	Crashes  int32 `json:"crashes"`
	Status   int32 `json:"status"`
}

type Thread struct {
	ctx    context.Context
	fuse   context.Context
	cancel context.CancelFunc
	stop   chan struct{}
	conf   ThreadConf
	stat   ThreadStat
	fn     ThreadFunc
	clct   *StatCollector
}

func NewThread(conf ThreadConf, fn ThreadFunc) *Thread {
	return &Thread{
		ctx:  context.Background(),
		stop: nil,
		conf: conf,
		stat: ThreadStat{},
		fn:   fn,
		clct: nil,
	}
}

func (t *Thread) WithContext(ctx context.Context) *Thread {
	if t.cancel != nil {
		t.cancel()
	}
	if t.stop != nil {
		<-t.stop
	}
	t.ctx = ctx
	return t
}

// Creates prometheus metric collector and register it.
// Panics in case of error.
// Watch for label uniqueness!
func (t *Thread) WithCollector(label string) *Thread {
	t.clct = NewStatCollector(label, &t.stat)
	prometheus.MustRegister(t.clct)
	return t
}

// Signals thread to stop.
func (t *Thread) Stop() <-chan struct{} {
	if t.cancel != nil {
		t.cancel()
	}
	return t.stop
}

// Waiting for stop.
func (t *Thread) Wait() <-chan struct{} {
	return t.stop
}

// Cleanup.
func (t *Thread) cleanup() {
	// unregister prometheus collector
	if t.clct != nil {
		prometheus.Unregister(t.clct)
		t.clct.stat = nil
		t.clct = nil
	}
}

// Terminates thread and cleans up all referenced objects.
func (t *Thread) Terminate() {
	if t.cancel != nil {
		t.cancel()
	}
	if t.stop != nil {
		<-t.stop
	}
	t.cleanup()
}

// Runs thread loop.
func (t *Thread) Run() *Thread {
	if t.stop != nil {
		select {
		case <-t.stop:
		default:
			return t
		}
	}
	t.stop = make(chan struct{}, 1)
	t.fuse, t.cancel = context.WithCancel(t.ctx)
	go t.loop()
	return t
}

func (t *Thread) loop() {
	defer close(t.stop)
	for {
		// check context
		select {
		case <-t.fuse.Done():
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
	t.fn(t.fuse)
}

func (t *Thread) restart() bool {
	// restart?
	switch {
	case t.conf.Restart == 0:
		return false
	case t.conf.Restart > 0 && t.stat.Restarts >= t.conf.Restart:
		return false
	}
	// restart!
	if t.conf.RestartDelay > 0 {
		atomic.StoreInt32(&t.stat.Status, statusWaiting)
		select {
		case <-t.fuse.Done():
			return false
		case <-time.After(t.conf.RestartDelay):
		}
	} else {
		select {
		case <-t.fuse.Done():
			return false
		default:
		}
	}
	atomic.AddInt32(&t.stat.Restarts, 1)
	return true
}
