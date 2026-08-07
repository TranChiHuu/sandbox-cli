VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X github.com/tranchihuu/sandbox-cli/internal/cli.Version=$(VERSION)

PREFIX ?= /usr/local
GO_IMAGE ?= golang:1.23-alpine
GO_CACHE ?= $(HOME)/.cache/sandbox-cli-go
HOST_OS := $(shell uname -s | tr '[:upper:]' '[:lower:]')
HOST_ARCH := $(shell uname -m | sed 's/x86_64/amd64/; s/aarch64/arm64/')

# Build in a container so a host Go toolchain is optional; Docker is already required
# to run sandbox at all. The cache is a host directory, not a named volume, so it is
# owned by the invoking user and --user can write to it.
GO_IN_DOCKER = mkdir -p $(GO_CACHE) bin && docker run --rm \
	-v "$(CURDIR)":/src -w /src \
	-v "$(GO_CACHE)":/gocache \
	--user $(shell id -u):$(shell id -g) \
	-e HOME=/gocache -e GOCACHE=/gocache/build -e GOMODCACHE=/gocache/mod \
	-e CGO_ENABLED=0 $(GO_TARGET) \
	$(GO_IMAGE) go

.PHONY: build install test image clean build-docker install-docker test-docker

build:
	go build -ldflags '$(LDFLAGS)' -o bin/sandbox ./cmd/sandbox

install:
	go install -ldflags '$(LDFLAGS)' ./cmd/sandbox

test:
	go test ./...

# Cross-compile for the host; test-docker leaves GO_TARGET empty so tests build for
# linux and can actually run inside the container.
build-docker: GO_TARGET = -e GOOS=$(HOST_OS) -e GOARCH=$(HOST_ARCH)
build-docker:
	$(GO_IN_DOCKER) build -ldflags '$(LDFLAGS)' -o bin/sandbox ./cmd/sandbox

install-docker: build-docker
	install -d $(PREFIX)/bin && install -m 0755 bin/sandbox $(PREFIX)/bin/sandbox

test-docker:
	$(GO_IN_DOCKER) test ./...

image:
	docker build -t sandbox-cli:latest - < internal/docker/Dockerfile

clean:
	rm -rf bin
