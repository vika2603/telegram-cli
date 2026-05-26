# tg build helpers.
#
# Most workflows use `go install ./cmd/tg` directly; this Makefile exists for
# tagged builds where the binary's `tg --version` output should carry an
# implementation tag (used during the parallel daemon-* work) plus the short
# git commit hash. Tag defaults to "dev" so a bare `make install` still works.

TAG ?= dev
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
VERSION := 0.0.0-$(TAG)+$(COMMIT)

LDFLAGS := -X 'github.com/vika2603/telegram-cli/internal/program.version=$(VERSION)'

.PHONY: install build test lint check

install:
	go install -ldflags "$(LDFLAGS)" ./cmd/tg

build:
	go build -ldflags "$(LDFLAGS)" -o ./tg ./cmd/tg

test:
	go test ./...

lint:
	gofmt -l .
	go vet ./...
	golangci-lint run

check: test lint
