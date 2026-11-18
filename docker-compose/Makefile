# docker-compose 本地服务镜像：双架构（amd64/arm64）构建并推送华为云 SWR 与 Harbor。
# 先 make docker-build，再 make docker-push；也可按仓库或镜像单独执行（见 make help）。

DOCKER ?= docker
DOCKER_PLATFORMS ?= linux/amd64,linux/arm64

# 推送目标仓库：与 cloud-scp 主镜像保持同 namespace / Harbor 项目。
SWR_PUSH_PREFIX ?= swr.cn-north-4.myhuaweicloud.com/newcapec
HARBOR_PUSH_PREFIX ?= harbor.newcapec.cn/zhihuinengyuan

# 上游镜像源（tag 与 compose 引用一致；SWR 不收 OCI index，经 FROM 换壳转为 Docker manifest list）。
# gotenberg 用 daocloud 拉透全名（blob 与官方一致；官方名经 daemon mirror 拉取会挂死）。
GOTENBERG_SOURCE ?= docker.m.daocloud.io/gotenberg/gotenberg:9
XBERG_SOURCE ?= ghcr.io/xberg-io/xberg:latest

# markitdown 源码（固定提交；本机直连 GitHub 与 debian/pypi 官方源不通，prepare 时克隆到本地并换清华源构建）。
MARKITDOWN_REPO_DIR ?= ../../../microsoft/markitdown
MARKITDOWN_PIN ?= ed178952a6672abdfd35b5f9e76a80b6cefc086e
MARKITDOWN_CLONE_PROXY ?= http://127.0.0.1:7897

# 自建镜像构建上下文（GitHub pin，与 compose build 配置一致）。
GOTENBERG_MCP_CONTEXT ?= https://github.com/avhulst/coolify-gotenberg-mcp.git#f0c12646c0b1baa30435e3d3541f340ba844a860

# 镜像与 tag（与 compose 引用一致，不新增日期 tag）。
GOTENBERG_IMAGE ?= gotenberg:9
XBERG_IMAGE ?= xberg:latest
MARKITDOWN_IMAGE ?= markitdown-mcp:microsoft-ed17895
GOTENBERG_MCP_IMAGE ?= gotenberg-mcp:avhulst-f0c1264

# 聚合：构建全部镜像（双平台，产物直接挂 SWR 与 Harbor 全名 tag）。
docker-build: build-gotenberg build-xberg build-markitdown build-gotenberg-mcp

# 聚合：推送全部镜像到 Harbor 与 SWR（Harbor 内网快先推；SWR 走公网上行慢，可单独执行）。
docker-push: docker-push-harbor docker-push-hw

# 聚合：推送全部镜像到 Harbor。
docker-push-harbor: push-harbor-gotenberg push-harbor-xberg push-harbor-markitdown push-harbor-gotenberg-mcp

# 聚合：推送全部镜像到华为云 SWR。
docker-push-hw: push-hw-gotenberg push-hw-xberg push-hw-markitdown push-hw-gotenberg-mcp

# 上游镜像换壳构建：buildkit 引用原始 layer 不重打包，产物为 Docker manifest list v2（attestation 天然丢弃）。
build-gotenberg:
	@printf 'FROM %s\n' "$(GOTENBERG_SOURCE)" | "$(DOCKER)" buildx build \
		--platform "$(DOCKER_PLATFORMS)" --provenance=false --load \
		-t "$(SWR_PUSH_PREFIX)/$(GOTENBERG_IMAGE)" -t "$(HARBOR_PUSH_PREFIX)/$(GOTENBERG_IMAGE)" -

# 同上：xberg 源走 ghcr。
build-xberg:
	@printf 'FROM %s\n' "$(XBERG_SOURCE)" | "$(DOCKER)" buildx build \
		--platform "$(DOCKER_PLATFORMS)" --provenance=false --load \
		-t "$(SWR_PUSH_PREFIX)/$(XBERG_IMAGE)" -t "$(HARBOR_PUSH_PREFIX)/$(XBERG_IMAGE)" -

