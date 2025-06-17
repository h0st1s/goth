package main

import (
	"context"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/h0st1s/goth"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	ctx, cancel := osctx()
	defer cancel()

	conf := goth.ThreadConf{Restart: -1, RestartDelay: time.Second}

	srv := goth.NewThread(conf, httpServer).
		WithContext(ctx).
		WithCollector("http_server").
		Run()
	defer srv.Terminate()

	<-srv.Wait()
}

func osctx() (context.Context, context.CancelFunc) {
	// context
	ctx, cancel := context.WithCancel(context.Background())
	// watchdog
	go func() {
		alarm := make(chan os.Signal, 1)
		signal.Notify(alarm, syscall.SIGINT, syscall.SIGTERM)
		s := <-alarm
		fmt.Printf("syscall signal received: %s\n", s.String())
		cancel()
	}()
	// return
	return ctx, cancel
}

func httpServer(ctx context.Context) {
	// define handler
	handler := mux.NewRouter()
	handler.Handle("/metrics", promhttp.Handler()).Methods("GET")
	handler.PathPrefix("/debug/").Handler(http.DefaultServeMux)
	// server
	server := http.Server{
		Addr:         ":9010",
		Handler:      handler,
		ReadTimeout:  time.Second * 30,
		WriteTimeout: time.Second * 600,
	}
	// graceful shutdown
	fuse, cancel := context.WithCancel(ctx)
	defer cancel()
	go func() {
		<-fuse.Done()
		if err := server.Shutdown(context.Background()); err != nil {
			fmt.Printf("[ http ]: server shutdown error: %s\n", err)
		}
	}()
	// listen and serve
	fmt.Printf("[ http ]: online\n")
	defer fmt.Printf("[ http ]: offline\n")
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		fmt.Printf("[ http ]: server error: %s\n", err.Error())
	}
}
