# 仓库级 Makefile：git hooks、跨前后端检查与统一发布
VERSION ?=
I18N_LOCALE ?=
I18N_SOURCE_LOCALE ?= zh-CN
I18N_LOCALES ?= en-US,zh-TW,ja-JP
I18N_OFFLINE ?= 0
I18N_AUTO_TRANSLATE ?= 1
I18N_MIGRATION_VERSION ?=

.PHONY: help init hooks check-boundary i18n i18n-check i18n-sync i18n-locale i18n-docs i18n-openapi tag

# 初始化开发环境（git hooks）
init: hooks

# 启用 git hooks（提交前强制执行模块边界检查）
hooks:
	@chmod +x scripts/githooks/*
	@git config core.hooksPath scripts/githooks
	@echo "==> git hooks 已启用 (scripts/githooks)"

# 检查管理端 npm 包的依赖边界
check-boundary:
	@bash scripts/check_admin_boundary.sh

# 执行常规国际化生成链路，不生成新语言文件
i18n: i18n-sync i18n-docs i18n-openapi

# 只检查语言包、语言集合和已提交的注册生成物
i18n-check:
	@$(MAKE) -C backend i18n-check

# 同步语言包集合、前端注册文件和代码生成语言目录
i18n-sync:
	@$(MAKE) -C backend i18n-sync \
		I18N_MIGRATION_VERSION="$(I18N_MIGRATION_VERSION)"

# 生成指定语言的固定语言包和动态翻译 SQL
i18n-locale:
	@test -n "$(I18N_LOCALE)" || (echo "请指定 I18N_LOCALE，例如 make i18n-locale I18N_LOCALE=ja-JP" && exit 1)
	@$(MAKE) -C backend i18n-locale \
		I18N_LOCALE="$(I18N_LOCALE)" \
		I18N_MIGRATION_VERSION="$(I18N_MIGRATION_VERSION)" \
		I18N_OFFLINE="$(I18N_OFFLINE)"

# 收集并翻译项目 Markdown 文档
i18n-docs:
	@$(MAKE) -C backend i18n-docs \
		I18N_LOCALES="$(I18N_LOCALES)" \
		I18N_SOURCE_LOCALE="$(I18N_SOURCE_LOCALE)" \
		I18N_OFFLINE="$(I18N_OFFLINE)"

# 生成 OpenAPI 多语言 YAML
i18n-openapi:
	@$(MAKE) -C backend i18n-openapi \
		I18N_LOCALES="$(I18N_LOCALES)" \
		I18N_SOURCE_LOCALE="$(I18N_SOURCE_LOCALE)" \
		I18N_OFFLINE="$(I18N_OFFLINE)" \
		I18N_AUTO_TRANSLATE="$(I18N_AUTO_TRANSLATE)"

# 统一升级、提交、打包、推送 tag，并等待 GitHub Actions 发布两个前端 npm 包
tag:
	@$(MAKE) i18n-docs
	@python3 scripts/tag_release.py $(if $(strip $(VERSION)),--version "$(VERSION)",)

# 查看所有可用目标及说明
help:
	@echo ""
	@echo "用法:"
	@echo " make [目标]"
	@echo ""
	@echo '可用目标:'
	@awk '/^[a-zA-Z\-_0-9]+:/ { \
	helpMessage = match(lastLine, /^# (.*)/); \
		if (helpMessage) { \
			helpCommand = substr($$1, 0, index($$1, ":")-1); \
			helpMessage = substr(lastLine, RSTART + 2, RLENGTH); \
			printf "\033[36m%-22s\033[0m %s\n", helpCommand,helpMessage; \
		} \
	} \
	{ lastLine = $$0 }' $(MAKEFILE_LIST)
