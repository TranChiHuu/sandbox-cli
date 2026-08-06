VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X github.com/tranchihuu/sandbox-cli/internal/cli.Version=$(VERSION)

.PHONY: build install test image clean

build:
	go build -ldflags '$(LDFLAGS)' -o bin/sandbox ./cmd/sandbox

install:
	go install -ldflags '$(LDFLAGS)' ./cmd/sandbox

test:
	go test ./...

image:
	docker build -t sandbox-cli:latest - < internal/docker/Dockerfile

clean:
	rm -rf bin
