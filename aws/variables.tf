variable "tag" {
  description = "Tag"
  type        = string
  default     = "dp"
}

variable "key_pair" {
  description = "Key Pair"
  type        = string
  default     = "vockey"
}

variable "instance" {
  description = "EC2 Instance Type"
  type        = string
  default     = "t2.nano"
}

variable "worker_count" {
  description = "Number of worker instances"
  type        = number
  default     = 3
}

variable "vpc_cidr" {
  description = "CIDR Block"
  type        = string
  default     = "10.0.0.0/16"
}

variable "aws_ami" {
  description = "EC2 AMI"
  type        = string
  default     = "ami-051f7e7f6c2f40dc1"
}
