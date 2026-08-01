.PHONY: fmt lint test vet api

GO_BIN_DIR ?= $(or $(shell go env GOBIN),$(shell go env GOPATH)/bin)

# 生成 Core 通用 protobuf Go 代码。
api:
	@echo "==> 生成 Core protobuf Go 代码"
	@cd api && PATH="$(GO_BIN_DIR):$$PATH" buf generate --template buf.gen.yaml
	@PATH="$(GO_BIN_DIR):$$PATH" find api/gen/go -type f -name '*.go' -exec goimports -w {} +
	@echo "==> Core protobuf Go 代码生成完成"

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
