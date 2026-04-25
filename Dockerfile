FROM golang:1.25-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o exorcist ./cmd/exorcist

FROM alpine:latest

# Install signal-cli
RUN apk add --no-cache \
    openjdk17-jre \
    wget \
    ca-certificates

# Install signal-cli
RUN wget https://github.com/AsamK/signal-cli/releases/download/v0.13.9/signal-cli-0.13.9-Linux.tar.gz && \
    tar xf signal-cli-0.13.9-Linux.tar.gz -C /opt && \
    ln -sf /opt/signal-cli-0.13.9/bin/signal-cli /usr/local/bin/

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/exorcist .

# Create data directory for database
RUN mkdir -p /data

ENV DB_PATH=/data/exorcist.db

CMD ["./exorcist"]
