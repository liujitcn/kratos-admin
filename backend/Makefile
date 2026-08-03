# Makefile 常用目标：初始化工具、生成代码、格式化代码、启动服务
.PHONY: help init plugin cli fmt lint i18n-sync i18n-check i18n-draft i18n-locales i18n-zh-tw gorm-gen wire api openapi ts ts-app ts-taro-app project-docs gen run build docker-build

IMAGE ?= backend
TAG ?= latest
DOCKERFILE ?= Dockerfile
BINARY ?= bin/server
PROJECT_DOCS_BIN ?= project-docs
GO_BIN_DIR ?= $(or $(shell go env GOBIN),$(shell go env GOPATH)/bin)
TS_PROTO_BIN ?= $(or \
	$(shell command -v protoc-gen-ts_proto 2>/dev/null), \
	$(wildcard $(CURDIR)/../frontend/admin/node_modules/.bin/protoc-gen-ts_proto), \
	$(wildcard $(CURDIR)/../frontend/uni-app/node_modules/.bin/protoc-gen-ts_proto), \
	$(wildcard $(CURDIR)/../frontend/taro-app/node_modules/.bin/protoc-gen-ts_proto), \
	$(lastword $(wildcard $(HOME)/.nvm/versions/node/*/bin/protoc-gen-ts_proto)) \
)
TS_PROTO_BIN_DIR := $(dir $(TS_PROTO_BIN))
GORM_GEN_BIN ?= $(or \
	$(shell command -v gorm-gen 2>/dev/null), \
	$(GO_BIN_DIR)/gorm-gen \
)
GORM_GEN_CONFIG ?= configs/data_local.yaml
GORM_GEN_DATABASE ?=
GORM_TABLE ?=
I18N_WRITE ?= 0
I18N_OFFLINE ?= 0
I18N_MIGRATION_VERSION ?=
I18N_LOCALE ?=

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
	@if [ -d ../../kratos-kit/cmd/project-docs ]; then \
		cd ../../kratos-kit/cmd/project-docs && go install .; \
	else \
		go install github.com/liujitcn/kratos-kit/cmd/project-docs@latest; \
	fi
	@if [ -d ../../gorm-kit/gen/cmd/gorm-gen ]; then \
		cd ../../gorm-kit/gen && go install ./cmd/gorm-gen; \
	else \
		go install github.com/liujitcn/gorm-kit/gen/cmd/gorm-gen@latest; \
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

# 同步语言包清单和前端注册产物
i18n-sync:
	@python3 ../scripts/sync_locales.py --write $(if $(strip $(I18N_MIGRATION_VERSION)),--migration-version "$(I18N_MIGRATION_VERSION)",)

# 检查 Proto 消息键和后端语言目录完整性
i18n-check:
	@go run ./internal/cmd/i18n -mode check

# 报告错误目录缺失草稿；I18N_WRITE=1 时使用 Google V1 只补缺失项
i18n-draft:
	@go run ./internal/cmd/i18n -mode draft $(if $(filter 1 true,$(I18N_WRITE)),-write)

# 使用 OpenCC 和 Google V1 生成新增语言的语言包和迁移数据
i18n-locales:
	@python3 ../scripts/generate_zh_tw.py --write --machine $(if $(strip $(I18N_LOCALE)),--locale "$(I18N_LOCALE)",) $(if $(strip $(I18N_MIGRATION_VERSION)),--migration-version "$(I18N_MIGRATION_VERSION)",) $(if $(filter 1 true,$(I18N_OFFLINE)),--offline)

# 保留繁体中文命令别名
i18n-zh-tw: i18n-locales

# 生成 GORM 代码
gorm-gen:
	@echo "==> 生成 GORM 代码"
	@test -x "$(GORM_GEN_BIN)" || (echo "gorm-gen not found. Run make cli first." && exit 1)
	$(GORM_GEN_BIN) -base_path=internal/data/gen -config='$(GORM_GEN_CONFIG)' $(if $(strip $(GORM_GEN_DATABASE)),-database='$(GORM_GEN_DATABASE)') $(if $(strip $(GORM_TABLE)),-table='$(GORM_TABLE)')
	@echo "==> GORM 代码生成完成"

# 生成 wire 依赖注入代码
wire: project-docs
	@echo "==> 生成 wire 依赖注入代码"
	@PATH="$(GO_BIN_DIR):$$PATH" wire .
	@PATH="$(GO_BIN_DIR):$$PATH" goimports -w wire.go wire_gen.go
	@echo "==> wire 依赖注入代码生成完成"

# 生成 Backend protobuf Go 代码
api:
	@echo "==> 生成 protobuf Go 代码"
	@cd api && \
		PATH="$(GO_BIN_DIR):$$PATH" buf generate --template buf.gen.yaml
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
	@cd api && \
		PATH="$(TS_PROTO_BIN_DIR):$(GO_BIN_DIR):$$PATH" buf generate --template buf.admin.core.typescript.gen.yaml --exclude-path proto/system/app
	@echo "==> TypeScript RPC 代码生成完成"

# 生成 uni-app 端 TypeScript RPC 代码
ts-app:
	@echo "==> 生成 uni-app TypeScript RPC 代码"
	@test -n "$(TS_PROTO_BIN)" || (echo "protoc-gen-ts_proto not found. Run pnpm install in frontend/uni-app, or install ts-proto globally." && exit 1)
	@cd api && \
		PATH="$(TS_PROTO_BIN_DIR):$(GO_BIN_DIR):$$PATH" buf generate --template buf.app.typescript.gen.yaml --exclude-path proto/system/admin
	@cd api && \
		PATH="$(TS_PROTO_BIN_DIR):$(GO_BIN_DIR):$$PATH" buf generate --template buf.app.core.typescript.gen.yaml --exclude-path proto/system/admin
	@echo "==> uni-app TypeScript RPC 代码生成完成"

# 生成 Taro React 端 TypeScript RPC 代码
ts-taro-app:
	@echo "==> 生成 Taro React TypeScript RPC 代码"
	@test -n "$(TS_PROTO_BIN)" || (echo "protoc-gen-ts_proto not found. Run pnpm install in frontend/taro-app, or install ts-proto globally." && exit 1)
	@cd api && \
		PATH="$(TS_PROTO_BIN_DIR):$(GO_BIN_DIR):$$PATH" buf generate --template buf.taro-app.typescript.gen.yaml --exclude-path proto/system/admin
	@cd api && \
		PATH="$(TS_PROTO_BIN_DIR):$(GO_BIN_DIR):$$PATH" buf generate --template buf.taro-app.core.typescript.gen.yaml --exclude-path proto/system/admin
	@echo "==> Taro React TypeScript RPC 代码生成完成"

# 收集当前项目约定范围内的 README 与 docs Markdown
project-docs:
	@echo "==> 收集项目文档"
	@command -v "$(PROJECT_DOCS_BIN)" >/dev/null 2>&1 || (echo "project-docs not found. Run make cli first." && exit 1)
	@cd .. && "$(PROJECT_DOCS_BIN)"
	@echo "==> 项目文档收集完成"

# 一键生成全部接口与文档产物（Go 代码 + OpenAPI 文档 + TypeScript 代码）
gen: project-docs api openapi ts ts-app ts-taro-app gorm-gen wire fmt
	@echo "==> 全部接口产物生成完成"

# 启动应用
run: project-docs api openapi
	@go run ./internal/cmd/server --conf ./configs

# 构建 Linux 可执行文件，供 Docker 镜像直接打包
build: project-docs
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
