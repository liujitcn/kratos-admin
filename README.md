# admin

这是一个前后端分离的管理后台项目，仓库包含 Go + Kratos 后端、Vue 管理端、初始化 SQL 和设计文档。

## 模块文档

| 模块 | 文档 | 说明 |
| --- | --- | --- |
| 后端服务 | [backend/README.md](backend/README.md) | 服务启动、配置、接口生成、构建和校验。 |
| 管理后台 | [frontend/admin/README.md](frontend/admin/README.md) | 页面结构、环境变量、开发与构建命令。 |
| 应用壳子 | [frontend/app/README.md](frontend/app/README.md) | 基础应用、系统 app 接口、账户与 AI 会话。 |

## 仓库结构

```text
.
├── backend          # Go + Kratos 后端服务
├── frontend
│   ├── admin       # Vue 管理后台
│   ├── app         # uni-app 基础应用壳子
│   └── Makefile    # 前端聚合命令
├── sql             # 初始化数据 SQL
└── docs            # 项目设计文档
```

## 本地启动

1. 创建 `admin_test` 数据库。
2. 首次启动后端完成当前模型的自动迁移，然后导入 `sql/default-data.sql` 和 `sql/base_area.sql`。
3. 启动后端，默认 HTTP 地址为 `http://localhost:7001`。
4. 启动管理后台，默认地址为 `http://localhost:8848`。
5. 如需启动基础应用壳子，执行 `make -C frontend run-app`。

默认后台账号来自 `sql/default-data.sql`：

- `super / 112233`
- `admin / 112233`

接口契约位于 `backend/api/proto`，按 `base`、`common`、`system` 分域；后端服务、管理端 API、应用 API 和生成 RPC 类型使用相同分层。后端托管 `backend/data/admin` 与 `backend/data/app` 下的前端构建产物，对应 `/admin` 与 `/app` 路径。
