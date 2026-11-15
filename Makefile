.PHONY: all build build-web build-go test benchmark simulate clean

all: build

build-web:
	@echo "Building React 19 FinOps studio..."
	cd web && npm install && npm run build

build-go:
	@echo "Building Go reverse proxy binary..."
	go build -o bin/cascadepilot.exe ./cmd/cascadepilot

build: build-web build-go
	@echo "Build complete: bin/cascadepilot.exe"

test:
	go test -v ./...

benchmark:
	go run ./cmd/cascadepilot benchmark

simulate:
	go run ./cmd/cascadepilot simulate --queries=examples/query_benchmark.jsonl --workers=4

clean:
	rm -rf bin/ internal/server/dist/* web/dist/ web/node_modules/ data/*.db
