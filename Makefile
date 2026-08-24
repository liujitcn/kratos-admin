# 仓库级 Makefile
#
# 常用流程：
#   初始化 Git hooks：make init
#   全仓检查：make check
#   国际化生成：make i18n
#   统一发布：make tag VERSION=0.0.1

.PHONY: help init hooks check \
	i18n i18n-check i18n-sync i18n-locale i18n-docs i18n-openapi \
	tag

# ===== 公共参数 =====

PYTHON ?= python3
VERSION ?=

# ===== 国际化参数 =====

I18N_LOCALE ?=
I18N_SOURCE_LOCALE ?= zh-CN
I18N_LOCALES ?= en-US,zh-TW,ja-JP
I18N_OFFLINE ?= 0
I18N_AUTO_TRANSLATE ?= 1
I18N_MIGRATION_VERSION ?=

# ===== 环境初始化 =====

# 初始化仓库 Git hooks
init:
	@$(MAKE) hooks

# 启用 Git hooks（提交前执行管理端暂存文件检查）
hooks:
	@chmod +x scripts/githooks/*
	@git config core.hooksPath scripts/githooks
	@echo "==> Git hooks 已启用: scripts/githooks"

# ===== 全仓检查 =====

# 按 Backend、Frontend 的顺序执行全仓检查
check:
	@$(MAKE) -C backend check
	@$(MAKE) -C frontend check
	@echo "==> 全仓检查完成"

# ===== 国际化 =====

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

# 收集并本地化项目 Markdown 文档
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

# 按语言包、项目文档、OpenAPI 的顺序执行国际化生成
i18n:
	@$(MAKE) i18n-sync
	@$(MAKE) i18n-docs
	@$(MAKE) i18n-openapi
	@echo "==> 全仓国际化产物生成完成"

# ===== 发布 =====

# 统一升级、提交、打包并发布 Backend 与 10 个前端 npm 包
tag:
	@$(MAKE) i18n-docs
	@$(PYTHON) scripts/tag_release.py $(if $(strip $(VERSION)),--version "$(VERSION)",)

# ===== 帮助 =====

# 查看常用流程、全部目标及可覆盖参数
help:
	@echo ""
	@echo "常用流程:"
	@echo "  make init                         初始化 Git hooks"
	@echo "  make check                        执行 Backend 和 Frontend 检查"
	@echo "  make i18n                         生成全部国际化产物"
	@echo "  make tag VERSION=0.0.1            统一发布（会提交并推送）"
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
	@printf "  %-24s %s\n" "VERSION" "发布版本；为空时由发布脚本决定"
	@printf "  %-24s %s\n" "I18N_LOCALES" "目标语言，当前: $(I18N_LOCALES)"
	@printf "  %-24s %s\n" "I18N_SOURCE_LOCALE" "源语言，当前: $(I18N_SOURCE_LOCALE)"
	@printf "  %-24s %s\n" "I18N_OFFLINE" "设为 1 时离线生成"
	@printf "  %-24s %s\n" "I18N_AUTO_TRANSLATE" "OpenAPI 是否自动翻译，当前: $(I18N_AUTO_TRANSLATE)"
	@echo ""
	@echo "更多命令:"
	@echo "  make -C backend help              查看 Backend 命令"
	@echo "  make -C frontend help             查看 Frontend 命令"

.DEFAULT_GOAL := help
