# Makefile 常用目标：初始化工具、生成代码、格式化代码、启动服务
.PHONY: help init plugin cli fmt lint gorm-gen wire api openapi ts ts-app gen run build docker-build

IMAGE ?= backend
TAG ?= latest
DOCKERFILE ?= Dockerfile
BINARY ?= bin/server
GO_BIN_DIR ?= $(shell GOBIN=$$(go env GOBIN); if [ -n "$$GOBIN" ]; then printf '%s' "$$GOBIN"; else printf '%s/bin' "$$(go env GOPATH)"; fi)
TS_PROTO_BIN ?= $(or \
	$(shell command -v protoc-gen-ts_proto 2>/dev/null), \
	$(wildcard $(CURDIR)/../frontend/admin/node_modules/.bin/protoc-gen-ts_proto), \
	$(wildcard $(CURDIR)/../frontend/app/node_modules/.bin/protoc-gen-ts_proto), \
	$(lastword $(wildcard $(HOME)/.nvm/versions/node/*/bin/protoc-gen-ts_proto)) \
)
TS_PROTO_BIN_DIR := $(dir $(TS_PROTO_BIN))
GEN_BIN ?= $(or \
	$(shell command -v gen 2>/dev/null), \
	$(shell GOBIN=$$(go env GOBIN); GOPATH=$$(go env GOPATH); if [ -n "$$GOBIN" ]; then printf '%s' "$$GOBIN/gen"; else printf '%s' "$$GOPATH/bin/gen"; fi) \
)
GORM_GEN_SOURCE ?= root:112233@tcp(192.168.60.52:3306)/kratos_admin?charset=utf8&parseTime=True&loc=Local&timeout=1000ms

# 初始化开发环境
init: plugin cli

# 安装 protoc 插件
plugin:
	# Go 相关插件
	@go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	@go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	@go install github.com/go-kratos/kratos/cmd/protoc-gen-go-http/v3@latest
	@go install github.com/go-kratos/kratos/cmd/protoc-gen-go-errors/v3@latest
	@go install github.com/google/gnostic/cmd/protoc-gen-openapi@latest
	@go install github.com/envoyproxy/protoc-gen-validate@latest
	@go install github.com/liujitcn/kratos-kit/cmd/protoc-gen-go-service@latest
	@if [ -d ../../kratos-kit/cmd/protoc-gen-go-agent-tool ]; then \
		cd ../../kratos-kit/cmd/protoc-gen-go-agent-tool && go install .; \
	else \
		go install github.com/liujitcn/kratos-kit/cmd/protoc-gen-go-agent-tool@latest; \
	fi
	@if [ -d ../../kratos-kit/cmd/protoc-gen-go-mcp-tool ]; then \
		cd ../../kratos-kit/cmd/protoc-gen-go-mcp-tool && go install .; \
	else \
		go install github.com/liujitcn/kratos-kit/cmd/protoc-gen-go-mcp-tool@latest; \
	fi
	@go install github.com/go-kratos/protoc-gen-typescript-http@latest

# 安装命令行工具
cli:
	@go install github.com/go-kratos/kratos/cmd/kratos/v3@latest
	@go install github.com/google/gnostic@latest
	@go install github.com/bufbuild/buf/cmd/buf@latest
	@go install entgo.io/ent/cmd/ent@latest
	@go install golang.org/x/tools/gopls@latest
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go install golang.org/x/tools/cmd/goimports@latest
	@go install honnef.co/go/tools/cmd/staticcheck@latest
	@if [ -d ../../gorm-kit/gen ]; then \
		cd ../../gorm-kit/gen && go install .; \
	else \
		go install github.com/liujitcn/gorm-kit/gen@latest; \
	fi

# 使用 goimports 统一整理 Go 代码的 import 与格式
fmt:
	@echo "==> 格式化 Go 代码"
	@PATH="$(GO_BIN_DIR):$$PATH" find . -type f -name '*.go' -exec goimports -w {} +
	@echo "==> Go 代码格式化完成"

# 运行 golangci-lint 静态检查（含禁止手写 SQL 检查）
lint:
	@echo "==> 运行 golangci-lint 静态检查"
	@PATH="$(GO_BIN_DIR):$$PATH" golangci-lint run ./...
	@echo "==> golangci-lint 静态检查完成"

# 生成 GORM 代码
gorm-gen:
	@echo "==> 生成 GORM 代码"
	@test -x "$(GEN_BIN)" || (echo "gen not found. Run make cli first." && exit 1)
	$(GEN_BIN) -base_path=pkg/gen -source='$(GORM_GEN_SOURCE)' $(if $(strip $(GORM_TABLE)),-table='$(GORM_TABLE)')
	@echo "==> GORM 代码生成完成"

# 生成 wire 依赖注入代码
wire:
	@echo "==> 生成 wire 依赖注入代码"
	@cd internal/cmd/server && \
		PATH="$(GO_BIN_DIR):$$PATH" wire
	@PATH="$(GO_BIN_DIR):$$PATH" goimports -w internal/cmd/server/wire.go internal/cmd/server/wire_gen.go
	@echo "==> wire 依赖注入代码生成完成"

# 生成 protobuf 的 Go 代码
api:
	@echo "==> 生成 protobuf Go 代码"
	@cd api && \
		PATH="$(GO_BIN_DIR):$$PATH" buf generate
	@PATH="$(GO_BIN_DIR):$$PATH" find api/gen/go -type f -name '*.go' -exec goimports -w {} +
	@echo "==> protobuf Go 代码生成完成"

# 生成 OpenAPI v3 文档
openapi:
	@echo "==> 生成 OpenAPI v3 文档"
	@cd api && \
		PATH="$(GO_BIN_DIR):$$PATH" buf generate --template buf.openapi.gen.yaml
	@echo "==> OpenAPI v3 文档生成完成"

# 生成 TypeScript 代码
ts:
	@echo "==> 生成 TypeScript RPC 代码"
	@test -n "$(TS_PROTO_BIN)" || (echo "protoc-gen-ts_proto not found. Run pnpm install in frontend/admin, or install ts-proto globally." && exit 1)
	@cd api && \
		PATH="$(TS_PROTO_BIN_DIR):$(GO_BIN_DIR):$$PATH" buf generate --template buf.admin.typescript.gen.yaml --exclude-path proto/system/app
	@echo "==> TypeScript RPC 代码生成完成"

# 生成 app 端 TypeScript RPC 代码
ts-app:
	@echo "==> 生成 app TypeScript RPC 代码"
	@test -n "$(TS_PROTO_BIN)" || (echo "protoc-gen-ts_proto not found. Run pnpm install in frontend/app, or install ts-proto globally." && exit 1)
	@cd api && \
		PATH="$(TS_PROTO_BIN_DIR):$(GO_BIN_DIR):$$PATH" buf generate --template buf.app.typescript.gen.yaml --exclude-path proto/system/admin
	@echo "==> app TypeScript RPC 代码生成完成"

# 一键生成全部接口产物（Go 代码 + OpenAPI 文档 + TypeScript 代码）
gen: api openapi ts ts-app gorm-gen wire fmt
	@echo "==> 全部接口产物生成完成"

# 启动应用
run: api openapi
	@go run ./internal/cmd/server --conf ./configs

# 构建 Linux 可执行文件，供 Docker 镜像直接打包
build:
	@mkdir -p $(dir $(BINARY))
	@CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o $(BINARY) ./internal/cmd/server

# 构建后端服务 Docker 镜像
docker-build: build
	@docker build -f $(DOCKERFILE) -t $(IMAGE):$(TAG) .

# 显示帮助
help:
	@echo ""
	@echo "Usage:"
	@echo " make [target]"
	@echo ""
	@echo 'Targets:'
	@awk '/^[a-zA-Z\-_0-9]+:/ { \
	helpMessage = match(lastLine, /^# (.*)/); \
		if (helpMessage) { \
			helpCommand = substr($$1, 0, index($$1, ":")-1); \
			helpMessage = substr(lastLine, RSTART + 2, RLENGTH); \
			printf "\033[36m%-22s\033[0m %s\n", helpCommand,helpMessage; \
		} \
	} \
	{ lastLine = $$0 }' $(MAKEFILE_LIST)

.DEFAULT_GOAL := help
