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
