# Backend Makefile
#
# 常用流程：
#   首次准备：make init
#   日常启动：make run
#   仅启动服务：make run-only
#   完整生成：make gen
#   提交前检查：make check
#   Linux 打包：make package

.PHONY: help init plugin cli fmt lint test check \
	gorm-gen api openapi ts ts-admin ts-uni-app ts-taro-app project-docs public-wire wire gen \
	i18n i18n-check i18n-sync i18n-locale i18n-docs i18n-openapi \
	run run-only build package docker-check docker-config docker-build docker-run docker-stop

# ===== 运行与构建参数 =====

# 服务入口、配置目录和附加运行参数。
SERVER_PACKAGE ?= ./internal/cmd/server
CONF ?= ./configs
RUN_ARGS ?=

# 二进制目标平台与输出路径，默认构建 Linux amd64 静态程序。
CGO_ENABLED ?= 0
GOOS ?= linux
GOARCH ?= amd64
BUILD_FLAGS ?=
BINARY ?= bin/server

# 压缩包输出参数，make package 会打包二进制和 configs 目录。
DIST_DIR ?= dist
PACKAGE_NAME ?= backend-$(GOOS)-$(GOARCH)
ARCHIVE ?= $(DIST_DIR)/$(PACKAGE_NAME).tar.gz

# Docker 镜像参数。
DOCKER ?= docker
DOCKERFILE ?= Dockerfile
DOCKER_PLATFORM ?= linux/$(GOARCH)
IMAGE ?= backend
TAG ?= latest
DOCKER_BUILD_ARGS ?=
CONTAINER_NAME ?= kratos-admin
DOCKER_NETWORK ?= bridge
DOCKER_HTTP_PORT ?= 7001
DOCKER_GRPC_PORT ?= 6001
DOCKER_DATA_DIR ?= $(CURDIR)/data
DOCKER_CONFIG_SOURCE_DIR ?= $(CURDIR)/configs
DOCKER_CONFIG_DIR ?= $(CURDIR)/runtime/configs
DOCKER_RUN_ARGS ?=

# ===== 工具路径参数 =====

