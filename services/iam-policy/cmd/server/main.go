package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/auth-platform/iam-policy-service/internal/config"
	"github.com/auth-platform/iam-policy-service/internal/grpc/handlers"
	"github.com/auth-platform/iam-policy-service/internal/policy"
	pb "github.com/auth-platform/iam-policy-service/proto"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize policy engine
	engine, err := policy.NewEngine(cfg.PolicyPath)
	if err != nil {
		log.Fatalf("Failed to initialize policy engine: %v", err)
	}

	// Start policy watcher for hot reload
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go engine.WatchPolicies(ctx, cfg.PolicyPath)

	// Create gRPC server
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", cfg.Host, cfg.Port))
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	server := grpc.NewServer()
	iamService := handlers.NewIAMPolicyService(engine, cfg)
	pb.RegisterIAMPolicyServiceServer(server, iamService)
	reflection.Register(server)

	// Graceful shutdown
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		log.Println("Shutting down...")
		server.GracefulStop()
		cancel()
	}()

	log.Printf("IAM Policy Service listening on %s:%d", cfg.Host, cfg.Port)
	if err := server.Serve(listener); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
