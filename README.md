# distributed-pagerank

## Requirements

- Protocol Buffers
  - `protoc`: required for compilation
  - `protobuf-devel` (or equivalent): required to import common message types

## Getting Started

Setup `.env` following `.env.example`
Setup `config.json` (not required)

## Building

- Protocol Buffers
  ```bash
  protoc \
  --go_out=. \
  --go_opt=paths=source_relative \
  --go-grpc_out=. \
  --go-grpc_opt=paths=source_relative \
  proto/node.proto && \
  protoc \
  --go_out=. \
  --go_opt=paths=source_relative \
  --go-grpc_out=. \
  --go-grpc_opt=paths=source_relative \
  proto/jobs.proto
  ```
- Node: `go build`

### Docker Compose

Run
```
docker compose up --build
```

## Notes 

- When running on localhost, using Docker Compose, the client can connect to the 
  API server directly by the port number: e.g. `:<port>`
- When running on localhost, using Docker Compose, the MASTER env var, has to be
  set like this: `<master_service_name>:<master_port>`
