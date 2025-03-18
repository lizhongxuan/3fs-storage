package craq

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

// NodeState represents the state of a node in the CRAQ chain
type NodeState int

const (
	// NodeStateUnknown represents an unknown node state
	NodeStateUnknown NodeState = iota
	// NodeStateUp represents a node that is up and running
	NodeStateUp
	// NodeStateDown represents a node that is down
	NodeStateDown
	// NodeStateSuspect represents a node that is suspected to be down
	NodeStateSuspect
)

// NodeRole represents the role of a node in the CRAQ chain
type NodeRole int

const (
	// NodeRoleUnknown represents an unknown role
	NodeRoleUnknown NodeRole = iota
	// NodeRoleHead represents the head of the chain
	NodeRoleHead
	// NodeRoleTail represents the tail of the chain
	NodeRoleTail
	// NodeRoleMiddle represents a node in the middle of the chain
	NodeRoleMiddle
)

// Node represents a node in the CRAQ chain
type Node struct {
	ID       string
	Address  string
	IsHead   bool
	IsTail   bool
	NextNode *Node
	PrevNode *Node
}

// BlockVersion represents a specific version of a block in CRAQ
type BlockVersion struct {
	Version   int
	Data      []byte
	Metadata  []byte
	Timestamp int64
	Clean     bool // true if this version is clean (committed)
}

// Block represents a replicated block in the CRAQ system
type Block struct {
	ID       string
	Versions []*BlockVersion
	mu       sync.RWMutex
}

// Chain represents a CRAQ replication chain
type Chain struct {
	chainLength   int
	replicaFactor int
	nodes         []*Node
	head          *Node
	tail          *Node
	blocks        map[string]*Block
	mu            sync.RWMutex
}

// NewChain creates a new CRAQ chain
func NewChain(chainLength, replicaFactor int) (*Chain, error) {
	if chainLength <= 0 {
		return nil, errors.New("chain length must be greater than zero")
	}

	if replicaFactor <= 0 || replicaFactor > chainLength {
		return nil, errors.New("replica factor must be greater than zero and less than or equal to chain length")
	}

	return &Chain{
		chainLength:   chainLength,
		replicaFactor: replicaFactor,
		nodes:         make([]*Node, 0, chainLength),
		blocks:        make(map[string]*Block),
	}, nil
}

// AddNode adds a node to the CRAQ chain
func (c *Chain) AddNode(id, address string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.nodes) >= c.chainLength {
		return errors.New("chain is already at maximum length")
	}

	// Create new node
	node := &Node{
		ID:      id,
		Address: address,
	}

	// If this is the first node, it's both head and tail
	if len(c.nodes) == 0 {
		node.IsHead = true
		node.IsTail = true
		c.head = node
		c.tail = node
	} else {
		// Add to the end of the chain
		node.IsTail = true
		node.PrevNode = c.tail
		c.tail.IsTail = false
		c.tail.NextNode = node
		c.tail = node
	}

	c.nodes = append(c.nodes, node)
	return nil
}

// Initialize initializes the CRAQ chain
func (c *Chain) Initialize() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.nodes) == 0 {
		return errors.New("cannot initialize an empty chain")
	}

	// In a real implementation, we would set up connections to nodes here
	return nil
}

// Write writes a block to the CRAQ chain
func (c *Chain) Write(blockID string, data []byte, metadata []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.head == nil {
		return errors.New("chain has no head node")
	}

	// Get or create block
	block, ok := c.blocks[blockID]
	if !ok {
		block = &Block{
			ID:       blockID,
			Versions: make([]*BlockVersion, 0),
		}
		c.blocks[blockID] = block
	}

	block.mu.Lock()
	defer block.mu.Unlock()

	// Calculate next version number
	nextVersion := 1
	if len(block.Versions) > 0 {
		nextVersion = block.Versions[len(block.Versions)-1].Version + 1
	}

	// Create new version
	version := &BlockVersion{
		Version:   nextVersion,
		Data:      data,
		Metadata:  metadata,
		Timestamp: time.Now().UnixNano(),
		Clean:     false, // Mark as dirty until propagated
	}

	// Add new version
	block.Versions = append(block.Versions, version)

	// In a real implementation, we would propagate to all nodes in the chain
	// For this mock implementation, we'll just mark it clean after a delay
	go func() {
		time.Sleep(100 * time.Millisecond) // Simulate propagation delay
		block.mu.Lock()
		defer block.mu.Unlock()
		for _, v := range block.Versions {
			if v.Version == nextVersion {
				v.Clean = true
				break
			}
		}
	}()

	return nil
}

// Read reads a block from the CRAQ chain
func (c *Chain) Read(blockID string) ([]byte, []byte, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	block, ok := c.blocks[blockID]
	if !ok {
		return nil, nil, fmt.Errorf("block %s not found", blockID)
	}

	block.mu.RLock()
	defer block.mu.RUnlock()

	if len(block.Versions) == 0 {
		return nil, nil, fmt.Errorf("block %s has no versions", blockID)
	}

	// Find the latest clean version
	var latestCleanVersion *BlockVersion
	for i := len(block.Versions) - 1; i >= 0; i-- {
		if block.Versions[i].Clean {
			latestCleanVersion = block.Versions[i]
			break
		}
	}

	if latestCleanVersion == nil {
		// If no clean version, return the oldest version
		latestCleanVersion = block.Versions[0]
	}

	return latestCleanVersion.Data, latestCleanVersion.Metadata, nil
}

// Delete deletes a block from the CRAQ chain
func (c *Chain) Delete(blockID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, ok := c.blocks[blockID]; !ok {
		return fmt.Errorf("block %s not found", blockID)
	}

	delete(c.blocks, blockID)

	// In a real implementation, we would propagate the delete to all nodes
	return nil
}

// GetStats returns statistics about the CRAQ chain
func (c *Chain) GetStats() (map[string]interface{}, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	stats := make(map[string]interface{})
	stats["node_count"] = len(c.nodes)
	stats["block_count"] = len(c.blocks)

	var totalVersions int
	for _, block := range c.blocks {
		block.mu.RLock()
		totalVersions += len(block.Versions)
		block.mu.RUnlock()
	}
	stats["total_versions"] = totalVersions

	return stats, nil
}

// IsHeadNode returns true if the node with the given ID is the head node
func (c *Chain) IsHeadNode(nodeID string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	if c.head == nil {
		return false
	}
	
	return c.head.ID == nodeID
}

// IsTailNode returns true if the node with the given ID is the tail node
func (c *Chain) IsTailNode(nodeID string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	if c.tail == nil {
		return false
	}
	
	return c.tail.ID == nodeID
}

// GetNodeCount returns the number of nodes in the chain
func (c *Chain) GetNodeCount() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	return len(c.nodes)
} 