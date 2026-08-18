# Makefile 常用目标：初始化工具、国际化、生成代码、格式化代码、启动服务
.PHONY: help init plugin cli fmt lint i18n i18n-sync i18n-check i18n-locale i18n-docs i18n-openapi gorm-gen wire api openapi ts ts-uni-app ts-taro-app project-docs gen run build docker-build

IMAGE ?= backend
TAG ?= latest
DOCKERFILE ?= Dockerfile
BINARY ?= bin/server
PROJECT_DOCS_BIN ?= project-docs
PROJECT_DOCS_PYTHON ?= python3
PROJECT_DOCS_SOURCE_DIR ?= $(CURDIR)/../../kratos-kit
PROJECT_DOCS_TRANSLATE_SCRIPT ?= $(PROJECT_DOCS_SOURCE_DIR)/cmd/i18n/project_docs.py
WIRE_DIR ?= internal/cmd/server
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
# 仅生成本仓库使用的表；新增业务表时同步补充此列表。
GORM_TABLE ?= ai_message,ai_session,base_api,base_api_i18n,base_area,base_config,base_dept,base_dict,base_dict_item,base_i18n,base_job,base_job_log,base_language,base_log,base_menu,base_migration,base_post,base_role,base_tenant,base_third_account,base_user,casbin_rule,code_gen_column,code_gen_proto,code_gen_table
I18N_OFFLINE ?= 0
I18N_MIGRATION_VERSION ?=
I18N_LOCALE ?=
I18N_SOURCE_LOCALE ?= zh-CN
I18N_LOCALES ?= en-US,zh-TW,ja-JP
OPENAPI_INPUT ?= internal/openapi/assets/openapi.yaml
OPENAPI_OUTPUT_DIR ?= internal/openapi/assets
OPENAPI_I18N_CONTENT ?= internal/i18n/assets ../frontend/admin/packages/core/src/locales
I18N_AUTO_TRANSLATE ?= 1

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

# 执行 backend 内的常规国际化生成链路
i18n: i18n-sync i18n-docs i18n-openapi

# 只检查语言包、语言集合和已提交的注册生成物
i18n-check:
	@python3 ../scripts/sync_locales.py

# 同步语言包集合、前端注册文件和代码生成语言目录
i18n-sync:
	@python3 ../scripts/sync_locales.py --write $(if $(strip $(I18N_MIGRATION_VERSION)),--migration-version "$(I18N_MIGRATION_VERSION)",)

# 生成指定语言的固定语言包和动态翻译 SQL
i18n-locale:
	@test -n "$(I18N_LOCALE)" || (echo "请指定 I18N_LOCALE，例如 make i18n-locale I18N_LOCALE=ja-JP" && exit 1)
	@python3 ../scripts/generate_locale_drafts.py \
		--write \
		--machine \
		--locale "$(I18N_LOCALE)" \
		$(if $(strip $(I18N_MIGRATION_VERSION)),--migration-version "$(I18N_MIGRATION_VERSION)",) \
		$(if $(filter 1 true,$(I18N_OFFLINE)),--offline,)

# 收集项目 Markdown 文档，并补充缺失的语言版本
i18n-docs: project-docs
	@if [ ! -f "$(PROJECT_DOCS_TRANSLATE_SCRIPT)" ]; then \
		echo "未找到项目文档翻译脚本: $(PROJECT_DOCS_TRANSLATE_SCRIPT)"; \
		echo "请检出 kratos-kit 源码，或设置 PROJECT_DOCS_TRANSLATE_SCRIPT 指向 cmd/i18n/project_docs.py"; \
		exit 1; \
	fi
	@"$(PROJECT_DOCS_PYTHON)" "$(PROJECT_DOCS_TRANSLATE_SCRIPT)" \
		--root "$(CURDIR)/.." \
		--output "$(CURDIR)/internal/docs" \
		--source-locale "$(I18N_SOURCE_LOCALE)" \
		--locales "$(I18N_LOCALES)" \
		$(if $(filter 1 true,$(I18N_OFFLINE)),--offline,)

# 生成 OpenAPI 多语言 YAML
i18n-openapi: openapi

# 生成 GORM 代码
gorm-gen: fmt
	@echo "==> 生成 GORM 代码"
	@test -x "$(GORM_GEN_BIN)" || (echo "gorm-gen not found. Run make cli first." && exit 1)
	$(GORM_GEN_BIN) -base_path=internal/data/gen -config='$(GORM_GEN_CONFIG)' $(if $(strip $(GORM_GEN_DATABASE)),-database='$(GORM_GEN_DATABASE)') $(if $(strip $(GORM_TABLE)),-table='$(GORM_TABLE)')
	# gorm/gen 会将 i18n 转为 I18N，项目现有生成代码统一使用 I18n。
	@for file in internal/data/gen/models/base_i18n.gen.go internal/data/gen/models/base_api_i18n.gen.go internal/data/gen/models/table_comment.gen.go internal/data/gen/query/base_i18n.gen.go internal/data/gen/query/base_api_i18n.gen.go internal/data/gen/query/gen.go internal/data/gen/data/base_i18n.go internal/data/gen/data/base_api_i18n.go internal/data/gen/data/data.go internal/data/gen/data/init.go; do \
		perl -pi -e 's/BaseAPII18N/BaseAPII18n/g; s/baseAPII18N/baseAPII18n/g; s/BaseI18N/BaseI18n/g; s/baseI18N/baseI18n/g' "$$file"; \
	done
	@echo "==> GORM 代码生成完成"

