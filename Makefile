all:
	@go generate ./...
	@go mod tidy
	@go test -cover ./...
	@[ -d cmd ] && go build -ldflags "-w" -o bin/ ./cmd/...
	@go test -tags integration -c -o bin/test || true

audit:
	@which golangci-lint >/dev/null || (echo "Cannot run linters. Have you installed golangci-lint?" && false)
	@golangci-lint run
