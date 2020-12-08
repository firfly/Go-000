package main

import (
	"context"
	"github.com/firfly/taliatoolkits/errgroup"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const serverShutdownDuration = 10 * time.Second

func main() {
	g := errgroup.WithContext(context.Background())

	g.Go(func(ctx context.Context) error {
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
		select {
		case <-sigs:
			log.Println("catch system term signal, quit all tasks in group")
			g.StopAll()
		case <-ctx.Done():
		}
		return nil
	})

	g.Go(func(ctx context.Context) error { return newServer(ctx, ":9000", g.StopAll) })
	g.Go(func(ctx context.Context) error { return newServer(ctx, ":9001", g.StopAll) })
	g.Go(func(ctx context.Context) error { return newServer(ctx, ":9002", g.StopAll) })

	if err := g.Wait(); err != nil {
		log.Fatal(err)
	}
}


// newServer 启动一个新的服务
func newServer(ctx context.Context, addr string, afterShutdownFn func()) error {
	s := &http.Server{Addr: addr}
	log.Println(addr + " server is starting")

	// 当前 server 退出时执行的逻辑，这里为调用 StoplLl 将其他的 server 也退掉
	s.RegisterOnShutdown(afterShutdownFn)

	go func() {
		// 监听退出信号
		<-ctx.Done()
		log.Println(addr + " server is shutting down")
		shutdownCtx, cancelFunc := context.WithDeadline(context.Background(), time.Now().Add(serverShutdownDuration))
		defer func() {
			log.Println(addr + " server shuts down")
			cancelFunc()
		}()
		_ = s.Shutdown(shutdownCtx)
	}()

	err := s.ListenAndServe()
	if err != nil {
		if err == http.ErrServerClosed {
			err = nil
		} else {
			log.Println(addr+" server started failed", err)
		}
	}
	return err
}