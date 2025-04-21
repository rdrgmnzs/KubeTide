# Build variables
IMG ?= ghcr.io/rdrgmnzs/kubetide:latest
VERSION ?= $(shell git describe --tags --abbrev=0 2>/dev/null || echo "0.1.0")
PLATFORMS ?= linux/amd64,linux/arm64

# Go variables
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOMOD=$(GOCMD) mod
GOVET=$(GOCMD) vet
GOCOVER=$(GOCMD) tool cover
CGO_ENABLED=0
GOOS=linux
GOARCH=amd64
GO111MODULE=on

# Kubernetes tools
KUBECTL=kubectl

# Helm
HELM=helm

# Build the controller binary
.PHONY: build
build:
	CGO_ENABLED=$(CGO_ENABLED) GOOS=$(GOOS) GOARCH=$(GOARCH) $(GOBUILD) -o bin/kubetide cmd/kubetide/main.go

# Run tests
.PHONY: test
test:
	$(GOTEST) -v ./... -coverprofile=coverage.out

# Run tests with coverage report
.PHONY: test-coverage
test-coverage: test
	$(GOCOVER) -html=coverage.out

# Run linting
.PHONY: lint
lint:
	golangci-lint run --timeout=5m ./...

# Format code
.PHONY: fmt
fmt:
	go fmt ./...

# Verify code
.PHONY: verify
verify: fmt lint vet

# Run go vet
.PHONY: vet
vet:
	$(GOVET) ./...

# Build the docker image
.PHONY: docker-build
docker-build:
	docker build -t ${IMG} .

# Push the docker image
.PHONY: docker-push
docker-push:
	docker push ${IMG}

# Build multi-architecture images
.PHONY: docker-buildx
docker-buildx:
	docker buildx build --platform=$(PLATFORMS) -t ${IMG} --push .

# Generate Helm chart package
.PHONY: helm-package
helm-package:
	$(HELM) package ./charts/kubetide --destination ./dist --version $(VERSION) --app-version $(VERSION)

# Install CRDs into a cluster
.PHONY: install
install:
	$(HELM) install kubetide ./charts/kubetide --namespace kubetide --create-namespace

# Uninstall CRDs from a cluster
.PHONY: uninstall
uninstall:
	$(HELM) uninstall kubetide -n kubetide

# Clean up binaries
.PHONY: clean
clean:
	$(GOCLEAN)
	rm -rf bin

# Update Go dependencies
.PHONY: deps
deps:
	$(GOMOD) tidy

# Generate SBOM (Software Bill of Materials)
.PHONY: sbom
sbom:
	go install github.com/CycloneDX/cyclonedx-gomod/cmd/cyclonedx-gomod@latest
	cyclonedx-gomod mod -output sbom.json -json

# Security scanning
.PHONY: security-scan
security-scan:
	go install github.com/securego/gosec/v2/cmd/gosec@latest
	gosec -quiet ./...

# Full build and verification
.PHONY: all
all: verify build test docker-build

.PHONY: help
help:
	@echo "Available targets:"
	@echo "  build           - Build the controller binary"
	@echo "  test            - Run tests"
	@echo "  test-coverage   - Run tests with coverage report"
	@echo "  lint            - Run linting"
	@echo "  fmt             - Format code"
	@echo "  verify          - Run fmt, lint, and vet"
	@echo "  vet             - Run go vet"
	@echo "  docker-build    - Build the Docker image"
	@echo "  docker-push     - Push the Docker image"
	@echo "  docker-buildx   - Build and push multi-arch Docker images"
	@echo "  helm-package    - Package Helm chart"
	@echo "  install         - Install controller to cluster"
	@echo "  uninstall       - Uninstall controller from cluster"
	@echo "  clean           - Clean up binaries"
	@echo "  deps            - Update Go dependencies"
	@echo "  sbom            - Generate Software Bill of Materials"
	@echo "  security-scan   - Run security scanning"
	@echo "  all             - Run build, test, and docker-build"