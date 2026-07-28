# admin

这是一个前后端分离的管理后台项目，仓库包含 Go + Kratos 后端、Vue 管理端、内置数据库迁移和设计文档。

## 模块文档

| 模块 | 文档 | 说明 |
| --- | --- | --- |
| 后端服务 | [backend/README.md](backend/README.md) | 服务启动、配置、接口生成、构建和校验。 |
| 管理后台 | [frontend/admin/README.md](frontend/admin/README.md) | 页面结构、环境变量、开发与构建命令。 |
| 应用壳子 | [frontend/app/README.md](frontend/app/README.md) | 基础应用、系统 app 接口、账户与 AI 会话。 |
| 接入指南 | [docs/服务接入指南.md](docs/服务接入指南.md) | 后端模块、管理后台和应用端的完整接入流程。 |

## 仓库结构

```text
.
├── backend          # Go + Kratos 后端服务
├── frontend
│   ├── admin       # Vue 管理后台
│   ├── app         # @liujitcn/kratos-app 公共应用宿主包
│   └── Makefile    # 前端聚合命令
└── docs            # 项目设计文档
```

## 本地启动

1. 创建 `kratos_admin` 数据库。
2. 启动后端，GORM 先按当前模型建表，再自动执行 `backend/migration/assets/mysql` 中的增量迁移。
3. 后端默认 HTTP 地址为 `http://localhost:7001`。
4. 启动管理后台，默认地址为 `http://localhost:8848`。
5. 如需启动应用壳子，执行 `make -C frontend run-app`；业务应用通过 `@liujitcn/kratos-app` 注册自己的页面并调用公共启动入口。

## 统一发布

统一发布命令会自动递增（或使用显式 `VERSION`）两个前端包的版本，将工作区全部改动统一提交，
执行后端测试和两个前端包的检查打包，确认成功后推送分支及发布 tag。`npm/vX.Y.Z` tag 会触发
GitHub Actions，通过 npm Trusted Publishing (OIDC) 发布两个 npm 包；本地命令会等待 workflow 完成：

```bash
make tag VERSION=0.0.4
```

发布 `0.0.4` 时会依次推送 `v0.0.4`、`backend/v0.0.4` 和 `npm/v0.0.4`。不指定 `VERSION` 时，脚本会先拉取远程
tag，再按根目录最新 tag 自动递增 patch 版本。发布要求当前分支是远程默认分支且提交基线已与远程同步；
工作区中的本地改动会全部纳入版本提交。本机需要安装并登录 GitHub CLI (`gh auth login`)。如果工作区无改动且
当前提交已经带有最新根版本 tag，再次执行 `make tag` 不会升级版本：脚本会补推缺失的同版本 tag，并在 npm
workflow 失败时重跑、运行中继续等待、成功时直接结束。

首次使用前，在 npmjs.com 的两个包设置中分别添加同一个 Trusted Publisher：GitHub 用户填写
`liujitcn`，仓库填写 `kratos-admin`，workflow 文件填写 `publish-npm.yml`，允许 `npm publish`。
workflow 使用短期 OIDC 凭据，不需要保存 npm token，也不会在本地反复提示二次认证。

## npm 包发布

前端 npm 包由 `frontend/Makefile` 统一管理。完整版本发布请使用上面的 `make tag`；
仅需要在本地应急发布已准备好的包时，确保已登录目标 npm registry 后执行：

```bash
pnpm login
make -C frontend publish
```

该命令会先执行类型检查、构建并生成两个包，再依次发布
`@liujitcn/kratos-admin` 和 `@liujitcn/kratos-app`。生成的 `.tgz` 文件位于各自的
`dist/npm` 目录。发布前会查询精确版本，registry 中已经存在的版本会自动跳过，因此可以在部分成功后安全重试。
本地发布使用当前 npm 会话并保留交互等待，默认跳过 pnpm 的工作区干净检查；需要强制工作区无未提交修改时设置
`NPM_SKIP_GIT_CHECKS=false`。启用 `auth-and-writes` 2FA 的账号仍可能对每个本地发布动作分别认证，日常完整发布应使用 `make tag`。

发布到私有 registry 时通过变量覆盖地址和 tag：

```bash
make -C frontend publish \
  NPM_REGISTRY=https://npm.example.com/ \
  NPM_TAG=beta
```

`NPM_REGISTRY` 默认是可发布的官方地址 `https://registry.npmjs.org/`；常见的
`npmmirror.com` 通常是只读下载镜像，不能作为发布地址。

数据库迁移由 `backend/migration` 管理。所有模块统一将版本记录保存到默认数据库的
`base_migration` 表，使用 `module` 区分迁移模块、`data_source` 区分目标数据源；SQL
仍在各自配置的数据源执行。`data.database` 兼容单库配置，`data.databases` 可按名称
创建多个 GORM 客户端。`base_migration` 作为 GORM 注册模型由 admin 服务启动时自动
创建或更新；仅当默认数据库配置 `enable_migrate: true` 时才会建表并执行迁移。
版本记录只在全部升级脚本成功后写入；脚本失败会回滚当前版本、输出错误并阻止服务启动，
修复脚本后重启会重新执行未记录的增量版本。

默认后台账号来自 admin 基础迁移（与 `backend/migration/assets/mysql/v0.0.1/default_data.up.sql` 内容一致）：

- `super / 112233`
- `admin / 112233`

接口契约位于 `backend/api/proto`，按 `base`、`common`、`system` 分域；后端服务、管理端 API、应用 API 和生成 RPC 类型使用相同分层。后端托管 `backend/data/admin` 与 `backend/data/app` 下的前端构建产物，对应 `/admin` 与 `/app` 路径。
