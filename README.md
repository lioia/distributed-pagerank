# distributed-pagerank

## Requirements

- Protocol Buffers
  - `protoc`: required for compilation
  - `protobuf-devel` (or equivalent): required to import common message types

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
- Configure `config.json` and run
```bash
python deploy/local.py
```
- Start by following the instructions

Docker Compose:
- Configure `config.json` and run
```bash
python deploy/compose.py
```
- Start using
```bash
docker compose up --build
```

### AWS

Requirements:
<!-- - `ansible`: [installation instructions](https://docs.ansible.com/ansible/2.9/installation_guide/intro_installation.html) -->
- `terraform`: [installation instructions](https://developer.hashicorp.com/terraform/downloads?product_intent=terraform)
- AWS CLI: [installation instructions](https://docs.aws.amazon.com/cli/latest/userguide/getting-started-install.html)
- `python`: run automated script

Getting started:
<!-- - Install AWS plugin for Ansible: -->
<!--   ```bash -->
<!--   ansible-galaxy collection install amazon.aws -->
<!--   ``` -->
- Configure variables in `config.json` as desired
- Configure AWS CLI in `$HOME/.aws/credentials`
- Download `labsuser.pem`
  - It might be necessary to change key's permissions: `chmod 400 labsuser.pem`
- Make `aws/client.sh` and `aws/node.sh` executable: 
  - `chmod +x aws/client.sh`
  - `chmod +x aws/node.sh`

Run:
```bash
python aws/deploy.py
```

To delete from AWS run `terraform destroy` in `aws/` folder

## Notes - Docker Compose

- To get the web client IP, run:
```bash
docker inspect --format='{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' `docker ps -f "ancestor=distributed-pagerank-client" | awk 'FNR==2{ print $1 }'`
```

## Project Structure

```
├── aws                       - AWS Deploy Configuration files
│   ├── ansible                 - Ansible deploy (not working, not updated)
│   │   ├── client.aws_ec2.yml
│   │   ├── client.yaml
│   │   ├── deploy_ansible.py
│   │   ├── dp.service.j2
│   │   ├── mq.aws_ec2.yml
│   │   ├── mq.yaml
│   │   ├── node.aws_ec2.yml
│   │   └── node.yaml
│   ├── client.sh               - Deploy Web client script
│   ├── mq.sh                   - Deploy RabbitMQ script
│   ├── node.sh                 - Deploy node script
│   ├── rabbitmq.repo           - RabbitMQ repo (used by mq.sh)
│   ├── dp.tf                   - Terraform specs
│   ├── variables.tf            - Terraform variables
│   └── terraform.tfvars        - Terraform configurable variables
├── config.json               - Configuration for local and docker compose deploy
├── deploy                    - Deploy Scripts for Docker Compose, Local and AWS
│   ├── aws.py                  - AWS deploy script (based on variables.tf)
│   ├── compose.py              - Generate compose.yaml (based on config.json)
│   ├── local.py                - Local deploy instructions (based on config.json)
│   ├── Dockerfile.client       - Dockerfile for the web client
│   └── Dockerfile.server       - Dockerfile for the node
├── go.mod                    - Go dependencies
├── go.sum                    - Go dependencies
├── cmd                       - Entrypoints
│   ├── client
│   │   └── main.go             - Web Client entrypoint
│   └── server
│       └── main.go             - Node entrypoint
├── pkg                       - Code logic
│   ├── graph                   - Graph logic
│   │   ├── graph.go              - Graph loading and random generation
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
└── README.md                 - This file
```
