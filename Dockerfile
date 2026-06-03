# Build stage
FROM docker.io/golang:1.26.4-alpine AS builder

WORKDIR /build

# Copy go module files first for better caching
COPY go.mod go.sum ./

RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 go build -o rmd-client .

FROM docker.io/ubuntu:26.04 AS llama-cpp-files

# Download and extract llama.cpp
WORKDIR /
ADD https://github.com/ggml-org/llama.cpp/releases/download/b9415/llama-b9415-bin-ubuntu-x64.tar.gz /
RUN tar -xvf llama-b9415-bin-ubuntu-x64.tar.gz

# Combine
FROM docker.io/ubuntu:26.04
RUN apt-get update && apt-get install -y libgomp1
COPY --from=llama-cpp-files /llama-b9415 /app
COPY --from=builder /build/rmd-client /app/rmd-client
RUN mkdir -p /var/app/data
VOLUME /var/app/data
ENV RMCLUSTER_CLIENT_DATA_DIR=/var/app/data
ENV PATH=/app

# Set entrypoint
ENTRYPOINT ["/app/rmd-client"]