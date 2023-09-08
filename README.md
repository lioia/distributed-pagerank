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

### AWS

Requirements:
- `ansible`: [installation instructions](https://docs.ansible.com/ansible/2.9/installation_guide/intro_installation.html)
  - `boto3` and `botocore`: dynamic inventory
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
- To enter input to the master node, attach to the Docker image:
  `docker attach <master-service-name>`

## To Do 
- Master starts API service to receive config
  - Results written to file (maybe http server)
- Local client to contact the Master and upload graph
