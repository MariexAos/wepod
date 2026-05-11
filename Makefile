BIN      := bin/wepod
PKG      := ./cmd/wepod
VERSION  := $(shell git describe --tags --always 2>/dev/null || echo dev)
LDFLAGS  := -s -w -X main.version=$(VERSION)

.PHONY: build test test-race vet lint fmt check install clean tidy

# `make check` mirrors CI — run before every commit.
check: fmt vet lint test-race

build:
	CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -o $(BIN) $(PKG)

test:
	go test -count=1 ./...

test-race:
	go test -race -count=1 ./...

vet:
	go vet ./...

lint:
	golangci-lint run --timeout=5m

fmt:
	@out=$$(gofmt -l .); \
	if [ -n "$$out" ]; then echo "Files need gofmt:"; echo "$$out"; exit 1; fi

install: build
	install -m 0755 $(BIN) /usr/local/bin/

tidy:
	go mod tidy

clean:
	rm -rf bin/
