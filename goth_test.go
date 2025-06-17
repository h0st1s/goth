package goth

import (
	"context"
	"testing"
	"time"
)

func TestThreadRunOnce(t *testing.T) {
	count := 0
	worker := func(ctx context.Context) { count++ }

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	config := ThreadConf{Restart: 0, RestartDelay: 0}
	thread := NewThread(config, worker).
		WithContext(ctx).
		WithCollector("test").
		Run()
	defer thread.Terminate()

	<-thread.Wait()

	if count == 0 {
		t.Errorf("worker was not executed!")
	} else if count != 1 {
		t.Errorf("worker was executed %d-times. expected: 1", count)
	}
}

func TestThreadRunNTimes(t *testing.T) {
	n := 4
	count := 0
	worker := func(ctx context.Context) { count++ }

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	config := ThreadConf{Restart: int32(n), RestartDelay: 0}
	thread := NewThread(config, worker).
		WithContext(ctx).
		WithCollector("test").
		Run()
	defer thread.Terminate()

	<-thread.Wait()

	if count == 0 {
		t.Errorf("worker was not executed!")
	} else if count != n+1 {
		t.Errorf("worker was executed %d-times. expected: %d", count, n+1)
	}
}

func TestThreadRunInfinite(t *testing.T) {
	count := 0
	worker := func(ctx context.Context) { count++ }

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	config := ThreadConf{Restart: -1, RestartDelay: time.Millisecond * 100}
	thread := NewThread(config, worker).
		WithContext(ctx).
		WithCollector("test").
		Run()
	defer thread.Terminate()

	<-thread.Wait()

	if count == 0 {
		t.Errorf("worker was not executed!")
	} else if count < 2 {
		t.Errorf("worker was executed %d-times. expected: >2", count)
	}
}

func TestThreadRunWithPanic(t *testing.T) {
	count := 0
	worker := func(ctx context.Context) {
		count++
		panic("panic!")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	config := ThreadConf{Restart: 1, RestartDelay: time.Millisecond * 100}
	thread := NewThread(config, worker).
		WithContext(ctx).
		WithCollector("test").
		Run()
	defer thread.Terminate()

	<-thread.Wait()

	if count == 0 {
		t.Errorf("worker was not executed!")
	} else if count != 2 {
		t.Errorf("worker was executed %d-times. expected: 2", count)
	}
}

func TestThreadRunStop(t *testing.T) {
	count := 0
	worker := func(ctx context.Context) {
		count++
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	config := ThreadConf{Restart: -1, RestartDelay: time.Millisecond * 100}
	thread := NewThread(config, worker).
		WithContext(ctx).
		WithCollector("test").
		Run()
	defer thread.Terminate()

	time.Sleep(time.Millisecond * 300)
	<-thread.Stop()
	before := count
	thread.Run()

	<-thread.Wait()

	if count == 0 {
		t.Errorf("worker was not executed!")
	} else if count == before {
		t.Errorf("worker was not started after stop!")
	}
}

func TestThreadRedefineContext(t *testing.T) {
	count := 0
	worker := func(ctx context.Context) {
		count++
	}

	ctx01, cancel01 := context.WithTimeout(context.Background(), time.Second)
	defer cancel01()

	config := ThreadConf{Restart: -1, RestartDelay: time.Millisecond * 100}
	thread := NewThread(config, worker).
		WithContext(ctx01).
		WithCollector("test").
		Run()
	defer thread.Terminate()

	time.Sleep(time.Millisecond * 500)
	ctx02, cancel02 := context.WithTimeout(context.Background(), time.Second)
	defer cancel02()
	thread.WithContext(ctx02)
	before := count
	thread.Run()

	<-thread.Wait()

	if count == 0 {
		t.Errorf("worker was not executed!")
	} else if count == before {
		t.Errorf("worker was not started after stop!")
	}
}

func TestThreadRunRun(t *testing.T) {
	count := 0
	worker := func(ctx context.Context) {
		count++
		select {
		case <-ctx.Done():
		case <-time.After(time.Second * 10):
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	config := ThreadConf{Restart: -1, RestartDelay: time.Millisecond * 100}
	thread := NewThread(config, worker).
		WithContext(ctx).
		WithCollector("test").
		Run()
	defer thread.Terminate()

	time.Sleep(time.Millisecond * 500)
	thread.Run()

	<-thread.Wait()

	if count == 0 {
		t.Errorf("worker was not executed!")
	} else if count > 1 {
		t.Errorf("worker was executed %d-times. expected: 1", count)
	}
}