PYTHON ?= python3
GO_BIN_DIR ?= $(or $(shell go env GOBIN),$(shell go env GOPATH)/bin)
PUBLIC_WIRE_DIR ?= internal/module
WIRE_DIR ?= internal/cmd/server
TS_PROTO_BIN ?= $(or \
	$(shell command -v protoc-gen-ts_proto 2>/dev/null), \
	$(wildcard $(CURDIR)/../frontend/admin/node_modules/.bin/protoc-gen-ts_proto), \
	$(wildcard $(CURDIR)/../frontend/uni-app/node_modules/.bin/protoc-gen-ts_proto), \
	$(wildcard $(CURDIR)/../frontend/taro-app/node_modules/.bin/protoc-gen-ts_proto), \
	$(lastword $(wildcard $(HOME)/.nvm/versions/node/*/bin/protoc-gen-ts_proto)) \
)
TS_PROTO_BIN_DIR := $(dir $(TS_PROTO_BIN))

# ===== GORM 生成参数 =====

GORM_GEN_BIN ?= $(or \
	$(shell command -v gorm-gen 2>/dev/null), \
	$(GO_BIN_DIR)/gorm-gen \
)
GORM_GEN_CONFIG ?= configs/data_local.yaml
GORM_GEN_DATABASE ?=
# 仅生成本仓库使用的表；新增业务表时同步补充此列表。
GORM_TABLE ?= ai_message,ai_session,base_api,base_api_i18n,base_area,base_config,base_dept,base_dict,base_dict_item,base_i18n,base_job,base_job_log,base_language,base_log,base_menu,base_migration,base_post,base_role,base_tenant,base_third_account,base_user,casbin_rule,code_gen_column,code_gen_proto,code_gen_table

# ===== 文档与国际化参数 =====

PROJECT_DOCS_PYTHON ?= $(PYTHON)
PROJECT_DOCS_SCRIPT ?= $(CURDIR)/../scripts/project_docs.py
I18N_OFFLINE ?= 0
I18N_MIGRATION_VERSION ?=
I18N_LOCALE ?=
I18N_SOURCE_LOCALE ?= zh-CN
I18N_LOCALES ?= en-US,zh-TW,ja-JP
I18N_AUTO_TRANSLATE ?= 1

# ===== OpenAPI 参数 =====

OPENAPI_INPUT ?= internal/openapi/assets/openapi.yaml
OPENAPI_OUTPUT_DIR ?= internal/openapi/assets
OPENAPI_I18N_CONTENT ?= internal/i18n/assets ../frontend/admin/packages/core/src/locales

# ===== 环境初始化 =====

# 按顺序安装 protoc 插件和开发命令行工具
init:
	@$(MAKE) plugin
	@$(MAKE) cli

# 安装 protoc 及项目自定义生成插件
plugin:
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

# 安装日常开发、检查和代码生成命令行工具
cli:
	@go install github.com/go-kratos/kratos/cmd/kratos/v3@latest
	@go install github.com/google/gnostic@latest
	@go install github.com/bufbuild/buf/cmd/buf@latest
	@go install entgo.io/ent/cmd/ent@latest
	@go install golang.org/x/tools/gopls@latest
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go install golang.org/x/tools/cmd/goimports@latest
	@go install honnef.co/go/tools/cmd/staticcheck@latest
	@if [ -d ../../gorm-kit/gen/cmd/gorm-gen ]; then \
		cd ../../gorm-kit/gen && go install ./cmd/gorm-gen; \
	else \
		go install github.com/liujitcn/gorm-kit/gen/cmd/gorm-gen@latest; \
	fi

# ===== 格式化与检查 =====

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

# 运行 Backend 全量 Go 测试
test:
	@echo "==> 运行 Go 测试"
	@go test ./...
	@echo "==> Go 测试完成"

# 只检查语言包、语言集合和已提交的注册生成物
i18n-check:
	@$(PYTHON) ../scripts/sync_locales.py

# 按顺序执行静态检查、Go 测试和国际化一致性检查
check:
	@$(MAKE) lint
	@$(MAKE) test
	@$(MAKE) i18n-check

# ===== 代码与文档生成 =====

# 根据数据库表生成 GORM 模型、查询和数据访问代码
gorm-gen:
	@echo "==> 生成 GORM 代码"
	@test -x "$(GORM_GEN_BIN)" || (echo "未找到 gorm-gen，请先执行 make cli" && exit 1)
	@$(GORM_GEN_BIN) -base_path=internal/data/gen -config='$(GORM_GEN_CONFIG)' $(if $(strip $(GORM_GEN_DATABASE)),-database='$(GORM_GEN_DATABASE)') $(if $(strip $(GORM_TABLE)),-table='$(GORM_TABLE)')
	@PATH="$(GO_BIN_DIR):$$PATH" find internal/data/gen -type f -name '*.go' -exec goimports -w {} +
	@echo "==> GORM 代码生成完成"

# 根据 Proto 契约生成 Backend protobuf Go 代码
api:
	@echo "==> 生成 protobuf Go 代码"
	@cd api && \
		PATH="$(GO_BIN_DIR):$$PATH" buf generate --template buf.gen.yaml
	@PATH="$(GO_BIN_DIR):$$PATH" find api/gen/go -type f -name '*.go' -exec goimports -w {} +
	@find api/gen/go -type f -name '*.go' -exec perl -pi -e 's/BaseI18N/BaseI18n/g; s/baseI18N/baseI18n/g' {} +
	@echo "==> protobuf Go 代码生成完成"

# 根据 Proto 契约生成 OpenAPI v3 及多语言文档
openapi:
	@echo "==> 生成 OpenAPI v3 文档"
	@cd api && \
		PATH="$(GO_BIN_DIR):$$PATH" buf generate --template buf.openapi.gen.yaml
	@$(PYTHON) ../scripts/generate_openapi_locales.py \
		--input "$(OPENAPI_INPUT)" \
		--output-dir "$(OPENAPI_OUTPUT_DIR)" \
		--source-locale "$(I18N_SOURCE_LOCALE)" \
		--locale "$(I18N_LOCALES)" \
		$(foreach content,$(OPENAPI_I18N_CONTENT),--i18n-content "$(content)") \
		$(if $(filter 1 true,$(I18N_AUTO_TRANSLATE)),--auto-translate,) \
		$(if $(filter 1 true,$(I18N_OFFLINE)),--offline,)
	@find "$(OPENAPI_OUTPUT_DIR)" -type f -name '*.yaml' -exec perl -pi -e 's/BaseI18N/BaseI18n/g; s/baseI18N/baseI18n/g' {} +
	@echo "==> OpenAPI v3 文档及国际化文档生成完成"

# 生成管理端、uni-app 和 Taro 的全部 TypeScript RPC 代码
ts: ts-admin ts-uni-app ts-taro-app
	@echo "==> 全部 TypeScript RPC 代码生成完成"

# 生成管理端 TypeScript RPC 代码
ts-admin:
	@echo "==> 生成管理端 TypeScript RPC 代码"
	@test -n "$(TS_PROTO_BIN)" || (echo "未找到 protoc-gen-ts_proto，请先在 frontend/admin 执行 pnpm install，或全局安装 ts-proto" && exit 1)
	@cd api && \
		PATH="$(TS_PROTO_BIN_DIR):$(GO_BIN_DIR):$$PATH" buf generate --template buf.admin.typescript.gen.yaml --exclude-path proto/system/app
	@cd api && \
		PATH="$(TS_PROTO_BIN_DIR):$(GO_BIN_DIR):$$PATH" buf generate --template buf.admin.core.typescript.gen.yaml --exclude-path proto/system/app
	@echo "==> 管理端 TypeScript RPC 代码生成完成"

# 生成 uni-app 端 TypeScript RPC 代码
ts-uni-app:
	@echo "==> 生成 uni-app TypeScript RPC 代码"
	@test -n "$(TS_PROTO_BIN)" || (echo "未找到 protoc-gen-ts_proto，请先在 frontend/uni-app 执行 pnpm install，或全局安装 ts-proto" && exit 1)
	@cd api && \
		PATH="$(TS_PROTO_BIN_DIR):$(GO_BIN_DIR):$$PATH" buf generate --template buf.uni-app.typescript.gen.yaml --exclude-path proto/system/admin
	@cd api && \
		PATH="$(TS_PROTO_BIN_DIR):$(GO_BIN_DIR):$$PATH" buf generate --template buf.uni-app.core.typescript.gen.yaml --exclude-path proto/system/admin
	@echo "==> uni-app TypeScript RPC 代码生成完成"

# 生成 Taro React 端 TypeScript RPC 代码
ts-taro-app:
	@echo "==> 生成 Taro React TypeScript RPC 代码"
	@test -n "$(TS_PROTO_BIN)" || (echo "未找到 protoc-gen-ts_proto，请先在 frontend/taro-app 执行 pnpm install，或全局安装 ts-proto" && exit 1)
	@cd api && \
		PATH="$(TS_PROTO_BIN_DIR):$(GO_BIN_DIR):$$PATH" buf generate --template buf.taro-app.typescript.gen.yaml --exclude-path proto/system/admin
	@cd api && \
		PATH="$(TS_PROTO_BIN_DIR):$(GO_BIN_DIR):$$PATH" buf generate --template buf.taro-app.core.typescript.gen.yaml --exclude-path proto/system/admin
	@echo "==> Taro React TypeScript RPC 代码生成完成"

# 收集 Markdown 文档并生成各语言文档目录
project-docs:
	@echo "==> 收集并本地化项目文档"
	@if [ ! -f "$(PROJECT_DOCS_SCRIPT)" ]; then \
		echo "未找到项目文档脚本: $(PROJECT_DOCS_SCRIPT)"; \
		echo "请检查仓库内 scripts/project_docs.py，或设置 PROJECT_DOCS_SCRIPT 指向其他脚本"; \
		exit 1; \
	fi
	@"$(PROJECT_DOCS_PYTHON)" "$(PROJECT_DOCS_SCRIPT)" \
		--root "$(CURDIR)/.." \
		--output "$(CURDIR)/internal/docs" \
		--source-locale "$(I18N_SOURCE_LOCALE)" \
		--locales "$(I18N_LOCALES)" \
		$(if $(filter 1 true,$(I18N_OFFLINE)),--offline,)
	@echo "==> 项目文档收集与本地化完成"

# 生成 Backend 公共入口使用的内部 Wire 依赖注入代码
public-wire:
	@echo "==> 生成 Backend 公共入口使用的内部 Wire 依赖注入代码"
	@cd "$(PUBLIC_WIRE_DIR)" && PATH="$(GO_BIN_DIR):$$PATH" wire . && \
		PATH="$(GO_BIN_DIR):$$PATH" goimports -w wire.go wire_gen.go && \
		echo "==> Backend 公共入口使用的内部 Wire 依赖注入代码生成完成"

# 生成独立服务入口 Wire 依赖注入代码
wire:
	@if [ ! -f "$(WIRE_DIR)/wire.go" ]; then \
		echo "==> 未找到 $(WIRE_DIR)/wire.go，跳过 Wire 依赖注入代码生成"; \
	else \
		echo "==> 生成独立服务入口 Wire 依赖注入代码"; \
		cd "$(WIRE_DIR)" && PATH="$(GO_BIN_DIR):$$PATH" wire . && \
		PATH="$(GO_BIN_DIR):$$PATH" goimports -w wire.go wire_gen.go && \
		echo "==> 独立服务入口 Wire 依赖注入代码生成完成"; \
	fi

# 按 GORM、接口、前端、文档、Wire、格式化的顺序生成全部产物
gen:
	@$(MAKE) gorm-gen
	@$(MAKE) api
	@$(MAKE) openapi
	@$(MAKE) ts
	@$(MAKE) project-docs
	@$(MAKE) public-wire
	@$(MAKE) wire
	@$(MAKE) fmt
	@echo "==> 全部代码与文档产物生成完成"

# ===== 国际化 =====

# 同步语言包集合、前端注册文件和代码生成语言目录
i18n-sync:
	@$(PYTHON) ../scripts/sync_locales.py --write $(if $(strip $(I18N_MIGRATION_VERSION)),--migration-version "$(I18N_MIGRATION_VERSION)",)

# 生成指定语言的固定语言包和动态翻译 SQL
i18n-locale:
	@test -n "$(I18N_LOCALE)" || (echo "请指定 I18N_LOCALE，例如 make i18n-locale I18N_LOCALE=ja-JP" && exit 1)
	@$(PYTHON) ../scripts/generate_locale_drafts.py \
		--write \
		--machine \
		--locale "$(I18N_LOCALE)" \
		$(if $(strip $(I18N_MIGRATION_VERSION)),--migration-version "$(I18N_MIGRATION_VERSION)",) \
		$(if $(filter 1 true,$(I18N_OFFLINE)),--offline,)

# 收集并本地化项目 Markdown 文档
i18n-docs: project-docs

# 生成 OpenAPI 多语言 YAML
i18n-openapi: openapi

# 按语言包、项目文档、OpenAPI 的顺序执行常规国际化生成
i18n:
	@$(MAKE) i18n-sync
	@$(MAKE) i18n-docs
	@$(MAKE) i18n-openapi
	@echo "==> 国际化产物生成完成"

# ===== 运行、构建与打包 =====

# 直接启动服务，不执行代码或文档生成
run-only:
	@go run "$(SERVER_PACKAGE)" --conf "$(CONF)" $(RUN_ARGS)

# 刷新启动所需产物后运行服务
run:
	@$(MAKE) project-docs
	@$(MAKE) api
	@$(MAKE) openapi
	@$(MAKE) wire
	@$(MAKE) run-only

# 刷新文档和 Wire 后构建目标平台可执行文件
build:
	@$(MAKE) project-docs
	@$(MAKE) wire
	@mkdir -p "$(dir $(BINARY))"
	@CGO_ENABLED="$(CGO_ENABLED)" GOOS="$(GOOS)" GOARCH="$(GOARCH)" go build $(BUILD_FLAGS) -o "$(BINARY)" "$(SERVER_PACKAGE)"
	@echo "==> 可执行文件已生成: $(BINARY) ($(GOOS)/$(GOARCH))"

# 构建并打包二进制与运行配置
package:
	@$(MAKE) build
	@mkdir -p "$(dir $(ARCHIVE))"
	@tar -czf "$(ARCHIVE)" "$(BINARY)" configs
	@echo "==> 发布压缩包已生成: $(ARCHIVE)"

# 检查 Docker 命令和服务端是否可用
docker-check:
	@command -v "$(DOCKER)" >/dev/null 2>&1 || (echo "未找到 Docker 命令: $(DOCKER)" && exit 1)
	@"$(DOCKER)" info >/dev/null 2>&1 || (echo "Docker 服务不可用，请确认 Docker Desktop 或 Docker daemon 已启动" && exit 1)
	@echo "==> Docker 可用: $$("$(DOCKER)" version --format 'client={{.Client.Version}} server={{.Server.Version}}')"

# 初始化可由宿主机修改的容器运行配置
docker-config:
	@test -d "$(DOCKER_CONFIG_SOURCE_DIR)" || (echo "未找到源配置目录: $(DOCKER_CONFIG_SOURCE_DIR)" && exit 1)
	@if [ ! -d "$(DOCKER_CONFIG_DIR)" ]; then \
		mkdir -p "$(DOCKER_CONFIG_DIR)"; \
		cp "$(DOCKER_CONFIG_SOURCE_DIR)"/*.yaml "$(DOCKER_CONFIG_DIR)/"; \
		for config_file in data.yaml data_local.yaml logger.yaml pprof.yaml registry.yaml; do \
			if [ -f "$(DOCKER_CONFIG_DIR)/$$config_file" ]; then \
				perl -pi -e 's/127\.0\.0\.1/host.docker.internal/g; s/localhost/host.docker.internal/g' "$(DOCKER_CONFIG_DIR)/$$config_file"; \
			fi; \
		done; \
		echo "==> Docker 运行配置已初始化: $(DOCKER_CONFIG_DIR)"; \
	fi

# 构建三端静态资源、后端程序和 Docker 镜像
docker-build: docker-check
	@test -f "$(DOCKERFILE)" || (echo "未找到 Dockerfile: $(DOCKERFILE)，请通过 DOCKERFILE 指定有效文件" && exit 1)
	@$(MAKE) -C ../frontend build-all
	@$(MAKE) build GOOS=linux GOARCH="$(GOARCH)"
	@"$(DOCKER)" build $(DOCKER_BUILD_ARGS) --platform "$(DOCKER_PLATFORM)" -f "$(DOCKERFILE)" -t "$(IMAGE):$(TAG)" .
	@echo "==> Docker 镜像已生成: $(IMAGE):$(TAG)"

# 使用宿主机数据和配置目录启动容器
docker-run: docker-check docker-config
	@"$(DOCKER)" image inspect "$(IMAGE):$(TAG)" >/dev/null 2>&1 || (echo "未找到 Docker 镜像: $(IMAGE):$(TAG)，请先执行 make docker-build" && exit 1)
	@test -d "$(DOCKER_CONFIG_DIR)" || (echo "未找到宿主机配置目录: $(DOCKER_CONFIG_DIR)" && exit 1)
	@mkdir -p "$(DOCKER_DATA_DIR)"
	@if "$(DOCKER)" container inspect "$(CONTAINER_NAME)" >/dev/null 2>&1; then \
		container_status=$$("$(DOCKER)" container inspect -f '{{.State.Status}}' "$(CONTAINER_NAME)"); \
		if [ "$$container_status" = "running" ]; then \
			echo "容器正在运行: $(CONTAINER_NAME)，请先执行 make docker-stop"; \
			exit 1; \
		fi; \
		"$(DOCKER)" container rm "$(CONTAINER_NAME)" >/dev/null; \
		echo "==> 已移除已停止容器: $(CONTAINER_NAME)"; \
	fi
	@"$(DOCKER)" run -d \
		--name "$(CONTAINER_NAME)" \
		--restart unless-stopped \
		--network "$(DOCKER_NETWORK)" \
		--add-host "host.docker.internal:host-gateway" \
		-p "$(DOCKER_HTTP_PORT):7001" \
		-p "$(DOCKER_GRPC_PORT):6001" \
		-v "$(abspath $(DOCKER_DATA_DIR)):/app/data" \
		-v "$(abspath $(DOCKER_CONFIG_DIR)):/app/configs:ro" \
		$(DOCKER_RUN_ARGS) \
		"$(IMAGE):$(TAG)"
	@echo "==> Docker 容器已启动: $(CONTAINER_NAME)"
	@echo "==> HTTP: http://127.0.0.1:$(DOCKER_HTTP_PORT)"
	@echo "==> gRPC: 127.0.0.1:$(DOCKER_GRPC_PORT)"

# 停止本地 Docker 容器并保留容器数据
docker-stop: docker-check
	@if ! "$(DOCKER)" container inspect "$(CONTAINER_NAME)" >/dev/null 2>&1; then \
		echo "容器不存在: $(CONTAINER_NAME)"; \
	elif [ "$$("$(DOCKER)" container inspect -f '{{.State.Status}}' "$(CONTAINER_NAME)")" != "running" ]; then \
		echo "容器已停止: $(CONTAINER_NAME)"; \
	else \
		"$(DOCKER)" stop "$(CONTAINER_NAME)"; \
		echo "==> Docker 容器已停止: $(CONTAINER_NAME)"; \
	fi

# ===== 帮助 =====

# 查看常用流程、全部目标及可覆盖参数
help:
	@echo ""
	@echo "常用流程:"
	@echo "  make init                         首次安装生成与检查工具"
	@echo "  make run                          刷新必要产物后启动服务"
	@echo "  make run-only                     已有生成产物时直接启动服务"
	@echo "  make gen                          完整生成代码和文档"
	@echo "  make check                        提交前执行 lint、测试和国际化检查"
	@echo "  make package                      构建 Linux amd64 发布压缩包"
	@echo ""
	@echo "可用目标:"
	@awk '/^[a-zA-Z\-_0-9]+:/ { \
	helpMessage = match(lastLine, /^# (.*)/); \
		if (helpMessage) { \
			helpCommand = substr($$1, 0, index($$1, ":")-1); \
			helpMessage = substr(lastLine, RSTART + 2, RLENGTH); \
			printf "\033[36m  %-20s\033[0m %s\n", helpCommand, helpMessage; \
		} \
	} \
	{ lastLine = $$0 }' $(MAKEFILE_LIST)
	@echo ""
	@echo "常用参数（命令行使用 参数=值 覆盖）:"
	@printf "  %-20s %s\n" "CONF" "运行配置目录，当前: $(CONF)"
	@printf "  %-20s %s\n" "GOOS / GOARCH" "目标平台，当前: $(GOOS) / $(GOARCH)"
	@printf "  %-20s %s\n" "BINARY" "二进制输出，当前: $(BINARY)"
	@printf "  %-20s %s\n" "ARCHIVE" "压缩包输出，当前: $(ARCHIVE)"
	@printf "  %-20s %s\n" "PUBLIC_WIRE_DIR" "公共入口内部 Wire 目录，当前: $(PUBLIC_WIRE_DIR)"
	@printf "  %-20s %s\n" "WIRE_DIR" "独立入口 Wire 目录，当前: $(WIRE_DIR)"
	@printf "  %-20s %s\n" "GORM_GEN_CONFIG" "GORM 数据源配置，当前: $(GORM_GEN_CONFIG)"
	@printf "  %-20s %s\n" "GORM_GEN_DATABASE" "可选数据库名，默认使用配置文件"
	@printf "  %-20s %s\n" "GORM_TABLE" "可选逗号分隔表清单"
	@printf "  %-20s %s\n" "I18N_LOCALES" "目标语言，当前: $(I18N_LOCALES)"
	@printf "  %-20s %s\n" "I18N_OFFLINE" "设为 1 时离线生成"
	@printf "  %-20s %s\n" "IMAGE / TAG" "Docker 镜像，当前: $(IMAGE):$(TAG)"
	@printf "  %-20s %s\n" "DOCKER_PLATFORM" "Docker 平台，当前: $(DOCKER_PLATFORM)"
	@printf "  %-20s %s\n" "CONTAINER_NAME" "Docker 容器名，当前: $(CONTAINER_NAME)"
	@printf "  %-20s %s\n" "DOCKER_NETWORK" "Docker 网络，当前: $(DOCKER_NETWORK)"
	@printf "  %-20s %s\n" "DOCKER_HTTP_PORT" "宿主机 HTTP 端口，当前: $(DOCKER_HTTP_PORT)"
	@printf "  %-20s %s\n" "DOCKER_GRPC_PORT" "宿主机 gRPC 端口，当前: $(DOCKER_GRPC_PORT)"
	@printf "  %-20s %s\n" "DOCKER_DATA_DIR" "宿主机数据目录，当前: $(DOCKER_DATA_DIR)"
	@printf "  %-20s %s\n" "DOCKER_CONFIG_SOURCE_DIR" "Docker 配置源目录，当前: $(DOCKER_CONFIG_SOURCE_DIR)"
	@printf "  %-20s %s\n" "DOCKER_CONFIG_DIR" "宿主机配置目录，当前: $(DOCKER_CONFIG_DIR)"
	@echo ""
	@echo "示例:"
	@echo "  make run-only CONF=./configs RUN_ARGS='--help'"
	@echo "  make build GOOS=darwin GOARCH=arm64 BINARY=bin/server-darwin-arm64"
	@echo "  make package GOOS=linux GOARCH=arm64"
	@echo "  make docker-build IMAGE=kratos-admin TAG=v1.0.0"
	@echo "  make docker-build GOARCH=arm64 DOCKER_PLATFORM=linux/arm64"
	@echo "  make docker-run IMAGE=kratos-admin TAG=v1.0.0"
	@echo "  make docker-stop IMAGE=kratos-admin TAG=v1.0.0"

.DEFAULT_GOAL := help
