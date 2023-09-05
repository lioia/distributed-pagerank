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
  proto/*.proto
  ```
- Node
  ```bash
  go build -ldflags="-s -w" -o build/node
  ```

### Running

Local:
- RabbitMQ:
  ```bash
  docker run -it --rm --name rabbitmq -p 5672:5672 -p 15672:15672 rabbitmq
  ```
- Node:
  ```bash
  ./build/node
  ```

Docker Compose:
```bash
docker compose up --build
```

## Notes - Docker Compose

- MASTER env var, has to be set like this: `<master_service_name>:<master_port>`
- To enter input to the master node, attach to the Docker image:
  `docker attach <master-service-name>`
