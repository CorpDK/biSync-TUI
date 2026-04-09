BINARY := syncctl
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-X main.version=$(VERSION)"
INSTALL_DIR := $(HOME)/.local/bin

.PHONY: build install test lint clean

build:
	go build $(LDFLAGS) -o $(BINARY) ./cmd/syncctl/

install: build
	mkdir -p $(INSTALL_DIR)
	cp $(BINARY) $(INSTALL_DIR)/$(BINARY)

test:
	go test ./...

lint:
	go vet ./...
	gofmt -l .

clean:
	rm -f $(BINARY)
