# 仓库级 Makefile：git hooks 与跨前后端的模块边界检查
.PHONY: help init hooks check-boundary

# 初始化开发环境（git hooks）
init: hooks

# 启用 git hooks（提交前强制执行模块边界检查）
hooks:
	@chmod +x scripts/githooks/*
	@git config core.hooksPath scripts/githooks
	@echo "==> git hooks 已启用 (scripts/githooks)"

# 检查基础模块与其他业务模块未反向依赖业务模块（按 backend/service 目录自动发现）
check-boundary:
	@bash scripts/check_module_boundary.sh

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
