package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/3fs-storage/internal/node"
	"github.com/3fs-storage/pkg/config"
)

func main() {
	// Parse command line flags
	configPath := flag.String("config", "config/config.yaml", "Path to configuration file")
	flag.Parse()

	// Load configuration
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize the storage node
	storageNode, err := node.NewStorageNode(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize storage node: %v", err)
	}

	// Start the storage node
	if err := storageNode.Start(); err != nil {
		log.Fatalf("Failed to start storage node: %v", err)
	}
	
	fmt.Println("3FS Storage Service started successfully")
	fmt.Printf("Node ID: %s, Listening on: %s\n", cfg.Storage.Node.ID, cfg.Storage.Node.ListenAddress)

	// Wait for shutdown signal
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	<-signalChan

	fmt.Println("Shutting down 3FS Storage Service...")
	if err := storageNode.Stop(); err != nil {
		log.Fatalf("Error during shutdown: %v", err)
	}
	fmt.Println("Shutdown complete")
} 