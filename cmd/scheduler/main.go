package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"task-scheduler/internal/api"
	"task-scheduler/internal/grpc"
	"task-scheduler/internal/scheduler"
	"task-scheduler/internal/storage"
)

func main() {
	var (
		nodeID    = flag.String("node-id", "node-1", "Node ID")
		httpAddr  = flag.String("http-addr", "localhost:8080", "HTTP server address")
		grpcAddr  = flag.String("grpc-addr", "localhost:9090", "gRPC server address")
		redisAddr = flag.String("redis-addr", "localhost:6379", "Redis server address")
		peers     = flag.String("peers", "", "Comma-separated list of peer addresses")
	)
	flag.Parse()

	redisClient := storage.NewRedisClient(*redisAddr, "", 0)

	ctx := context.Background()
	if err := redisClient.Ping(ctx); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	log.Printf("Connected to Redis at %s", *redisAddr)

	storage := storage.NewRedisStorage(redisClient)

	var peerList []string
	if *peers != "" {
		peerList = []string{*peers}
	}

	sched := scheduler.NewScheduler(*nodeID, *httpAddr, storage, peerList)

	if err := sched.Start(); err != nil {
		log.Fatalf("Failed to start scheduler: %v", err)
	}

	apiServer := api.NewAPI(sched)

	grpcServer := grpc.NewServer(sched)

	go func() {
		log.Printf("Starting HTTP server on %s", *httpAddr)
		if err := apiServer.Run(*httpAddr); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start HTTP server: %v", err)
		}
	}()

	go func() {
		log.Printf("Starting gRPC server on %s", *grpcAddr)
		if err := grpcServer.Start(*grpcAddr); err != nil {
			log.Fatalf("Failed to start gRPC server: %v", err)
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down...")

	grpcServer.Stop()

	sched.Stop()

	log.Println("Shutdown complete")
}
