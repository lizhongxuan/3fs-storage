package rdma

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"time"
)

// ConnectionState represents the state of an RDMA connection
type ConnectionState int

const (
	// ConnectionStateDisconnected represents a disconnected connection
	ConnectionStateDisconnected ConnectionState = iota
	// ConnectionStateConnecting represents a connection in progress
	ConnectionStateConnecting
	// ConnectionStateConnected represents an established connection
	ConnectionStateConnected
	// ConnectionStateError represents a connection in error state
	ConnectionStateError
)

// Connection represents an RDMA connection to a remote node
type Connection struct {
	Address      string
	State        ConnectionState
	LastActivity time.Time
	conn         net.Conn
	mu           sync.Mutex
}

// Transport provides RDMA communication capabilities (simulated with TCP)
type Transport struct {
	connections     map[string]*Connection
	isRDMAAvailable bool
	listener        net.Listener
	ctx             context.Context
	cancel          context.CancelFunc
	mu              sync.RWMutex
}

// NewTransport creates a new RDMA transport
func NewTransport(ctx context.Context) (*Transport, error) {
	childCtx, cancel := context.WithCancel(ctx)
	
	// In a real implementation, we would check if RDMA is available
	// For this mock implementation, we'll just simulate it
	isRDMAAvailable := false

	return &Transport{
		connections:     make(map[string]*Connection),
		isRDMAAvailable: isRDMAAvailable,
		ctx:             childCtx,
		cancel:          cancel,
	}, nil
}

// Start starts the RDMA transport
func (t *Transport) Start(address string) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	
	// Start the listener
	var err error
	t.listener, err = net.Listen("tcp", address)
	if err != nil {
		return fmt.Errorf("failed to start listener: %w", err)
	}
	
	// Start the accept loop
	go t.acceptLoop()
	
	return nil
}

// Stop stops the RDMA transport
func (t *Transport) Stop() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	
	// Cancel the context
	t.cancel()
	
	// Close the listener
	if t.listener != nil {
		if err := t.listener.Close(); err != nil {
			return fmt.Errorf("failed to close listener: %w", err)
		}
	}
	
	// Close all connections
	for _, conn := range t.connections {
		conn.mu.Lock()
		if conn.conn != nil {
			conn.conn.Close()
		}
		conn.State = ConnectionStateDisconnected
		conn.mu.Unlock()
	}
	
	return nil
}

// acceptLoop accepts incoming connections
func (t *Transport) acceptLoop() {
	for {
		select {
		case <-t.ctx.Done():
			return
		default:
			conn, err := t.listener.Accept()
			if err != nil {
				if !errors.Is(err, net.ErrClosed) {
					fmt.Printf("Error accepting connection: %v\n", err)
				}
				return
			}
			
			go t.handleConnection(conn)
		}
	}
}

// handleConnection handles an incoming connection
func (t *Transport) handleConnection(conn net.Conn) {
	// In a real implementation, we would handle RDMA connection setup
	// For this mock implementation, we'll just read and write data
	
	defer conn.Close()
	
	buf := make([]byte, 1024)
	for {
		select {
		case <-t.ctx.Done():
			return
		default:
			n, err := conn.Read(buf)
			if err != nil {
				if err != io.EOF {
					fmt.Printf("Error reading from connection: %v\n", err)
				}
				return
			}
			
			// Process the data
			// In a real implementation, we would handle RDMA commands
			// For this mock implementation, we'll just echo the data back
			if _, err := conn.Write(buf[:n]); err != nil {
				fmt.Printf("Error writing to connection: %v\n", err)
				return
			}
		}
	}
}

// Connect establishes a connection to a remote node
func (t *Transport) Connect(address string) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	
	// Check if already connected
	if conn, ok := t.connections[address]; ok {
		conn.mu.Lock()
		defer conn.mu.Unlock()
		
		if conn.State == ConnectionStateConnected {
			return nil
		}
	}
	
	// Create a new connection
	connection := &Connection{
		Address: address,
		State:   ConnectionStateConnecting,
	}
	t.connections[address] = connection
	
	// Connect to the remote node
	conn, err := net.Dial("tcp", address)
	if err != nil {
		connection.State = ConnectionStateError
		return fmt.Errorf("failed to connect to %s: %w", address, err)
	}
	
	connection.conn = conn
	connection.State = ConnectionStateConnected
	connection.LastActivity = time.Now()
	
	return nil
}

// Disconnect closes a connection to a remote node
func (t *Transport) Disconnect(address string) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	
	conn, ok := t.connections[address]
	if !ok {
		return fmt.Errorf("no connection to %s", address)
	}
	
	conn.mu.Lock()
	defer conn.mu.Unlock()
	
	if conn.State != ConnectionStateConnected {
		return nil
	}
	
	if conn.conn != nil {
		if err := conn.conn.Close(); err != nil {
			return fmt.Errorf("failed to close connection to %s: %w", address, err)
		}
	}
	
	conn.State = ConnectionStateDisconnected
	
	return nil
}

// WriteData writes data to a remote node
func (t *Transport) WriteData(address string, data []byte) error {
	t.mu.RLock()
	conn, ok := t.connections[address]
	t.mu.RUnlock()
	
	if !ok {
		return fmt.Errorf("no connection to %s", address)
	}
	
	conn.mu.Lock()
	defer conn.mu.Unlock()
	
	if conn.State != ConnectionStateConnected {
		return fmt.Errorf("connection to %s is not connected", address)
	}
	
	if _, err := conn.conn.Write(data); err != nil {
		conn.State = ConnectionStateError
		return fmt.Errorf("failed to write data to %s: %w", address, err)
	}
	
	conn.LastActivity = time.Now()
	
	return nil
}

// ReadData reads data from a remote node
func (t *Transport) ReadData(address string) ([]byte, error) {
	t.mu.RLock()
	conn, ok := t.connections[address]
	t.mu.RUnlock()
	
	if !ok {
		return nil, fmt.Errorf("no connection to %s", address)
	}
	
	conn.mu.Lock()
	defer conn.mu.Unlock()
	
	if conn.State != ConnectionStateConnected {
		return nil, fmt.Errorf("connection to %s is not connected", address)
	}
	
	buf := make([]byte, 4096)
	n, err := conn.conn.Read(buf)
	if err != nil {
		if err != io.EOF {
			conn.State = ConnectionStateError
		}
		return nil, fmt.Errorf("failed to read data from %s: %w", address, err)
	}
	
	conn.LastActivity = time.Now()
	
	return buf[:n], nil
}

// IsRDMAAvailable returns whether RDMA is available
func (t *Transport) IsRDMAAvailable() bool {
	return t.isRDMAAvailable
} 