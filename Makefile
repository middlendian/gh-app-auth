BINDIR := ./bin
LDFLAGS := -s -w

.PHONY: build build-mac build-linux build-all clean test lint check

build:
	go build -ldflags "$(LDFLAGS)" -o $(BINDIR)/gh-app-auth .

build-mac:
	GOOS=darwin GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o $(BINDIR)/darwin-arm64/gh-app-auth .
	GOOS=darwin GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(BINDIR)/darwin-amd64/gh-app-auth .

build-linux:
	GOOS=linux GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o $(BINDIR)/linux-arm64/gh-app-auth .
	GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(BINDIR)/linux-amd64/gh-app-auth .

build-all: build-mac build-linux

test:
	go test ./...

lint:
	golangci-lint run ./...

check: test lint
	goreleaser check || true  # brews deprecation warning is intentional (needed for service support)

clean:
	rm -rf $(BINDIR)
