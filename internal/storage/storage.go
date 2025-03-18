package storage

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
)

// LocalStorage provides local storage operations for blocks
type LocalStorage struct {
	dataPath  string
	maxSizeGB int
	cache     map[string][]byte
	mu        sync.RWMutex
}

// NewLocalStorage creates a new local storage manager
func NewLocalStorage(dataPath string, maxSizeGB int) (*LocalStorage, error) {
	if dataPath == "" {
		return nil, fmt.Errorf("data path cannot be empty")
	}
	
	if maxSizeGB <= 0 {
		return nil, fmt.Errorf("max size must be greater than zero")
	}
	
	return &LocalStorage{
		dataPath:  dataPath,
		maxSizeGB: maxSizeGB,
		cache:     make(map[string][]byte),
	}, nil
}

// Initialize creates the necessary directories for the storage
func (s *LocalStorage) Initialize() error {
	// Create the main data directory
	if err := os.MkdirAll(s.dataPath, 0755); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}
	
	// Create subdirectories for sharding
	for i := 0; i < 256; i++ {
		subdir := filepath.Join(s.dataPath, fmt.Sprintf("%02x", i))
		if err := os.MkdirAll(subdir, 0755); err != nil {
			return fmt.Errorf("failed to create shard directory %s: %w", subdir, err)
		}
	}
	
	return nil
}

// getBlockPath returns the path to store a block based on its ID
func (s *LocalStorage) getBlockPath(blockID string) string {
	if len(blockID) < 2 {
		blockID = "00" + blockID
	}
	shard := blockID[:2]
	return filepath.Join(s.dataPath, shard, blockID)
}

// getMetadataPath returns the path to store a block's metadata
func (s *LocalStorage) getMetadataPath(blockID string) string {
	return s.getBlockPath(blockID) + ".meta"
}

// WriteBlock writes a block to the local storage
func (s *LocalStorage) WriteBlock(blockID string, data []byte, metadata []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// Get the path for the block
	blockPath := s.getBlockPath(blockID)
	
	// Write the block data
	if err := ioutil.WriteFile(blockPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write block data: %w", err)
	}
	
	// Write metadata if provided
	if metadata != nil {
		metaPath := s.getMetadataPath(blockID)
		if err := ioutil.WriteFile(metaPath, metadata, 0644); err != nil {
			// Try to clean up the block file if metadata write fails
			os.Remove(blockPath)
			return fmt.Errorf("failed to write block metadata: %w", err)
		}
	}
	
	// Update cache
	s.cache[blockID] = data
	
	return nil
}

// ReadBlock reads a block from the local storage
func (s *LocalStorage) ReadBlock(blockID string) ([]byte, []byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	// Check cache first
	if data, ok := s.cache[blockID]; ok {
		// Still need to read metadata from disk
		hasMetadata, metadata, err := s.ReadBlockMetadata(blockID)
		if err != nil {
			return nil, nil, err
		}
		if !hasMetadata {
			return nil, nil, fmt.Errorf("block %s not found", blockID)
		}
		return data, metadata, nil
	}
	
	// Get the path for the block
	blockPath := s.getBlockPath(blockID)
	
	// Check if the block exists
	if _, err := os.Stat(blockPath); os.IsNotExist(err) {
		return nil, nil, fmt.Errorf("block %s not found", blockID)
	}
	
	// Read the block data
	data, err := ioutil.ReadFile(blockPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read block data: %w", err)
	}
	
	// Read the metadata if it exists
	_, metadata, err := s.ReadBlockMetadata(blockID)
	if err != nil {
		return nil, nil, err
	}
	
	// Update cache
	s.cache[blockID] = data
	
	return data, metadata, nil
}

// ReadBlockMetadata reads a block's metadata from the local storage
func (s *LocalStorage) ReadBlockMetadata(blockID string) (bool, []byte, error) {
	metaPath := s.getMetadataPath(blockID)
	
	// Check if the metadata exists
	if _, err := os.Stat(metaPath); os.IsNotExist(err) {
		return false, nil, nil
	}
	
	// Read the metadata
	metadata, err := ioutil.ReadFile(metaPath)
	if err != nil {
		return true, nil, fmt.Errorf("failed to read block metadata: %w", err)
	}
	
	return true, metadata, nil
}

// DeleteBlock deletes a block from the local storage
func (s *LocalStorage) DeleteBlock(blockID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// Get the paths
	blockPath := s.getBlockPath(blockID)
	metaPath := s.getMetadataPath(blockID)
	
	// Delete the block data
	if err := os.Remove(blockPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete block data: %w", err)
	}
	
	// Delete the metadata
	if err := os.Remove(metaPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete block metadata: %w", err)
	}
	
	// Remove from cache
	delete(s.cache, blockID)
	
	return nil
}

// Flush writes all cached blocks to disk
func (s *LocalStorage) Flush() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// Clear the cache
	s.cache = make(map[string][]byte)
	
	return nil
}

// GetUsedSpace returns the amount of disk space used by the storage in bytes
func (s *LocalStorage) GetUsedSpace() (int64, error) {
	var size int64
	
	err := filepath.Walk(s.dataPath, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	
	if err != nil {
		return 0, fmt.Errorf("failed to calculate used space: %w", err)
	}
	
	return size, nil
}

// CalculateChecksum returns the SHA-256 checksum of the provided data
func CalculateChecksum(data []byte) []byte {
	hash := sha256.Sum256(data)
	return hash[:]
}

// BlockMetadata represents metadata for a block
type BlockMetadata struct {
	Checksum    string `json:"checksum"`
	Size        int    `json:"size"`
	Version     int    `json:"version"`
	CreatedAt   int64  `json:"created_at"`
	LastModified int64 `json:"last_modified"`
}

// NewBlockMetadata creates new metadata for a block
func NewBlockMetadata(data []byte, version int, createdAt int64) *BlockMetadata {
	checksum := CalculateChecksum(data)
	
	return &BlockMetadata{
		Checksum:    hex.EncodeToString(checksum),
		Size:        len(data),
		Version:     version,
		CreatedAt:   createdAt,
		LastModified: createdAt,
	}
} 