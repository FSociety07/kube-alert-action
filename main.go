package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"go_heap/webhook"
)

func main() {
	logf.SetLogger(zap.New(zap.UseDevMode(true)))

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	s := &webhook.Server{
		Addr: ":8999",
	}

	log.Println("starting server on 8999")
	if err := s.Start(ctx); err != nil {
		log.Fatalf("server stopped with error %v", err)
	}

	log.Println("server shutdown cleanly")
}
