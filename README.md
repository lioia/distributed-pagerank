# distributed-pagerank

## Requirements

- Protocol Buffers
  - `protoc`: required for compilation
  - `protobuf-devel` (or equivalent): required to import common message types

## Getting Started

Setup `.env` following `.env.example`

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
- Server (Node) 
  ```bash
  go build -ldflags="-s -w" -o build/server cmd/server/main.go
  ```
- Client
  ```bash
  go build -ldflags="-s -w" -o build/client cmd/client/main.go
  ```

### Running

Local:
- RabbitMQ:
  ```bash
  docker run -it --rm --name rabbitmq -p 5672:5672 -p 15672:15672 rabbitmq
  ```
- Server:
  ```bash
  ./build/server
  ```
- Client:
  ```bash
  ./build/client
  ```

Docker Compose:
```bash
docker compose up --build
```

### AWS

Requirements:
- `ansible`: [installation instructions](https://docs.ansible.com/ansible/2.9/installation_guide/intro_installation.html)
- `terraform`: [installation instructions](https://developer.hashicorp.com/terraform/downloads?product_intent=terraform)
- AWS CLI: [installation instructions](https://docs.aws.amazon.com/cli/latest/userguide/getting-started-install.html)
- `python`: run automated script

Getting started:
- Install AWS plugin for Ansible:
  ```bash
  ansible-galaxy collection install amazon.aws
  ```
- Configure variables in `aws/terraform.tfvars` as desired
- Configure AWS CLI in `$HOME/.aws/credentials`
- Download `key.pem`

Run:
```bash
python aws/deploy.py
```

## Notes - Docker Compose

- MASTER env var, has to be set like this: `<master_service_name>:<master_port>`
- To get the web client IP, run `docker ps` to get the container id
  and `docker inspect --format='{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' $id`

## To Do 
- Send and print master node on web client
