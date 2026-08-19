# Frontend Makefile
#
# 常用流程：
#   安装全部依赖：make init
#   启动管理后台：make run
#   检查全部前端：make check
#   构建全部宿主：make build-all
#   生成全部 npm 包：make package

.PHONY: help init reinstall pnpm-admin pnpm-uni-app pnpm-taro-app \
	run run-admin run-uni-app run-taro-app \
	check check-admin check-uni-app check-taro-app \
	build build-all build-admin build-uni-app build-taro-app build-package \
	package package-admin package-uni-app package-taro-app \
	publish publish-only publish-admin publish-uni-app publish-taro-app

# ===== 执行参数 =====

PNPM ?= pnpm
CI ?= 1

# ===== npm 发布参数 =====

NPM_REGISTRY ?= https://registry.npmjs.org/
NPM_ACCESS ?= public
NPM_TAG ?= latest
NPM_PACK_DEST ?= dist/npm
NPM_SKIP_GIT_CHECKS ?= true

NPM_PUBLISH_ENV := NPM_REGISTRY="$(NPM_REGISTRY)" NPM_ACCESS="$(NPM_ACCESS)" NPM_TAG="$(NPM_TAG)" NPM_SKIP_GIT_CHECKS="$(NPM_SKIP_GIT_CHECKS)"

# ===== 依赖安装 =====

# 按管理后台、uni-app、Taro 的顺序安装全部前端依赖
init:
	@$(MAKE) pnpm-admin
	@$(MAKE) pnpm-uni-app
	@$(MAKE) pnpm-taro-app

# 删除全部前端 node_modules 并重新安装依赖
reinstall:
	@bash scripts/reinstall_dependencies.sh

# 安装管理后台依赖
pnpm-admin:
	@cd admin && CI="$(CI)" $(PNPM) install

# 安装 uni-app 应用端依赖
pnpm-uni-app:
	@cd uni-app && CI="$(CI)" $(PNPM) install

# 安装 Taro React 应用端依赖
pnpm-taro-app:
	@cd taro-app && CI="$(CI)" $(PNPM) install

# ===== 本地运行 =====

# 启动管理后台开发环境
run:
	@$(MAKE) run-admin

# 启动管理后台开发服务
run-admin:
	@cd admin && CI="$(CI)" $(PNPM) run dev

# 启动 uni-app 微信小程序开发服务
run-uni-app:
	@cd uni-app && CI="$(CI)" $(PNPM) run dev:mp-weixin

# 启动 Taro 微信小程序开发服务
run-taro-app:
	@cd taro-app && CI="$(CI)" $(PNPM) run dev:mp-weixin

# ===== 代码检查 =====

# 检查管理后台的静态规则、类型、测试和包边界
check-admin:
	@cd admin && \
		CI="$(CI)" $(PNPM) exec oxlint --vue-plugin --config ./internal/lint-config/oxlint.json --ignore-path .oxlintignore ./apps ./packages ./internal ./scripts && \
		CI="$(CI)" $(PNPM) run type:check && \
		CI="$(CI)" $(PNPM) run test && \
		CI="$(CI)" $(PNPM) run check:exports

# 检查 uni-app 的静态规则、类型、测试和包边界
check-uni-app:
	@cd uni-app && \
		CI="$(CI)" $(PNPM) exec eslint . && \
		CI="$(CI)" $(PNPM) run tsc && \
		CI="$(CI)" $(PNPM) run test && \
		CI="$(CI)" $(PNPM) run check:exports

# 检查 Taro 的静态规则、类型、测试和包边界
check-taro-app:
	@cd taro-app && \
		CI="$(CI)" $(PNPM) exec eslint . --ext .js,.mjs,.ts,.tsx && \
		CI="$(CI)" $(PNPM) run tsc && \
		CI="$(CI)" $(PNPM) run test && \
		CI="$(CI)" $(PNPM) run check:exports

# 按管理后台、uni-app、Taro 的顺序检查全部前端
check:
	@$(MAKE) check-admin
	@$(MAKE) check-uni-app
	@$(MAKE) check-taro-app
	@echo "==> 全部前端检查完成"

# ===== 应用与模块构建 =====

# 构建管理后台静态资源
build:
	@$(MAKE) build-admin

# 构建管理后台静态资源
build-admin:
	@cd admin && CI="$(CI)" $(PNPM) run build

# 构建 uni-app H5 静态资源
build-uni-app:
	@cd uni-app && CI="$(CI)" $(PNPM) run build:h5

# 构建 Taro H5 静态资源
build-taro-app:
	@cd taro-app && CI="$(CI)" $(PNPM) run build:h5

