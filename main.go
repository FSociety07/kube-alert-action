package main

import (
	"context"
	"go_heap/api/v1alpha1"
	"go_heap/webhook"
	"log"
	"os/signal"
	"syscall"

	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
)

func main() {
	cfg := ctrl.GetConfigOrDie()

	if err := v1alpha1.AddToScheme(scheme.Scheme); err != nil {
		log.Fatalf("Unable to add scheme: %v", err)
	}

	mgr, err := ctrl.NewManager(cfg, ctrl.Options{Scheme: scheme.Scheme})
	if err != nil {
		log.Fatalf("Unable to start manager: %v", err)
	}

	if err := mgr.Add(&webhook.Server{Client: mgr.GetClient(), Addr: ":8080"}); err != nil {
		log.Fatalf("Unable to add webhook server to the manager: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	if err := mgr.Start(ctx); err != nil {
		log.Fatalf("Unable to start manager: %v", err)
	}

}
