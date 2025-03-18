package block

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/3fs-storage/internal/craq"
	"github.com/3fs-storage/internal/storage"
)

// BlockID is a unique identifier for a block
type BlockID string

// Block represents a data block in the storage system
type Block struct {
	ID       BlockID
	Data     []byte
	Version  uint64
	Checksum []byte
}

// Service manages block operations in the storage system
type Service struct {
	localStorage *storage.LocalStorage
	craqChain    *craq.Chain
	mu           sync.RWMutex
}

// NewService creates a new block service
func NewService(localStorage *storage.LocalStorage, craqChain *craq.Chain) (*Service, error) {
	if localStorage == nil {
		return nil, errors.New("localStorage cannot be nil")
	}

	return &Service{
		localStorage: localStorage,
		craqChain:    craqChain,
	}, nil
}

// WriteBlock writes a block to the storage system
func (s *Service) WriteBlock(blockID string, data []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Create block metadata
	metadata := storage.NewBlockMetadata(data, 1, time.Now().UnixNano())
	metadataBytes, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal block metadata: %w", err)
	}

	// If CRAQ chain is available, replicate the block
	if s.craqChain != nil {
		if err := s.craqChain.Write(blockID, data, metadataBytes); err != nil {
			return fmt.Errorf("failed to replicate block: %w", err)
		}
	}

	// Always write to local storage as well
	if err := s.localStorage.WriteBlock(blockID, data, metadataBytes); err != nil {
		return fmt.Errorf("failed to write block to local storage: %w", err)
	}

	return nil
}

// ReadBlock reads a block from the storage system
func (s *Service) ReadBlock(blockID string) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Try to read from CRAQ chain first if available
	if s.craqChain != nil {
		data, _, err := s.craqChain.Read(blockID)
		if err == nil && data != nil {
			return data, nil
		}
		// Fall back to local storage if CRAQ read fails
	}

	// Read from local storage
	data, _, err := s.localStorage.ReadBlock(blockID)
	if err != nil {
		return nil, fmt.Errorf("failed to read block: %w", err)
	}

	return data, nil
}

// ReadBlockMetadata reads metadata for a block
func (s *Service) ReadBlockMetadata(blockID string) (*storage.BlockMetadata, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Try CRAQ chain first if available
	if s.craqChain != nil {
		_, metadataBytes, err := s.craqChain.Read(blockID)
		if err == nil && metadataBytes != nil {
			var metadata storage.BlockMetadata
			if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
				return nil, fmt.Errorf("failed to unmarshal block metadata: %w", err)
			}
			return &metadata, nil
		}
	}

	// Fall back to local storage
	exists, metadataBytes, err := s.localStorage.ReadBlockMetadata(blockID)
	if err != nil {
		return nil, fmt.Errorf("failed to read block metadata: %w", err)
	}

	if !exists || metadataBytes == nil {
		return nil, fmt.Errorf("block %s not found", blockID)
	}

	var metadata storage.BlockMetadata
	if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal block metadata: %w", err)
	}

	return &metadata, nil
}

// DeleteBlock deletes a block from the storage system
func (s *Service) DeleteBlock(blockID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Delete from CRAQ chain if available
	if s.craqChain != nil {
		if err := s.craqChain.Delete(blockID); err != nil {
			return fmt.Errorf("failed to delete block from replication chain: %w", err)
		}
	}

	// Delete from local storage
	if err := s.localStorage.DeleteBlock(blockID); err != nil {
		return fmt.Errorf("failed to delete block from local storage: %w", err)
	}

	return nil
}

// ListBlocks lists all blocks in the storage system (not implemented yet)
func (s *Service) ListBlocks() ([]string, error) {
	// This would typically scan local storage and/or query the metadata service
	// For now, return a not implemented error
	return nil, errors.New("list blocks operation not implemented")
}

// Initialize initializes the block service
func (s *Service) Initialize() error {
	// Initialize local storage
	if err := s.localStorage.Initialize(); err != nil {
		return fmt.Errorf("failed to initialize local storage: %w", err)
	}

	// Initialize CRAQ chain if available
	if s.craqChain != nil {
		if err := s.craqChain.Initialize(); err != nil {
			return fmt.Errorf("failed to initialize CRAQ chain: %w", err)
		}
	}

	return nil
}

// GetStats returns statistics for the block service
func (s *Service) GetStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Get local storage stats
	usedSpace, err := s.localStorage.GetUsedSpace()
	if err != nil {
		return nil, fmt.Errorf("failed to get storage stats: %w", err)
	}

	stats["used_space_bytes"] = usedSpace

	// Add CRAQ chain stats if available
	if s.craqChain != nil {
		craqStats, err := s.craqChain.GetStats()
		if err == nil {
			for k, v := range craqStats {
				stats["craq_"+k] = v
			}
		}
	}

	return stats, nil
}

// calculateChecksum calculates a checksum for the given data
// In a real implementation, this would use a secure hash function
func calculateChecksum(data []byte) []byte {
	// This is a placeholder. In a real implementation, use a cryptographic hash.
	// For example: sha256.Sum256(data)
	checksum := make([]byte, 4)
	
	// Simple XOR-based checksum for demonstration
	for i, b := range data {
		checksum[i%4] ^= b
	}
	
	return checksum
} 