package node

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"

	"github.com/3fs-storage/internal/block"
	"github.com/3fs-storage/internal/craq"
	"github.com/3fs-storage/internal/rdma"
	"github.com/3fs-storage/internal/storage"
	"github.com/3fs-storage/pkg/config"
)

// StorageNode represents a node in the storage service cluster
type StorageNode struct {
	cfg           *config.Config
	blockService  *block.Service
	craqChain     *craq.Chain
	rdmaTransport *rdma.Transport
	localStorage  *storage.LocalStorage
	
	listener      net.Listener
	isRunning     bool
	mu            sync.Mutex
	ctx           context.Context
	cancel        context.CancelFunc
}

// NewStorageNode creates a new storage node with the provided configuration
func NewStorageNode(cfg *config.Config) (*StorageNode, error) {
	if cfg == nil {
		return nil, errors.New("configuration cannot be nil")
	}

	ctx, cancel := context.WithCancel(context.Background())
	
	// Initialize local storage
	localStorage, err := storage.NewLocalStorage(cfg.Storage.Local.DataPath, cfg.Storage.Local.MaxSpaceGB)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to initialize local storage: %w", err)
	}
	
	// Initialize RDMA transport (if available)
	var rdmaTransport *rdma.Transport
	rdmaTransport, err = rdma.NewTransport(ctx)
	if err != nil {
		// Fall back to TCP if RDMA is not available
		fmt.Printf("Warning: RDMA not available, falling back to TCP: %v\n", err)
		rdmaTransport = nil
	}
	
	// Initialize CRAQ chain
	craqChain, err := craq.NewChain(cfg.Storage.Replication.ChainLength, cfg.Storage.Replication.Factor)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to initialize CRAQ chain: %w", err)
	}
	
	// Add this node to the chain
	if err := craqChain.AddNode(cfg.Storage.Node.ID, cfg.Storage.Node.ListenAddress); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to add node to CRAQ chain: %w", err)
	}
	
	// Add other nodes from the configuration
	for _, nodeInfo := range cfg.Storage.Cluster.Nodes {
		if nodeInfo.ID != cfg.Storage.Node.ID {
			if err := craqChain.AddNode(nodeInfo.ID, nodeInfo.Address); err != nil {
				cancel()
				return nil, fmt.Errorf("failed to add node %s to CRAQ chain: %w", nodeInfo.ID, err)
			}
		}
	}
	
	// Initialize block service
	blockService, err := block.NewService(localStorage, craqChain)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to initialize block service: %w", err)
	}
	
	return &StorageNode{
		cfg:           cfg,
		blockService:  blockService,
		craqChain:     craqChain,
		rdmaTransport: rdmaTransport,
		localStorage:  localStorage,
		ctx:           ctx,
		cancel:        cancel,
	}, nil
}

// Start starts the storage node
func (n *StorageNode) Start() error {
	n.mu.Lock()
	defer n.mu.Unlock()
	
	if n.isRunning {
		return errors.New("node is already running")
	}
	
	// Initialize local storage
	if err := n.localStorage.Initialize(); err != nil {
		return fmt.Errorf("failed to initialize local storage: %w", err)
	}
	
	// Initialize block service
	if err := n.blockService.Initialize(); err != nil {
		return fmt.Errorf("failed to initialize block service: %w", err)
	}
	
	// Start RDMA transport if available
	if n.rdmaTransport != nil {
		if err := n.rdmaTransport.Start(n.cfg.Storage.Node.ListenAddress); err != nil {
			return fmt.Errorf("failed to start RDMA transport: %w", err)
		}
	} else {
		// Fall back to TCP if RDMA is not available
		var err error
		n.listener, err = net.Listen("tcp", n.cfg.Storage.Node.ListenAddress)
		if err != nil {
			return fmt.Errorf("failed to start TCP listener: %w", err)
		}
		
		// Start accepting connections
		go n.acceptConnections()
	}
	
	n.isRunning = true
	
	return nil
}

// Stop stops the storage node
func (n *StorageNode) Stop() error {
	n.mu.Lock()
	defer n.mu.Unlock()
	
	if !n.isRunning {
		return errors.New("node is not running")
	}
	
	// Cancel the context to stop background operations
	n.cancel()
	
	// Stop RDMA transport if available
	if n.rdmaTransport != nil {
		if err := n.rdmaTransport.Stop(); err != nil {
			return fmt.Errorf("failed to stop RDMA transport: %w", err)
		}
	} else if n.listener != nil {
		// Close TCP listener if used
		if err := n.listener.Close(); err != nil {
			return fmt.Errorf("failed to close TCP listener: %w", err)
		}
	}
	
	// Flush local storage
	if err := n.localStorage.Flush(); err != nil {
		return fmt.Errorf("failed to flush local storage: %w", err)
	}
	
	n.isRunning = false
	
	return nil
}

// acceptConnections accepts incoming TCP connections
func (n *StorageNode) acceptConnections() {
	for {
		conn, err := n.listener.Accept()
		if err != nil {
			// Check if we're shutting down
			select {
			case <-n.ctx.Done():
				return
			default:
				fmt.Printf("Error accepting connection: %v\n", err)
				continue
			}
		}
		
		go n.handleConnection(conn)
	}
}

// handleConnection handles an incoming TCP connection
func (n *StorageNode) handleConnection(conn net.Conn) {
	defer conn.Close()
	
	// In a real implementation, we would handle protocol-specific commands
	// For this mock implementation, we'll just close the connection
	fmt.Printf("Received connection from %s\n", conn.RemoteAddr().String())
}

// GetNodeID returns the ID of this node
func (n *StorageNode) GetNodeID() string {
	return n.cfg.Storage.Node.ID
}

// IsRunning returns whether the node is running
func (n *StorageNode) IsRunning() bool {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.isRunning
} 