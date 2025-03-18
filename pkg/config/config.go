package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

// Config represents the complete application configuration
type Config struct {
	Storage StorageConfig `yaml:"storage"`
}

// StorageConfig holds the storage service specific configuration
type StorageConfig struct {
	Node        NodeConfig        `yaml:"node"`
	Cluster     ClusterConfig     `yaml:"cluster"`
	Replication ReplicationConfig `yaml:"replication"`
	Local       LocalConfig       `yaml:"local"`
}

// NodeConfig holds the configuration for this specific node
type NodeConfig struct {
	ID            string `yaml:"id"`
	ListenAddress string `yaml:"listen_address"`
}

// ClusterConfig holds the configuration for the storage cluster
type ClusterConfig struct {
	Nodes []NodeInfo `yaml:"nodes"`
}

// NodeInfo represents information about a node in the cluster
type NodeInfo struct {
	ID      string `yaml:"id"`
	Address string `yaml:"address"`
}

// ReplicationConfig holds the configuration for data replication
type ReplicationConfig struct {
	Factor     int `yaml:"factor"`
	ChainLength int `yaml:"chain_length"`
}

// LocalConfig holds the configuration for local storage
type LocalConfig struct {
	DataPath   string `yaml:"data_path"`
	MaxSpaceGB int    `yaml:"max_space_gb"`
}

// LoadConfig loads the configuration from a given file path
func LoadConfig(configPath string) (*Config, error) {
	configFile, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := yaml.Unmarshal(configFile, &config); err != nil {
		return nil, err
	}

	// Apply environment variable overrides if any
	applyEnvironmentOverrides(&config)

	return &config, nil
}

// applyEnvironmentOverrides allows overriding config values with environment variables
func applyEnvironmentOverrides(config *Config) {
	// Examples of environment variables that could override config:
	if nodeID := os.Getenv("STORAGE_NODE_ID"); nodeID != "" {
		config.Storage.Node.ID = nodeID
	}
	if listenAddr := os.Getenv("STORAGE_LISTEN_ADDRESS"); listenAddr != "" {
		config.Storage.Node.ListenAddress = listenAddr
	}
	if dataPath := os.Getenv("STORAGE_DATA_PATH"); dataPath != "" {
		config.Storage.Local.DataPath = dataPath
	}
} 