# 生成 wire 依赖注入代码
wire:
	@if [ ! -f "$(WIRE_DIR)/wire.go" ]; then \
		echo "==> 未找到 $(WIRE_DIR)/wire.go，跳过 wire 依赖注入代码生成"; \
	else \
		echo "==> 生成 wire 依赖注入代码"; \
		cd "$(WIRE_DIR)" && PATH="$(GO_BIN_DIR):$$PATH" wire . && \
		PATH="$(GO_BIN_DIR):$$PATH" goimports -w wire.go wire_gen.go && \
		echo "==> wire 依赖注入代码生成完成"; \
	fi

# 生成 Backend protobuf Go 代码
api:
	@echo "==> 生成 protobuf Go 代码"
	@cd api && \
		PATH="$(GO_BIN_DIR):$$PATH" buf generate --template buf.gen.yaml
	@PATH="$(GO_BIN_DIR):$$PATH" find api/gen/go -type f -name '*.go' -exec goimports -w {} +
	@find api/gen/go -type f -name '*.go' -exec perl -pi -e 's/BaseI18N/BaseI18n/g; s/baseI18N/baseI18n/g' {} +
	@echo "==> protobuf Go 代码生成完成"

# 生成 OpenAPI v3 文档
openapi:
	@echo "==> 生成 OpenAPI v3 文档"
	@cd api && \
		PATH="$(GO_BIN_DIR):$$PATH" buf generate --template buf.openapi.gen.yaml
	@python3 ../scripts/generate_openapi_locales.py \
		--input "$(OPENAPI_INPUT)" \
		--output-dir "$(OPENAPI_OUTPUT_DIR)" \
		--source-locale "$(I18N_SOURCE_LOCALE)" \
		--locale "$(I18N_LOCALES)" \
		$(foreach content,$(OPENAPI_I18N_CONTENT),--i18n-content "$(content)") \
		$(if $(filter 1 true,$(I18N_AUTO_TRANSLATE)),--auto-translate,) \
		$(if $(filter 1 true,$(I18N_OFFLINE)),--offline,)
	@find "$(OPENAPI_OUTPUT_DIR)" -type f -name '*.yaml' -exec perl -pi -e 's/BaseI18N/BaseI18n/g; s/baseI18N/baseI18n/g' {} +
	@echo "==> OpenAPI v3 文档及国际化文档生成完成"

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
ts-uni-app:
	@echo "==> 生成 uni-app TypeScript RPC 代码"
	@test -n "$(TS_PROTO_BIN)" || (echo "protoc-gen-ts_proto not found. Run pnpm install in frontend/uni-app, or install ts-proto globally." && exit 1)
	@cd api && \
		PATH="$(TS_PROTO_BIN_DIR):$(GO_BIN_DIR):$$PATH" buf generate --template buf.uni-app.typescript.gen.yaml --exclude-path proto/system/admin
	@cd api && \
		PATH="$(TS_PROTO_BIN_DIR):$(GO_BIN_DIR):$$PATH" buf generate --template buf.uni-app.core.typescript.gen.yaml --exclude-path proto/system/admin
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

# 只收集当前项目约定范围内的 README 与 docs Markdown
project-docs:
	@echo "==> 收集项目文档"
	@if [ -d "$(PROJECT_DOCS_SOURCE_DIR)/cmd/project-docs" ]; then \
		cd "$(PROJECT_DOCS_SOURCE_DIR)/cmd/project-docs" && go run . --root "$(CURDIR)/.." --output "$(CURDIR)/internal/docs"; \
	elif command -v "$(PROJECT_DOCS_BIN)" >/dev/null 2>&1; then \
		cd "$(CURDIR)/.." && "$(PROJECT_DOCS_BIN)" --output "$(CURDIR)/internal/docs"; \
	else \
		echo "project-docs not found. Run make cli first or set PROJECT_DOCS_BIN."; \
		exit 1; \
	fi
	@echo "==> 项目文档收集完成"

# 一键生成全部接口与文档产物（Go 代码 + OpenAPI 文档 + TypeScript 代码）
gen: project-docs api openapi ts ts-uni-app ts-taro-app gorm-gen wire fmt
	@echo "==> 全部接口产物生成完成"

# 启动应用
run: project-docs api openapi wire
	@go run ./internal/cmd/server --conf ./configs

# 构建 Linux 可执行文件，供 Docker 镜像直接打包
build: project-docs wire
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
