# backend

`backend` 是 Go + Kratos 管理服务，提供 HTTP、gRPC、SSE、MCP、数据库访问、文件上传、静态资源托管和代码生成能力。

## 目录

```text
backend
├── api/proto       # Backend 自有 Proto 契约，common/v1 从 Buf 引入
├── api/gen         # 对外公开的 Go 协议、gRPC Client 与服务注册代码
├── cmd/server      # 独立微服务启动入口
├── configs         # 运行配置
├── core            # 不包含 Proto 的 Kratos 基础运行时 module
│   └── pkg/errorsx # 稳定的公共业务错误构造能力
├── internal
│   ├── agent       # Eino 模型、工具、回调和工作流适配
│   ├── biz         # Case 与业务规则
│   ├── config      # 配置解析和数据源初始化
│   ├── const       # Backend 内部共享常量
│   ├── data        # GORM 生成代码和队列数据适配
│   ├── server      # HTTP、gRPC、MCP、中间件和 OpenAPI
│   └── service     # Proto 服务实现
├── migration       # 版本化数据库迁移资源
├── app.go          # 对外模块门面与独立应用入口
└── wire_gen.go     # Backend 内部依赖装配生成代码
```

## 运行形态

Backend 同时支持独立微服务和进程内 Go 模块。两种形态共享 `api/gen/go` 中生成的
`FooServiceClient` 接口，调用方不得直接导入 `internal` 中的 Case、Repository、Model 或 Query。

独立微服务模式使用标准远程 gRPC 连接：

```go
conn, err := grpc.NewClient(target)
if err != nil {
	return err
}
client := systemadminv1.NewBaseUserServiceClient(conn)
```

同一进程的启动器通过根包创建模块，并把 `ClientConn()` 交给相同的生成客户端：

```go
module, cleanup, err := kratosadmin.NewModule(ctx)
if err != nil {
	return err
}
defer cleanup()

client := systemadminv1.NewBaseUserServiceClient(module.ClientConn())
```

`ClientConn()` 将调用分派给已注册的内部 Service，再进入 Case 和 Repository，不经过网络。
当前 Backend RPC 均为 unary；进程内连接会对 streaming RPC 返回明确的不支持错误。

其他 Kratos 启动器可直接把模块挂到 Core：

```go
module, moduleCleanup, err := kratosadmin.NewModule(ctx)
if err != nil {
	return err
}
app, hostCleanup, err := core.NewApp(ctx, core.WithModules(module))
```

调用方退出时需要依次执行 `hostCleanup` 和 `moduleCleanup`。独立部署入口使用
`kratosadmin.NewApp`，内部复用同一套模块装配并额外创建 HTTP、gRPC Server。
可编译的独立 module 示例位于 `examples/module-host`。

## 配置与数据库

默认配置文件位于 `configs`：

- `data.yaml`：数据库、Redis 和队列连接。
- `auth.yaml`：JWT 认证及白名单。
- `pprof.yaml`：性能分析服务。

默认数据库连接：

```text
root:112233@tcp(127.0.0.1:3306)/kratos_admin?charset=utf8mb4&parseTime=True&loc=Local&timeout=1000ms
```

多数据源配置使用 `data.databases`，名称必须与迁移版本目录下的一级子目录一致：

```yaml
data:
  databases:
    default:
      driver: mysql
      source: root:112233@tcp(127.0.0.1:3306)/kratos_admin
      enable_migrate: true
    shop:
      driver: mysql
      source: root:112233@tcp(127.0.0.1:3306)/shop
      enable_migrate: true
```

初始化数据库（仅需创建库，服务启动会由 GORM 自动建表并执行内置迁移）：

```sql
CREATE DATABASE kratos_admin CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
```

迁移资源位于 `migration/assets/mysql`，最外层只放版本目录，例如 `v0.0.1`。
版本目录下的直系文件属于 `default` 数据源；一级子目录名表示目标数据源：

```text
v0.0.1/
  default_data.description.md
  default_data.up.sql
  shop/
    shop.description.md
    shop.up.sql
```

服务启动时先完成各数据源 GORM Client 初始化，再按目录数据源执行迁移。所有模块统一
使用默认数据库的 `base_migration` 保存执行版本，`module` 区分迁移模块，`data_source`
区分目标数据源；同一版本的 `default` 和 `shop` 会产生两条记录。版本记录只在该版本的全部
升级脚本成功后写入；任一脚本失败会回滚当前版本、记录错误并阻止服务启动，修复后重启会重试。
迁移记录模型由 admin 启动入口注册到 GORM；默认数据库配置为 `enable_migrate: true`
时由 GORM 建表并执行迁移，未配置或为 `false` 时自动建表和迁移脚本都会跳过。

接入项目直接使用 `github.com/liujitcn/kratos-kit/database/gorm/migration` 注册
自己的 contributor。每个 contributor 可声明对基础模块的依赖；数据源客户端通过
`data.databases` 的名称注入，迁移执行器会按资源目录自动查找并执行。不依赖
kratos-admin 的项目也可以直接复用这套能力。

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

生成代码不得手工修改。Backend 自有协议以 `backend/api/proto` 为协议源；通用 `common/v1`
协议从 `buf.build/liujitcn/kratos-common` 引入。`make api`、`make ts` 和 `make ts-app`
会同时生成 Backend 自有协议与锁定版本的通用协议产物，`make openapi` 通过 Buf 依赖解析通用类型。

通用 OpenAPI 注册、SSE 发布、任务注册、队列编解码和进程内调用连接由 `backend/core` 提供；
Backend 的业务适配器、数据库访问和传输实现全部位于 `internal`。框架 request-id、
recovery、tracing、metadata 等拦截器由 `kratos-kit/rpc` 按配置统一挂载，Backend 不重复注册。

管理端构建产物位于 `data/admin`，应用端构建产物位于 `data/app`，后端启动后分别可通过 `http://localhost:7001/admin` 与 `http://localhost:7001/app` 访问。Backend 内置协议生成系统 OpenAPI 文档 `/api/docs/openapi/admin`；外部模块按各自声明的 key 暴露为 `/api/docs/openapi/{key}`，管理端 API 文档页面会读取文档选项并按 key 切换。

## 校验

```bash
go test ./...
go vet ./...
```
