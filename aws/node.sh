#!/bin/bash
# $1: key.pem
# $2: Public IP Address
# $3: Private IP Address

echo "Installing distributed-pagerank in $2"
echo "Installing Golang and Protocol Buffer compiler"
ssh -o StrictHostKeyChecking=no -i $1 ec2-user@$2 "sudo yum install -y golang protobuf-compiler protobuf-devel gcc glibc"
echo "Installing gRPC Protocol Buffer extension"
ssh -o StrictHostKeyChecking=no -i $1 ec2-user@$2 "go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.28"
ssh -o StrictHostKeyChecking=no -i $1 ec2-user@$2 "go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.2"
ssh -o StrictHostKeyChecking=no -i $1 ec2-user@$2 "mkdir -p dp/proto"
echo "Copying Application"
scp -o StrictHostKeyChecking=no -i $1 -r ../proto/*.proto ec2-user@$2:/home/ec2-user/dp/proto
scp -o StrictHostKeyChecking=no -i $1 -r ../pkg ec2-user@$2:/home/ec2-user/dp/pkg
scp -o StrictHostKeyChecking=no -i $1 -r ../cmd ec2-user@$2:/home/ec2-user/dp/cmd
scp -o StrictHostKeyChecking=no -i $1 ../go.mod ec2-user@$2:/home/ec2-user/dp/go.mod
scp -o StrictHostKeyChecking=no -i $1 ../go.sum ec2-user@$2:/home/ec2-user/dp/go.sum
scp -o StrictHostKeyChecking=no -i $1 dp.service_$3 ec2-user@$2:/home/ec2-user/dp/dp.service
echo "Moving systemd service"
ssh -o StrictHostKeyChecking=no -i $1 ec2-user@$2 "sudo mv /home/ec2-user/dp/dp.service /etc/systemd/system/dp.service"
echo "Compiling Protocol Buffers"
ssh -o StrictHostKeyChecking=no -i $1 ec2-user@$2 "cd dp && PATH=\$PATH:\$(go env GOPATH)/bin protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative proto/*.proto"
echo "Building Application"
ssh -o StrictHostKeyChecking=no -i $1 ec2-user@$2 "cd dp && go build -ldflags=\"-s -w\" -o build/node cmd/server/main.go"
echo "Enabling and starting service"
ssh -o StrictHostKeyChecking=no -i $1 ec2-user@$2 "sudo systemctl daemon-reload"
ssh -o StrictHostKeyChecking=no -i $1 ec2-user@$2 "sudo systemctl enable --now dp.service"
