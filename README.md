# distributed-pagerank

## Requirements

- Protocol Buffers
  - `protoc`: required for compilation
  - `protobuf-devel` (or equivalent): required to import common message types

## Getting Started

Setup  `.env` following `.env.example`

## Building

- Protocol Buffers
  ```bash
  protoc \
  --go_out=. \
  --go_opt=paths=source_relative \
  --go-grpc_out=. \
  --go-grpc_opt=paths=source_relative \
  proto/node.proto
  ```
- Server
  ```bash
  go build -o build/server cmd/server/main.go
  ```
- Client:
  ```bash
  go build -o build/client cmd/client/main.go
  ```

## Notes 

- When running on localhost, using Docker Compose, the client can connect to the 
  API server directly by the port number: e.g. `:<port>`
