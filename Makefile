APP := promptq
VERSION ?= 0.1.0
PKG := ./cmd/promptq
LDFLAGS := -s -w -X github.com/ast-lw/promptq/internal/promptq.Version=$(VERSION)

.PHONY: test build run debug clean

test:
	go test ./...

build:
	mkdir -p dist
	go build -trimpath -ldflags "$(LDFLAGS)" -o dist/$(APP) $(PKG)

run:
	go run $(PKG) studio

debug:
	PROMPTQ_HOME="$$(mktemp -d)" dlv debug $(PKG) -- studio

clean:
	rm -rf dist
