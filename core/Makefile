BINARY  := sc
OUTDIR  := bin

# Read version from plugin.json, default to "dev"
VERSION := $(shell grep '"version"' ../.claude-plugin/plugin.json 2>/dev/null | head -1 | sed 's/.*"version": *"\([^"]*\)".*/v\1/')
ifeq ($(VERSION),)
  VERSION := dev
endif

GOFLAGS := -trimpath -ldflags="-s -w -X solon-core/cmd.Version=$(VERSION)"

.PHONY: build build-darwin-arm64 build-darwin-amd64 build-linux-amd64 build-linux-arm64 build-windows-amd64 build-all clean test

build:
	go build $(GOFLAGS) -o $(OUTDIR)/$(BINARY) .

build-darwin-arm64:
	GOOS=darwin GOARCH=arm64 go build $(GOFLAGS) -o $(OUTDIR)/$(BINARY)-darwin-arm64 .

build-darwin-amd64:
	GOOS=darwin GOARCH=amd64 go build $(GOFLAGS) -o $(OUTDIR)/$(BINARY)-darwin-amd64 .

build-linux-amd64:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(GOFLAGS) -o $(OUTDIR)/$(BINARY)-linux-amd64 .

build-linux-arm64:
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build $(GOFLAGS) -o $(OUTDIR)/$(BINARY)-linux-arm64 .

build-windows-amd64:
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build $(GOFLAGS) -o $(OUTDIR)/$(BINARY)-windows-amd64.exe .

build-all: build-darwin-arm64 build-darwin-amd64 build-linux-amd64 build-linux-arm64 build-windows-amd64

clean:
	rm -rf $(OUTDIR)/$(BINARY)*

test:
	go test ./...
