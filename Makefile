# 仓库级 Makefile：git hooks、跨前后端检查与统一发布
VERSION ?=
I18N_MIGRATION_VERSION ?=

.PHONY: help init hooks check-boundary i18n-sync tag

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

# 同步语言包清单和前端注册产物
i18n-sync:
	@python3 scripts/sync_locales.py --write $(if $(strip $(I18N_MIGRATION_VERSION)),--migration-version "$(I18N_MIGRATION_VERSION)",)

# 统一升级、提交、打包、推送 tag，并等待 GitHub Actions 发布两个前端 npm 包
tag:
	@$(MAKE) -C backend project-docs
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
