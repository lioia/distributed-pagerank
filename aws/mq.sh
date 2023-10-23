#!/bin/bash
# $1: key.pem
# $2: Public IP Address
# $3: Rabbit User
# $4: Rabbit Password

echo "Installing RabbitMQ in $2"
echo "Importing RabbitMQ Keys"
ssh -o StrictHostKeyChecking=no -i $1 ec2-user@$2 "sudo rpm --import 'https://github.com/rabbitmq/signing-keys/releases/download/3.0/rabbitmq-release-signing-key.asc'"
ssh -o StrictHostKeyChecking=no -i $1 ec2-user@$2 "sudo rpm --import 'https://github.com/rabbitmq/signing-keys/releases/download/3.0/cloudsmith.rabbitmq-erlang.E495BB49CC4BBE5B.key'"
ssh -o StrictHostKeyChecking=no -i $1 ec2-user@$2 "sudo rpm --import 'https://github.com/rabbitmq/signing-keys/releases/download/3.0/cloudsmith.rabbitmq-server.9F4587F226208342.key'"
echo "Copying RabbitMQ Repository"
scp -o StrictHostKeyChecking=no -i $1 rabbitmq.repo ec2-user@$2:/home/ec2-user/rabbitmq.repo
echo "Installing, starting and configuring RabbitMQ"
ssh -o StrictHostKeyChecking=no -i $1 ec2-user@$2 "sudo mv /home/ec2-user/rabbitmq.repo /etc/yum.repos.d/rabbitmq.repo"
ssh -o StrictHostKeyChecking=no -i $1 ec2-user@$2 "sudo yum update -y && sudo yum install -y socat logrotate erlang rabbitmq-server"
ssh -o StrictHostKeyChecking=no -i $1 ec2-user@$2 "sudo systemctl enable --now rabbitmq-server.service"
ssh -o StrictHostKeyChecking=no -i $1 ec2-user@$2 "sudo rabbitmqctl add_user $3 $4"
ssh -o StrictHostKeyChecking=no -i $1 ec2-user@$2 "sudo rabbitmqctl set_user_tags $3 administrator"
ssh -o StrictHostKeyChecking=no -i $1 ec2-user@$2 "sudo rabbitmqctl set_permissions $3 \".*\" \".*\" \".*\""
ssh -o StrictHostKeyChecking=no -i $1 ec2-user@$2 "sudo systemctl restart rabbitmq-server.service"
