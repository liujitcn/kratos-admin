# backend

`backend` 是 Go + Kratos 管理服务，提供 HTTP、gRPC、SSE、MCP、数据库访问、文件上传、静态资源托管和代码生成能力。

## 目录

```text
backend
├── api/proto       # base、common、system 的 proto 契约
├── api/gen         # 生成的 Go 接口代码
├── configs         # 运行配置
├── internal/cmd    # 服务启动入口和 Wire 组合根
├── migration       # 版本化数据库迁移资源
├── pkg             # 配置、公共能力、生成模型和中间件
├── server          # 传输层服务注册
└── service         # base、system 业务用例与服务
```

## 配置与数据库

默认配置文件位于 `configs`：

- `data.yaml`：数据库、Redis 和队列连接。
- `auth.yaml`：JWT 认证及白名单。
- `pprof.yaml`：性能分析服务。

默认数据库连接：

```text
root:112233@tcp(127.0.0.1:3306)/kratos_admin?charset=utf8mb4&parseTime=True&loc=Local&timeout=1000ms
```

初始化数据库（仅需创建库，服务启动会由 GORM 自动建表并执行内置迁移）：

```sql
CREATE DATABASE kratos_admin CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
```

迁移资源位于 `migration/assets/mysql`，最外层只放版本目录，例如
`v0.0.1`。每个版本目录必须包含同一功能名的 `<feature>.up.sql` 和
`<feature>.description.md`：前者记录本次升级的 SQL，后者记录本次升级内容。
所有模块统一使用默认数据库的 `base_migration` 保存执行版本，并通过
`business` 字段区分模块；服务启动时先完成 GORM 建表，再在对应数据源执行基础数据和
地区数据迁移，不需要手动导入 SQL。迁移记录模型由 admin 启动入口注册到 GORM；默认数据库配置为 `enable_migrate: true` 时
由 GORM 建表并执行默认库迁移，未配置或为 `false` 时自动建表和迁移脚本都会跳过。

接入项目直接使用 `github.com/liujitcn/kratos-kit/database/gorm/migration` 注册
自己的 contributor。每个 contributor 可声明独立目标数据源和对基础模块的依赖，
迁移执行器会在统一版本表中按 `business` 区分记录；不依赖 kratos-admin 的项目也可以
直接复用这套能力。

## 常用命令

```bash
make api       # 生成 Go 接口代码
make openapi   # 生成 OpenAPI 文档
make ts        # 生成管理端 TypeScript RPC
make ts-app    # 生成应用端 TypeScript RPC
make gorm-gen  # 生成 GORM 模型、查询和数据访问代码
make wire      # 生成依赖注入代码
make gen       # 执行全部生成命令
make run       # 启动服务
make build     # 构建 Linux 可执行文件
```

生成代码不得手工修改，接口和表结构变更后使用对应 Makefile 目标重新生成。

管理端构建产物位于 `data/admin`，应用端构建产物位于 `data/app`，后端启动后分别可通过 `http://localhost:7001/admin` 与 `http://localhost:7001/app` 访问。OpenAPI 文档接口为 `/api/docs/openapi`。

## 校验

```bash
go test ./...
go vet ./...
```
