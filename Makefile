.PHONY: all audit .tidy generate test build be7000

all: .tidy generate build

audit:
	@which golangci-lint >/dev/null || (echo "Cannot run linters. Have you installed golangci-lint?" && false)
	@golangci-lint run

test:
	@go test -race ./...

.tidy:
	@go mod tidy

generate:
	@go generate ./...

build:
	@CGO_ENABLED=0 go build -ldflags "-w" -o bin/ ./cmd/...

be7000:
	@go mod tidy
	docker build --platform=linux/arm64 . -t bot:arm64
