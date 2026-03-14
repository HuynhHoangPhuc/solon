BINARY  := solon-hooks
OUTDIR  := bin
GOFLAGS := -trimpath -ldflags="-s -w"

.PHONY: build build-darwin-arm64 build-darwin-amd64 build-linux-amd64 build-all clean test

build:
	go build $(GOFLAGS) -o $(OUTDIR)/$(BINARY) .

build-darwin-arm64:
	GOOS=darwin GOARCH=arm64 go build $(GOFLAGS) -o $(OUTDIR)/$(BINARY)-darwin-arm64 .

build-darwin-amd64:
	GOOS=darwin GOARCH=amd64 go build $(GOFLAGS) -o $(OUTDIR)/$(BINARY)-darwin-amd64 .

build-linux-amd64:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(GOFLAGS) -o $(OUTDIR)/$(BINARY)-linux-amd64 .

build-all: build-darwin-arm64 build-darwin-amd64 build-linux-amd64

clean:
	rm -rf $(OUTDIR)/$(BINARY)*

test:
	go test ./...
