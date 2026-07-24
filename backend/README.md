# backend

`backend` 是 Go + Kratos 管理服务，提供 HTTP、gRPC、SSE、MCP、数据库访问、文件上传、静态资源托管和代码生成能力。

## 目录

```text
backend
├── api/proto       # base、common、system 的 proto 契约
├── api/gen         # 生成的 Go 接口代码
├── configs         # 运行配置
├── internal/cmd    # 服务启动入口和 Wire 组合根
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
root:112233@tcp(127.0.0.1:3306)/admin_test?charset=utf8mb4&parseTime=True&loc=Local&timeout=1000ms
```

初始化数据库：

```sql
CREATE DATABASE admin_test CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
```

```bash
mysql -uroot -p admin_test < sql/default-data.sql
mysql -uroot -p admin_test < sql/base_area.sql
```

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