# 构建管理后台、uni-app H5 和 Taro H5 宿主
build-all:
	@$(MAKE) build-admin
	@$(MAKE) build-uni-app
	@$(MAKE) build-taro-app
	@echo "==> 全部前端宿主构建完成"

# 构建可供业务后台组合使用的管理端模块包
build-package:
	@cd admin && CI="$(CI)" $(PNPM) run build:package

# ===== npm 打包 =====

# 检查并生成管理后台 npm 包
package-admin:
	@$(MAKE) check-admin
	@$(MAKE) build-package
	@for package_dir in admin/packages/core admin/packages/modules/system admin/packages/cli; do \
		(cd "$$package_dir" && CI="$(CI)" $(PNPM) pack --pack-destination "$(NPM_PACK_DEST)") || exit 1; \
	done

# 检查并生成 uni-app npm 包
package-uni-app:
	@$(MAKE) check-uni-app
	@cd uni-app && CI="$(CI)" $(PNPM) run build:packages

# 检查并生成 Taro npm 包
package-taro-app:
	@$(MAKE) check-taro-app
	@cd taro-app && CI="$(CI)" $(PNPM) run build:packages

# 按管理后台、uni-app、Taro 的顺序检查并生成全部 npm 包
package:
	@$(MAKE) package-admin
	@$(MAKE) package-uni-app
	@$(MAKE) package-taro-app
	@echo "==> 全部前端 npm 包生成完成"

# ===== npm 发布 =====

# 检查、打包并发布管理后台 npm 包
publish-admin:
	@$(MAKE) package-admin
	@$(NPM_PUBLISH_ENV) bash scripts/publish_packages.sh admin/packages/core admin/packages/modules/system admin/packages/cli

# 检查、打包并发布 uni-app npm 包
publish-uni-app:
	@$(MAKE) package-uni-app
	@$(NPM_PUBLISH_ENV) bash scripts/publish_packages.sh uni-app/packages/core uni-app/packages/modules/system uni-app/packages/cli

# 检查、打包并发布 Taro npm 包
publish-taro-app:
	@$(MAKE) package-taro-app
	@$(NPM_PUBLISH_ENV) bash scripts/publish_packages.sh taro-app/packages/core taro-app/packages/ui taro-app/packages/modules/system taro-app/packages/cli

# 发布已经检查并构建的全部 npm 包
publish-only:
	@$(NPM_PUBLISH_ENV) bash scripts/publish_packages.sh admin/packages/core admin/packages/modules/system admin/packages/cli uni-app/packages/core uni-app/packages/modules/system uni-app/packages/cli taro-app/packages/core taro-app/packages/ui taro-app/packages/modules/system taro-app/packages/cli

# 按打包、发布的顺序发布全部 npm 包
publish:
	@$(MAKE) package
	@$(MAKE) publish-only

# ===== 帮助 =====

# 查看常用流程、全部目标及可覆盖参数
help:
	@echo ""
	@echo "常用流程:"
	@echo "  make init                         安装三个 workspace 的依赖"
	@echo "  make run                          启动管理后台"
	@echo "  make check                        检查三个 workspace"
	@echo "  make build-all                    构建管理端、uni-app H5、Taro H5"
	@echo "  make package                      检查并生成 10 个 npm 包"
	@echo "  make publish                      打包并发布 10 个 npm 包"
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
	@printf "  %-24s %s\n" "PNPM" "pnpm 命令，当前: $(PNPM)"
	@printf "  %-24s %s\n" "CI" "是否使用 CI 模式，当前: $(CI)"
	@printf "  %-24s %s\n" "NPM_REGISTRY" "npm registry，当前: $(NPM_REGISTRY)"
	@printf "  %-24s %s\n" "NPM_ACCESS" "npm 访问级别，当前: $(NPM_ACCESS)"
	@printf "  %-24s %s\n" "NPM_TAG" "npm dist-tag，当前: $(NPM_TAG)"
	@printf "  %-24s %s\n" "NPM_PACK_DEST" "pnpm pack 输出目录，当前: $(NPM_PACK_DEST)"
	@printf "  %-24s %s\n" "NPM_SKIP_GIT_CHECKS" "发布时跳过 pnpm Git 检查，当前: $(NPM_SKIP_GIT_CHECKS)"
	@echo ""
	@echo "示例:"
	@echo "  make run-admin"
	@echo "  make run-uni-app"
	@echo "  make check-taro-app"
	@echo "  make publish NPM_REGISTRY=https://registry.npmjs.org/ NPM_TAG=next"

.DEFAULT_GOAL := help
