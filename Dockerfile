FROM golang:1.20-alpine

ARG PORT

WORKDIR /app

# Update distro packages, install protoc & install protoc generators
RUN apk update && \
    apk add --no-cache protoc && \
    apk add --no-cache protobuf-dev && \
    go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.28 && \
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.2

# Update env to include go binaries
# https://grpc.io/docs/languages/go/quickstart/
ENV PATH="$PATH:$(go env GOPATH)/bin"

# Copy & install additional packages required
COPY go.mod go.sum ./
RUN go mod download

# Copy Source Code
COPY cmd/ cmd/ 
COPY pkg/ pkg/ 
COPY proto/ proto/ 

# Compile Protocol Buffers and server
RUN protoc \
    --go_out=. \
    --go_opt=paths=source_relative \
    --go-grpc_out=. \
    --go-grpc_opt=paths=source_relative \
    proto/node.proto && \
    CGO_ENABLED=0 GOOS=linux go build -o server cmd/server/main.go

EXPOSE $PORT

CMD [ "/app/server" ]
