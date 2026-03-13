BINARY := gve
MODULE := github.com/castle-x/gve
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null | sed 's/^v//' || echo "0.1.0")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

LDFLAGS := -X $(MODULE)/internal/version.Version=$(VERSION) \
           -X $(MODULE)/internal/version.GitCommit=$(COMMIT) \
           -X $(MODULE)/internal/version.BuildDate=$(DATE)

.PHONY: build install clean test

build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) ./cmd/gve

install:
	go install -ldflags "$(LDFLAGS)" ./cmd/gve

clean:
	rm -f $(BINARY)

test:
	go test ./...
