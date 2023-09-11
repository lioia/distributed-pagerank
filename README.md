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

## Project Structure

```
├── aws                       - AWS deploy configuration files
│   ├── ansible                 - Ansible deploy (not updated, not working correctly)
│   │   ├── deploy_ansible.py
│   │   ├── dp.service.j2
│   │   ├── mq.aws_ec2.yml
│   │   ├── mq.yaml
│   │   ├── node.aws_ec2.yml
│   │   └── node.yaml
│   ├── client.sh               - Deploy Web client script
│   ├── mq.sh                   - Deploy RabbitMQ script
│   ├── node.sh                 - Deploy node script
│   ├── deploy.py               - General deploy script
│   ├── rabbitmq.repo           - RabbitMQ repo (used by mq.sh)
│   ├── dp.tf                   - Terraform specs
│   ├── variables.tf            - Terraform variables
│   └── terraform.tfvars        - Terraform configurable variables
├── cmd                       - Entrypoints
│   ├── client
│   │   └── main.go             - Web Client entrypoint
│   └── server
│       └── main.go             - Node entrypoint
├── compose.yaml              - Docker Compose configuration
├── Dockerfile.client         - Web Client Docker image
├── Dockerfile.server         - Node Docker image
├── go.mod                    - Go dependencies
├── go.sum                    - Go dependencies
├── pkg                       - Code logic
│   ├── graph                   - Graph logic
│   │   ├── graph.go              - Graph loading
│   │   └── pagerank.go           - PageRank implementation (single node)
│   ├── node                    - gRPC and node logic
│   │   ├── api.go                - gRPC interaction between client and master
│   │   ├── server.go             - gRPC interaction between nodes
│   │   ├── models.go             - Node models
│   │   ├── master.go             - Master node logic
│   │   └── worker.go             - Worker node logic
│   └── utils                   - Utility functions
│       ├── env.go                - Environment variables loading
│       ├── logs.go               - Custom Logging
│       ├── queue.go              - Useful queue functions
│       └── utils.go              - General useful functions
├── proto                     - Protocol Buffers
│   ├── common.proto            - Common messages
│   ├── api.proto               - API services (web client - master)
│   ├── node.proto              - Node services (node - node)
│   └── jobs.proto              - Queue Messages
├── public                    - Files used by web client
│   ├── index.html              - Main web page
│   └── tmpl.html               - Templates page
└── README                    - This file
```
