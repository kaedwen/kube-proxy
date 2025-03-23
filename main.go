package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	utils "github.com/kaedwen/kube-proxy/pkg"
	"github.com/kaedwen/kube-proxy/pkg/proxy"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := utils.ConfigOrDie()

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("yehaa\n"))
	})

	p := proxy.New(mux)
	if err := p.Run(ctx, cfg, "default"); err != nil {
		panic(err)
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	<-c
	cancel()

	ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := p.Cleanup(ctx); err != nil {
		log.Println("failed to cleanup", err.Error())
	}
}
