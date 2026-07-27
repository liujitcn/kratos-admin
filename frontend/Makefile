# Makefile 常用目标：安装依赖、启动前端、构建前端、打包发布 npm 包、显示帮助
NPM_REGISTRY ?= https://registry.npmjs.org/
NPM_ACCESS ?= public
NPM_TAG ?= latest
NPM_PACK_DEST ?= dist/npm

.PHONY: help init pnpm-admin pnpm-app run run-admin run-app build build-admin build-app build-package package package-admin package-app publish publish-admin publish-app

# 安装管理后台依赖
init: pnpm-admin pnpm-app

# 安装管理后台依赖
pnpm-admin:
	@cd admin && \
		CI=1 pnpm install

# 安装 app 端依赖
pnpm-app:
	@cd app && \
		CI=1 pnpm install

# 启动管理后台开发环境
run: run-admin

# 启动管理后台开发服务
run-admin:
	@cd admin && \
		CI=1 pnpm run dev

# 启动 app 端微信小程序开发环境
run-app:
	@cd app && \
		CI=1 pnpm run dev:mp-weixin

# 构建管理后台产物
build: build-admin

# 构建管理后台静态资源
build-admin:
	@cd admin && \
		CI=1 pnpm run build

# 构建可供业务后台组合使用的前端模块包
build-package:
	@cd admin && \
		CI=1 pnpm run build:package

# 检查并生成管理后台 npm 包
package-admin:
	@cd admin && \
		CI=1 pnpm run type:check && \
		CI=1 pnpm run build:package && \
		CI=1 pnpm pack --pack-destination "$(NPM_PACK_DEST)"

# 检查并生成应用端 npm 包
package-app:
	@cd app && \
		CI=1 pnpm run tsc && \
		CI=1 pnpm pack --pack-destination "$(NPM_PACK_DEST)"

# 检查并生成两个 npm 包
package: package-admin package-app

# 发布管理后台 npm 包
publish-admin: package-admin
	@cd admin && \
		CI=1 pnpm publish \
			--registry "$(NPM_REGISTRY)" \
			--access "$(NPM_ACCESS)" \
			--tag "$(NPM_TAG)"

# 发布应用端 npm 包
publish-app: package-app
	@cd app && \
		CI=1 pnpm publish \
			--registry "$(NPM_REGISTRY)" \
			--access "$(NPM_ACCESS)" \
			--tag "$(NPM_TAG)"

# 检查、打包并按顺序发布两个 npm 包
publish: package
	@cd admin && \
		CI=1 pnpm publish \
			--registry "$(NPM_REGISTRY)" \
			--access "$(NPM_ACCESS)" \
			--tag "$(NPM_TAG)" && \
		cd ../app && \
		CI=1 pnpm publish \
			--registry "$(NPM_REGISTRY)" \
			--access "$(NPM_ACCESS)" \
			--tag "$(NPM_TAG)"

# 构建 app 端 H5 静态资源
build-app:
	@cd app && \
		CI=1 pnpm run build:h5

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
