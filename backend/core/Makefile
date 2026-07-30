.PHONY: fmt lint test vet

GO_BIN_DIR ?= $(shell GOBIN=$$(go env GOBIN); if [ -n "$$GOBIN" ]; then printf '%s' "$$GOBIN"; else printf '%s/bin' "$$(go env GOPATH)"; fi)

# 格式化 Go 源码。
fmt:
	@PATH="$(GO_BIN_DIR):$$PATH" find . -type f -name '*.go' -exec goimports -w {} +

# 运行 Core 静态检查。
lint: vet

# 验证核心模块。
test:
	@go test ./...

# 静态检查核心模块。
vet:
	@go vet ./...
