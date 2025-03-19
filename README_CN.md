# 3FS 存储服务

DeepSeek 火焰飞行者文件系统（Fire-Flyer File System，3FS）存储服务组件的 Go 语言实现。

## 概述

3FS 存储服务管理分布式文件系统中的实际数据存储和检索。它提供块存储接口，并通过带分配查询的链式复制（Chain Replication with Apportioned Queries，CRAQ）确保强一致性。

### 主要特性

- **块存储接口**：文件被分割成块以实现高效存储和检索
- **强一致性**：使用带分配查询的链式复制（CRAQ）实现
- **分布式架构**：服务在多个节点上运行，每个节点管理本地 SSD
- **高性能**：利用 RDMA 网络进行快速数据传输（可回退到 TCP）
- **复制**：数据块在多个 SSD 之间复制以实现冗余和容错

## 架构

存储服务由几个关键组件组成：

### 存储节点

存储节点代表存储服务的单个实例。它管理本地存储，参与 CRAQ 链进行复制，并处理客户端请求。每个节点可以在 CRAQ 链中充当头节点、尾节点或中间节点。

### 本地存储

本地存储组件管理数据块在本地 SSD 上的物理存储。它提供读取、写入和删除块的操作，以及管理与每个块相关的元数据。

### 块服务

块服务提供块操作的更高级接口。它协调本地存储和 CRAQ 链之间的关系，以确保数据的一致性和复制。

### CRAQ 链

CRAQ（带分配查询的链式复制）链管理数据在多个节点之间的复制。它确保数据一致地复制，并为块提供版本控制。

### RDMA 传输

RDMA 传输在可用的情况下，使用远程直接内存访问（RDMA）提供节点间的高性能数据传输。如果 RDMA 不可用，则回退到 TCP。

## 实现细节

### 数据模型

- **块**：存储的基本单位。每个块都有唯一的 ID、数据和元数据。
- **块元数据**：包含有关块的信息，包括校验和、大小、版本和时间戳。
- **CRAQ 链**：负责复制数据的节点链。写入通过头节点，读取可以由任何节点提供。

### 一致性模型

存储服务使用 CRAQ 为读写提供强一致性。CRAQ 的工作原理如下：

1. 写入总是发送到链的头部。
2. 头节点将写入转发到链中的下一个节点。
3. 链中的每个节点向前一个节点确认写入。
4. 当尾节点接收到写入时，它向前一个节点确认。
5. 确认沿着链向头节点传播，将版本标记为"干净"。
6. 读取可以由任何节点提供，但只返回"干净"版本以确保一致性。

### 存储效率

为了优化存储效率，实现包括：

1. **分片**：块根据其 ID 分布在子目录中，以避免大量文件导致性能下降。
2. **缓存**：频繁访问的块被缓存在内存中，以减少磁盘 I/O。
3. **校验和**：对所有块进行校验和计算，以确保数据完整性。

## 开始使用

### 先决条件

- Go 1.20 或更高版本
- RDMA 功能的网络硬件（可选，用于生产部署）
- 具有 SSD 的多个节点，用于分布式设置

### 安装

```bash
# 克隆仓库
git clone https://github.com/lizhongxuan/3fs-storage.git
cd 3fs-storage

# 构建服务
go build -o 3fs-storage cmd/main.go

# 运行服务
./3fs-storage
```

### 配置

可以通过 `config.yaml` 文件或环境变量配置服务：

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

环境变量可以覆盖这些设置：

- `STORAGE_NODE_ID`：此节点的唯一标识符
- `STORAGE_LISTEN_ADDRESS`：此节点要监听的地址
- `STORAGE_DATA_PATH`：存储数据块的路径
- `STORAGE_MAX_SPACE_GB`：最大存储空间（GB）

## API

存储服务为与其他 3FS 组件的内部通信提供 gRPC API。主要操作有：

- **WriteBlock**：向存储系统写入块
- **ReadBlock**：从存储系统读取块
- **DeleteBlock**：从存储系统删除块
- **ReadBlockMetadata**：读取块的元数据，不读取数据

## 开发

### 项目结构

```
├── cmd/                 # 命令行应用程序
│   └── main.go          # 主入口点
├── internal/            # 私有应用程序代码
│   ├── block/           # 块管理
│   ├── craq/            # CRAQ 实现
│   ├── rdma/            # RDMA 传输
│   ├── storage/         # 本地存储处理
│   └── node/            # 节点管理
├── pkg/                 # 公共库
│   ├── api/             # API 定义
│   ├── config/          # 配置处理
│   └── util/            # 实用函数
├── config/              # 配置文件
│   └── config.yaml      # 默认配置
├── docs/                # 文档
└── test/                # 测试
```

### 构建和测试

```bash
# 构建服务
go build -o 3fs-storage cmd/main.go

# 运行测试
go test ./...

# 使用特定配置运行
./3fs-storage -config=path/to/config.yaml
```

## 性能考虑

3FS 存储服务旨在提供高性能：

1. **RDMA 传输**：在可用时使用 RDMA 进行高吞吐量、低延迟的数据传输
2. **高效复制**：通过链式复制最小化网络开销
3. **内存缓存**：缓存频繁访问的块以减少磁盘 I/O
4. **分片存储**：将块分布在目录中以提高文件系统性能

## 未来增强

- **压缩**：添加对块级压缩的支持，以减少存储需求
- **加密**：实现静态和传输中的加密以提高安全性
- **动态调整大小**：允许节点动态加入和离开集群
- **自动重平衡**：添加或删除节点时重新分配块
- **配额管理**：实现按用户或按组的存储配额

## 参考

- [DeepSeek 3FS 项目](https://github.com/deepseek-ai/Fire-Flyer-File-System)
- [CRAQ: 带分配查询的链式复制](https://www.cs.cornell.edu/people/egs/papers/sosp09-craq.pdf)
- [用于高性能存储系统的 RDMA](https://www.usenix.org/conference/fast21/presentation/wei)

## 许可证

本项目基于 Apache 2.0 许可证。 