# 准备 markitdown 上游源码：克隆固定提交并将 Dockerfile 的 apt/pip 源换为清华镜像（官方源直连不通）。
prepare-markitdown:
	@test -d "$(MARKITDOWN_REPO_DIR)/.git" || (mkdir -p "$(MARKITDOWN_REPO_DIR)" && cd "$(MARKITDOWN_REPO_DIR)" && git init -q && git remote add origin https://github.com/microsoft/markitdown.git)
	@cd "$(MARKITDOWN_REPO_DIR)" && (https_proxy=$(MARKITDOWN_CLONE_PROXY) git fetch --depth 1 -q origin "$(MARKITDOWN_PIN)" || git cat-file -e "$(MARKITDOWN_PIN)^{commit}") && git checkout -q "$(MARKITDOWN_PIN)"
	@perl -pi -e 's|RUN apt-get update|RUN sed -i "s/deb.debian.org/mirrors.tuna.tsinghua.edu.cn/g" /etc/apt/sources.list.d/debian.sources \\\n    && apt-get update|' "$(MARKITDOWN_REPO_DIR)/packages/markitdown-mcp/Dockerfile"
	@sed -i '' 's|pip install --no-cache-dir /app|pip install --no-cache-dir -i https://pypi.tuna.tsinghua.edu.cn/simple /app|' "$(MARKITDOWN_REPO_DIR)/packages/markitdown-mcp/Dockerfile"
	@echo "==> markitdown 上游源码已就绪: $(MARKITDOWN_PIN)（apt/pip 已换清华源）"

# 自建镜像：从上游源码目录双平台构建（依赖 prepare-markitdown 换源）。
build-markitdown: prepare-markitdown
	@"$(DOCKER)" buildx build --platform "$(DOCKER_PLATFORMS)" --provenance=false --load \
		-t "$(SWR_PUSH_PREFIX)/$(MARKITDOWN_IMAGE)" -t "$(HARBOR_PUSH_PREFIX)/$(MARKITDOWN_IMAGE)" \
		"$(MARKITDOWN_REPO_DIR)/packages/markitdown-mcp"

# 自建镜像：gotenberg-mcp，构建上下文与 compose 保持同一提交 pin。
build-gotenberg-mcp:
	@"$(DOCKER)" buildx build --platform "$(DOCKER_PLATFORMS)" --provenance=false --load \
		-t "$(SWR_PUSH_PREFIX)/$(GOTENBERG_MCP_IMAGE)" -t "$(HARBOR_PUSH_PREFIX)/$(GOTENBERG_MCP_IMAGE)" \
		"$(GOTENBERG_MCP_CONTEXT)"

# 推送 gotenberg:9 到 Harbor。
push-harbor-gotenberg:
	@"$(DOCKER)" image inspect "$(HARBOR_PUSH_PREFIX)/$(GOTENBERG_IMAGE)" >/dev/null 2>&1 || (echo "未找到镜像 $(HARBOR_PUSH_PREFIX)/$(GOTENBERG_IMAGE)，请先执行 make build-gotenberg" && exit 1)
	@"$(DOCKER)" push "$(HARBOR_PUSH_PREFIX)/$(GOTENBERG_IMAGE)"

# 推送 xberg:latest 到 Harbor。
push-harbor-xberg:
	@"$(DOCKER)" image inspect "$(HARBOR_PUSH_PREFIX)/$(XBERG_IMAGE)" >/dev/null 2>&1 || (echo "未找到镜像 $(HARBOR_PUSH_PREFIX)/$(XBERG_IMAGE)，请先执行 make build-xberg" && exit 1)
	@"$(DOCKER)" push "$(HARBOR_PUSH_PREFIX)/$(XBERG_IMAGE)"

# 推送 markitdown-mcp 到 Harbor。
push-harbor-markitdown:
	@"$(DOCKER)" image inspect "$(HARBOR_PUSH_PREFIX)/$(MARKITDOWN_IMAGE)" >/dev/null 2>&1 || (echo "未找到镜像 $(HARBOR_PUSH_PREFIX)/$(MARKITDOWN_IMAGE)，请先执行 make build-markitdown" && exit 1)
	@"$(DOCKER)" push "$(HARBOR_PUSH_PREFIX)/$(MARKITDOWN_IMAGE)"

# 推送 gotenberg-mcp 到 Harbor。
push-harbor-gotenberg-mcp:
	@"$(DOCKER)" image inspect "$(HARBOR_PUSH_PREFIX)/$(GOTENBERG_MCP_IMAGE)" >/dev/null 2>&1 || (echo "未找到镜像 $(HARBOR_PUSH_PREFIX)/$(GOTENBERG_MCP_IMAGE)，请先执行 make build-gotenberg-mcp" && exit 1)
	@"$(DOCKER)" push "$(HARBOR_PUSH_PREFIX)/$(GOTENBERG_MCP_IMAGE)"

# 推送 gotenberg:9 到华为云 SWR。
push-hw-gotenberg:
	@"$(DOCKER)" image inspect "$(SWR_PUSH_PREFIX)/$(GOTENBERG_IMAGE)" >/dev/null 2>&1 || (echo "未找到镜像 $(SWR_PUSH_PREFIX)/$(GOTENBERG_IMAGE)，请先执行 make build-gotenberg" && exit 1)
	@"$(DOCKER)" push "$(SWR_PUSH_PREFIX)/$(GOTENBERG_IMAGE)"

# 推送 xberg:latest 到华为云 SWR。
push-hw-xberg:
	@"$(DOCKER)" image inspect "$(SWR_PUSH_PREFIX)/$(XBERG_IMAGE)" >/dev/null 2>&1 || (echo "未找到镜像 $(SWR_PUSH_PREFIX)/$(XBERG_IMAGE)，请先执行 make build-xberg" && exit 1)
	@"$(DOCKER)" push "$(SWR_PUSH_PREFIX)/$(XBERG_IMAGE)"

# 推送 markitdown-mcp 到华为云 SWR。
push-hw-markitdown:
	@"$(DOCKER)" image inspect "$(SWR_PUSH_PREFIX)/$(MARKITDOWN_IMAGE)" >/dev/null 2>&1 || (echo "未找到镜像 $(SWR_PUSH_PREFIX)/$(MARKITDOWN_IMAGE)，请先执行 make build-markitdown" && exit 1)
	@"$(DOCKER)" push "$(SWR_PUSH_PREFIX)/$(MARKITDOWN_IMAGE)"

# 推送 gotenberg-mcp 到华为云 SWR。
push-hw-gotenberg-mcp:
	@"$(DOCKER)" image inspect "$(SWR_PUSH_PREFIX)/$(GOTENBERG_MCP_IMAGE)" >/dev/null 2>&1 || (echo "未找到镜像 $(SWR_PUSH_PREFIX)/$(GOTENBERG_MCP_IMAGE)，请先执行 make build-gotenberg-mcp" && exit 1)
	@"$(DOCKER)" push "$(SWR_PUSH_PREFIX)/$(GOTENBERG_MCP_IMAGE)"

# 输出目标速览。
help:
	@echo "构建（双平台 amd64+arm64，产物挂 SWR/Harbor 全名）:"
	@echo "  make docker-build           构建全部镜像"
	@echo "  make build-gotenberg        gotenberg:9（上游 FROM 换壳）"
	@echo "  make build-xberg            xberg:latest（上游 FROM 换壳）"
	@echo "  make build-markitdown       markitdown-mcp:microsoft-ed17895（prepare 换源后源码构建）"
	@echo "  make build-gotenberg-mcp    gotenberg-mcp:avhulst-f0c1264（源码构建）"
	@echo "推送:"
	@echo "  make docker-push            推送全部镜像到 Harbor 与 SWR"
	@echo "  make docker-push-harbor     推送全部镜像到 Harbor"
	@echo "  make docker-push-hw         推送全部镜像到华为云 SWR"
	@echo "  make push-harbor-<name>     单镜像推 Harbor（name: gotenberg/xberg/markitdown/gotenberg-mcp）"
	@echo "  make push-hw-<name>         单镜像推 SWR（name 同上）"

.PHONY: docker-build docker-push docker-push-harbor docker-push-hw \
	prepare-markitdown \
	build-gotenberg build-xberg build-markitdown build-gotenberg-mcp \
	push-harbor-gotenberg push-harbor-xberg push-harbor-markitdown push-harbor-gotenberg-mcp \
	push-hw-gotenberg push-hw-xberg push-hw-markitdown push-hw-gotenberg-mcp \
	help
