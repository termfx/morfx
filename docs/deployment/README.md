# Deployment and Operations Guide

This directory contains comprehensive guides for deploying and operating morfx in production environments.

## Contents

- [Installation Methods](./installation.md) - Different ways to install morfx
- [Configuration Management](./configuration.md) - Production configuration patterns
- [Performance Tuning](./performance.md) - Optimizing for production workloads
- [Monitoring & Observability](./monitoring.md) - Tracking morfx performance
- [Troubleshooting](./troubleshooting.md) - Common issues and solutions

## Installation Options

### Binary Installation

```bash
# Download from GitHub releases
curl -L https://github.com/termfx/morfx/releases/latest/download/morfx-linux-amd64 -o morfx
chmod +x morfx
sudo mv morfx /usr/local/bin/
```

### From Source

```bash
git clone https://github.com/termfx/morfx.git
cd morfx
make build
sudo cp bin/morfx /usr/local/bin/
```

### Container Deployment

```dockerfile
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY . .
RUN make build

FROM alpine:latest
RUN apk --no-cache add ca-certificates
COPY --from=builder /app/bin/morfx /usr/local/bin/morfx
ENTRYPOINT ["morfx"]
```

## Production Considerations

### Performance

- **Database**: SQLite with FTS5 for optimal performance
- **Memory**: Typical usage 10-50MB, scales with codebase size
- **CPU**: Multi-core aware, leverages goroutines for parallel processing
- **I/O**: Optimized for large file operations with batching

### Security

- **Encryption**: Optional data encryption for sensitive code transformations
- **Permissions**: Requires read/write access to target directories
- **Network**: No network connectivity required for core operations

### Scalability

- **File Processing**: Handles codebases with millions of lines
- **Concurrent Operations**: Automatically parallelizes work
- **Memory Efficiency**: Streaming processing for large files
- **Database Optimization**: Automatic indexing and query optimization
