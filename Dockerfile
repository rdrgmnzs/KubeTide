# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /workspace

# Install security scanner
RUN go install github.com/securego/gosec/v2/cmd/gosec@latest

# Copy Go module files and download dependencies
COPY go.mod go.sum* ./
RUN go mod download

# Copy source code
COPY cmd/ cmd/
COPY pkg/ pkg/

# Build with security flags
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -o kubetide cmd/kubetide/main.go

# Run security scan
RUN gosec -quiet ./...

# Final stage
FROM gcr.io/distroless/static:nonroot

WORKDIR /

# Copy the binary
COPY --from=builder /workspace/kubetide .

# Use non-root user
USER 65532:65532

# Expose metrics and health probe ports
EXPOSE 8080 8081

ENTRYPOINT ["/kubetide"]