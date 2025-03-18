# 3FS Storage Service

A Go implementation of the Storage Service component of DeepSeek's Fire-Flyer File System (3FS).

## Overview

The 3FS Storage Service manages the actual data storage and retrieval in the distributed file system. It provides a block storage interface and ensures strong consistency through Chain Replication with Apportioned Queries (CRAQ).

### Key Features

- **Block Storage Interface**: Files are split into blocks for efficient storage and retrieval
- **Strong Consistency**: Implemented using Chain Replication with Apportioned Queries (CRAQ)
- **Distributed Architecture**: Service runs across multiple nodes, each managing local SSDs
- **High Performance**: Utilizes RDMA networking for fast data transmission (with TCP fallback)
- **Replication**: Data blocks are replicated across multiple SSDs for redundancy and fault tolerance

## Architecture

The Storage Service consists of several key components:

### Storage Node

The Storage Node represents a single instance of the storage service. It manages local storage, participates in the CRAQ chain for replication, and handles client requests. Each node can act as a head, tail, or intermediate node in the CRAQ chain.

### Local Storage

The Local Storage component manages the physical storage of data blocks on local SSDs. It provides operations for reading, writing, and deleting blocks, as well as managing metadata associated with each block.

### Block Service

The Block Service provides a higher-level interface for block operations. It coordinates between the local storage and the CRAQ chain to ensure consistency and replication of data.

### CRAQ Chain

The CRAQ (Chain Replication with Apportioned Queries) Chain manages the replication of data across multiple nodes. It ensures that data is consistently replicated and provides versioning for blocks.

### RDMA Transport

The RDMA Transport provides high-performance data transmission between nodes using Remote Direct Memory Access (RDMA) where available. It falls back to TCP if RDMA is not available.

## Implementation Details

### Data Model

- **Block**: The basic unit of storage. Each block has a unique ID, data, and metadata.
- **Block Metadata**: Contains information about a block, including its checksum, size, version, and timestamps.
- **CRAQ Chain**: A chain of nodes responsible for replicating data. Writes go through the head, and reads can be served by any node.

### Consistency Model

The storage service uses CRAQ to provide strong consistency for reads and writes. CRAQ works as follows:

1. Writes are always sent to the head of the chain.
2. The head forwards the write to the next node in the chain.
3. Each node in the chain acknowledges the write to the previous node.
4. When the tail receives the write, it acknowledges it to the previous node.
5. The acknowledgment propagates back to the head, marking the version as "clean."
6. Reads can be served by any node, but only "clean" versions are returned to ensure consistency.

### Storage Efficiency

To optimize storage efficiency, the implementation includes:

1. **Sharding**: Blocks are distributed across subdirectories based on their ID to avoid performance degradation with large numbers of files.
2. **Caching**: Frequently accessed blocks are cached in memory to reduce disk I/O.
3. **Checksumming**: All blocks are checksummed to ensure data integrity.

## Getting Started

### Prerequisites

- Go 1.20 or higher
- RDMA-capable network hardware (optional, for production deployments)
- Multiple nodes with SSDs for distributed setup

### Installation

```bash
# Clone the repository
git clone https://github.com/lizhongxuan/3fs-storage.git
cd 3fs-storage

# Build the service
go build -o 3fs-storage cmd/main.go

# Run the service
./3fs-storage
```

### Configuration

The service can be configured through the `config.yaml` file or environment variables:

```yaml
storage:
  node:
    id: "node1"
    listen_address: "0.0.0.0:7000"
  
  cluster:
    nodes:
      - id: "node1"
        address: "192.168.1.101:7000"
      - id: "node2"
        address: "192.168.1.102:7000"
      - id: "node3"
        address: "192.168.1.103:7000"
  
  replication:
    factor: 3
    chain_length: 3
  
  local:
    data_path: "/data/3fs"
    max_space_gb: 1000
```

Environment variables can override these settings:

- `STORAGE_NODE_ID`: Unique identifier for this node
- `STORAGE_LISTEN_ADDRESS`: Address for this node to listen on
- `STORAGE_DATA_PATH`: Path to store data blocks
- `STORAGE_MAX_SPACE_GB`: Maximum storage space in GB

## API

The Storage Service exposes a gRPC API for internal communication with other 3FS components. The key operations are:

- **WriteBlock**: Write a block to the storage system
- **ReadBlock**: Read a block from the storage system
- **DeleteBlock**: Delete a block from the storage system
- **ReadBlockMetadata**: Read metadata for a block without reading the data

## Development

### Project Structure

```
├── cmd/                 # Command-line applications
│   └── main.go          # Main entry point
├── internal/            # Private application code
│   ├── block/           # Block management
│   ├── craq/            # CRAQ implementation
│   ├── rdma/            # RDMA transport
│   ├── storage/         # Local storage handling
│   └── node/            # Node management
├── pkg/                 # Public libraries
│   ├── api/             # API definitions
│   ├── config/          # Configuration handling
│   └── util/            # Utility functions
├── config/              # Configuration files
│   └── config.yaml      # Default configuration
├── docs/                # Documentation
└── test/                # Tests
```

### Building and Testing

```bash
# Build the service
go build -o 3fs-storage cmd/main.go

# Run tests
go test ./...

# Run with a specific configuration
./3fs-storage -config=path/to/config.yaml
```

## Performance Considerations

The 3FS Storage Service is designed for high performance:

1. **RDMA Transport**: Uses RDMA for high-throughput, low-latency data transfer when available
2. **Efficient Replication**: Minimizes network overhead through chain replication
3. **In-memory Caching**: Caches frequently accessed blocks to reduce disk I/O
4. **Sharded Storage**: Distributes blocks across directories to improve file system performance

## Future Enhancements

- **Compression**: Add support for block-level compression to reduce storage requirements
- **Encryption**: Implement at-rest and in-transit encryption for security
- **Dynamic Resizing**: Allow nodes to join and leave the cluster dynamically
- **Automatic Rebalancing**: Redistribute blocks when nodes are added or removed
- **Quota Management**: Implement per-user or per-group storage quotas

## References

- [DeepSeek 3FS Project](https://github.com/deepseek-ai/Fire-Flyer-File-System)
- [CRAQ: Chain Replication with Apportioned Queries](https://www.cs.cornell.edu/people/egs/papers/sosp09-craq.pdf)
- [RDMA for High-Performance Storage Systems](https://www.usenix.org/conference/fast21/presentation/wei)

## License

This project is licensed under the Apache 2.0 License. 