all: generate tidy
	@go test -cover ./...
	@[ -d cmd ] && go build -ldflags "-w" -o bin/ ./cmd/...
	@go test -tags integration -c -o bin/test || true

audit:
	@which golangci-lint >/dev/null || (echo "Cannot run linters. Have you installed golangci-lint?" && false)
	@golangci-lint run

generate:
	@go generate ./...

tidy:
	@go mod tidy

be7000:
	docker build --build-arg GOOS=linux --build-arg GOARCH=arm64 --platform=linux/arm64 . -t bot:arm64
