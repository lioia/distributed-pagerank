#!/bin/bash
# $1: key.pem
# $2: Public IP Address
# $3: Rabbit User
# $4: Rabbit Password

echo "Installing RabbitMQ in $1"
echo "Importing RabbitMQ Keys"
ssh -o StrictHostKeyChecking=no -i $1 ec2-user@$2 <<EOF
sudo rpm --import 'https://github.com/rabbitmq/signing-keys/releases/download/3.0/rabbitmq-release-signing-key.asc'
sudo rpm --import 'https://github.com/rabbitmq/signing-keys/releases/download/3.0/cloudsmith.rabbitmq-erlang.E495BB49CC4BBE5B.key'
sudo rpm --import 'https://github.com/rabbitmq/signing-keys/releases/download/3.0/cloudsmith.rabbitmq-server.9F4587F226208342.key'
EOF
echo "Copying RabbitMQ Repository"
scp -o StrictHostKeyChecking=no -i $1 rabbitmq.repo ec2-user@$2:/home/ec2-user/rabbitmq.repo
echo "Installing, starting and configuring RabbitMQ"
ssh -o StrictHostKeyChecking=no -i $1 ec2-user@$2 <<EOF
sudo mv /home/ec2-user/rabbitmq.repo /etc/yum.repos.d/rabbitmq.repo
sudo yum update -y
sudo yum install -y socat logrotate erlang rabbitmq-server
sudo systemctl enable --now rabbitmq-server.service
sudo rabbitmqctl add_user $3 $4
sudo rabbitmqctl set_user_tags $3 administrator
sudo rabbitmqctl set_permisssions $3 ".*" ".*" ".*"
sudo systemctl restart rabbitmq-server.service
EOF
