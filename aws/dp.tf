terraform {
  required_providers {
    aws = {
      source = "hashicorp/aws"
    }
  }
}

provider "aws" {
  profile = "default"
  region  = "us-east-1"
}

# Create AWS VPC
resource "aws_vpc" "dp-vpc" {
  cidr_block           = var.vpc_cidr
  enable_dns_support   = true
  enable_dns_hostnames = true

  tags = {
    Name = "${var.tag}-vpc"
  }
}

# Create public subnet 
resource "aws_subnet" "dp-subnet" {
  vpc_id                  = aws_vpc.dp-vpc.id
  cidr_block              = var.vpc_cidr
  map_public_ip_on_launch = true

  tags = {
    Name = "${var.tag}-subnet"
  }
}

# Create Internet Gateway 
resource "aws_internet_gateway" "dp-ig" {
  vpc_id = aws_vpc.dp-vpc.id

  tags = {
    Name = "${var.tag}-ig"
  }
}

# Create route table 
resource "aws_route_table" "dp-route-table" {
  vpc_id = aws_vpc.dp-vpc.id

  route {
    cidr_block = "0.0.0.0/0"
    gateway_id = aws_internet_gateway.dp-ig.id
  }

  tags = {
    Name = "${var.tag}-route-table"
  }
}

# Associate Route Table to Subnet
resource "aws_route_table_association" "crta-subnet" {
  subnet_id      = aws_subnet.dp-subnet.id
  route_table_id = aws_route_table.dp-route-table.id
}

# Create Security Group
resource "aws_security_group" "dp-security-group" {
  vpc_id      = aws_vpc.dp-vpc.id
  description = "DP Security Group"

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  # SSH used by Ansible
  ingress {
    cidr_blocks = ["0.0.0.0/0"]
    from_port   = 22
    to_port     = 22
    protocol    = "tcp"
  }

  # 5678 used by client
  ingress {
    cidr_blocks = ["0.0.0.0/0"]
    from_port   = 5678
    to_port     = 5678
    protocol    = "tcp"
  }

  # 1234 used by node
  ingress {
    cidr_blocks = [var.vpc_cidr]
    from_port   = 1234
    to_port     = 1234
    protocol    = "tcp"
  }

  # 5672 used by RabbitMQ
  ingress {
    cidr_blocks = [var.vpc_cidr]
    from_port   = 5672
    to_port     = 5672
    protocol    = "tcp"
  }

  tags = {
    Name = "${var.tag}-security-group"
  }
}

# Create EC2 instance for RabbitMQ
resource "aws_instance" "dp-mq" {
  ami                         = var.aws_ami
  instance_type               = var.instance
  subnet_id                   = aws_subnet.dp-subnet.id
  vpc_security_group_ids      = [aws_security_group.dp-security-group.id]
  associate_public_ip_address = true
  key_name                    = var.key_pair

  tags = {
    Name = "${var.tag}-mq"
  }
}

# Create EC2 instance for initial master
resource "aws_instance" "dp-master" {
  ami                    = var.aws_ami
  instance_type          = var.instance
  subnet_id              = aws_subnet.dp-subnet.id
  vpc_security_group_ids = [aws_security_group.dp-security-group.id]
  key_name               = var.key_pair

  tags = {
    Name = "${var.tag}-node-master"
  }
}

# Create EC2 instance for initial workers
resource "aws_instance" "dp-worker" {
  count                  = var.worker_count
  ami                    = var.aws_ami
  instance_type          = var.instance
  subnet_id              = aws_subnet.dp-subnet.id
  vpc_security_group_ids = [aws_security_group.dp-security-group.id]
  key_name               = var.key_pair

  tags = {
    Name = "${var.tag}-node-worker"
  }
}

# Output variables (used by ansible)
output "dp-mq-host" {
  value = aws_instance.dp-mq.private_ip
}

output "dp-mq-user" {
  value = var.mq_user
}

output "dp-mq-password" {
  value     = var.mq_password
  sensitive = true
}

output "dp-master-host" {
  value = aws_instance.dp-master.public_ip
}

# Additional variables (used by bash scripts)
output "dp-workers-hosts" {
  value = aws_instance.dp-worker.*.public_ip
}

output "dp-mq-public-host" {
  value = aws_instance.dp-mq.public_ip
}
