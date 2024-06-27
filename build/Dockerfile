# Use the official Go base image
FROM golang:1.20 as builder
ARG TARGETOS
ARG TARGETARCH

# Set the working directory
WORKDIR /app

# Copy go.mod and go.sum files
COPY go.mod go.mod
COPY go.sum go.sum
RUN go env -w GOPROXY=https://goproxy.cn,direct

# Download dependencies
RUN go mod download

# Copy all project files to the working directory
COPY . .

# Compile the Go program, outputting the binary file to /app/bin/cli
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -o squash-docker-image cmd/squash-docker-image/main.go

# Use the official Debian base image as the final base image
FROM alpine:3.14

RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories && apk --no-cache add docker

# Set the working directory
WORKDIR /imagebuild

# Copy the compiled binary file from the build stage
COPY --from=builder /app/squash-docker-image  /imagebuild/squash-docker-image
