#!/bin/bash
# $1: key.pem
# $2: Public IP Address
# $3: Master
# $4: Rabbit Host
# $5: Rabbit User
# $6: Rabbit Password

echo "Installing distributed-pagerank in $1"
echo "Importing Golang and Protocol Buffers"
ssh -o StrictHostKeyChecking=no -i $1 ec2-user@$2 <<EOF
sudo yum install -y golang protobuf-compiler protobuf-devel
go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.28
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.2
mkdir dp
EOF
echo "Copying Application"
scp -o StrictHostKeyChecking=no -i $1 -r ../proto ec2-user@$2:/home/ec2-user/dp/proto
scp -o StrictHostKeyChecking=no -i $1 -r ../pkg ec2-user@$2:/home/ec2-user/dp/pkg
scp -o StrictHostKeyChecking=no -i $1 -r ../cmd ec2-user@$2:/home/ec2-user/dp/cmd
scp -o StrictHostKeyChecking=no -i $1 ../go.mod ec2-user@$2:/home/ec2-user/dp/go.mod
scp -o StrictHostKeyChecking=no -i $1 ../go.sum ec2-user@$2:/home/ec2-user/dp/go.sum
scp -o StrictHostKeyChecking=no -i $1 dp.service ec2-user@$2:/home/ec2-user/dp/dp.service
echo "Compiling protocol buffers, application and enabling service"
ssh -o StrictHostKeyChecking=no -i $1 ec2-user@$2 <<EOF
sudo mv /home/ec2-user/dp/dp.service /etc/systemd/system/dp.service
cd dp
export PATH="$PATH:$(go env GOPATH)/bin"
protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative proto/*.proto
go build -ldflags="-s -w" -o build/node cmd/server/main.go
sudo systemctl daemon-reload
sudo systemctl enable --now dp.service
cd ..
EOF
