# CRAQ-Based Distributed Chunk Storage System

This project implements a **distributed file storage system** using the **CRAQ protocol**, optimized for reliable and versioned chunked file storage.

## 🧱 Architecture

```
          +---------+             +---------+             +---------+
          |  Node 1 |  ───────▶   |  Node 2 |  ───────▶   |  Node 3 |
          |  (Head) |             |         |             |  (Tail) |
          +---------+             +---------+             +---------+
```

- Chain replication: Head → Middle → Tail
- Tail is source of truth and final commit point
- All nodes write to local disk + update DB metadata

## ✨ Features

- ✅ Stream-based file ingestion using gRPC
- ✅ CRAQ-style head-to-tail chain replication
- ✅ Dirty/Clean chunk tracking
- ✅ Manager node for:
  - Head node discovery (write)
  - Read node selection (tail preferred, round-robin fallback)
- ✅ Storage backed by CockroachDB
- ✅ Simple client library for write/read

## 📁 Directory Structure

```
craq-cluster/
├── cmd/
│   ├── node/       # Node startup code
│   └── client/     # CLI client to upload/read files
├── gen/
│   └── rpcpb/      # Generated gRPC stubs
├── internal/
│   └── config/     # Node config loader
├── pkg/
│   ├── craq/       # CRAQ logic (Node, Server)
│   └── storage/    # DiskStore and CraqStore
├── proto/
│   └── node.proto  # gRPC schema
└── README.md
```

## 🛠 Build Instructions

### 1. Export Go bin path

```bash
export PATH="$PATH:$(go env GOPATH)/bin"
```

### 2. Generate gRPC code from proto

```bash
export PATH="$PATH:$(go env GOPATH)/bin"

protoc \
  --proto_path=proto \
  --go_out=gen/managerpb \
  --go-grpc_out=gen/managerpb \
  proto/manager.proto

protoc \
  --proto_path=proto \
  --go_out=gen/rpcpb \
  --go-grpc_out=gen/rpcpb \
  proto/node.proto
```

### 3. Build node binary

```bash
go build -o craq-node cmd/node/main.go
```

### 4. Build client binary

```bash
go build -o craq-client cmd/client/main.go
```

## 🚀 Running the System

### 1. Configure nodes (config/config.json)

```json
{
  "nodes": [
    {"id": "n1", "addr": "localhost:8001", "isHead": true},
    {"id": "n2", "addr": "localhost:8002"},
    {"id": "n3", "addr": "localhost:8003", "isTail": true}
  ],
  "db": {
    "addr": "postgresql://root@<your-db-ip>:26257/craq?sslmode=disable"
  }
}
```

### 2. Start Nodes

```bash
NODE_ID=n1 ./craq-node
NODE_ID=n2 ./craq-node
NODE_ID=n3 ./craq-node
```

### 3. Start Manager

```bash
./craq-manager
```

## 🧪 Usage

### Upload a File

```bash
./craq-client ./README.md chunk-readme
```

### Read a Chunk

The client automatically reads from the tail and prints chunk content.

## 🧬 Database Schema

```sql
CREATE TABLE IF NOT EXISTS public.chunk_metadata (
  folder STRING NOT NULL DEFAULT '/',
  file_name STRING NOT NULL,
  seq INT8 NOT NULL,
  state STRING NOT NULL,
  path STRING NOT NULL,
  CONSTRAINT pk_folder_file PRIMARY KEY (folder, file_name)
);
```


