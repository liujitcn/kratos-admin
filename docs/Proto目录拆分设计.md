# 接口域目录设计

## 状态

本文档记录当前接口与服务的分域方式。协议目录按能力域和终端组织，所有新代码必须落到下列边界。

## 目录结构

```text
backend
├── api
│   ├── proto
│   │   ├── base/v1            # 登录、文件、OAuth、SSE、AI、MCP 等公共能力
│   │   ├── common/v1          # 错误、通用类型和跨域基础枚举
│   │   └── system
│   │       ├── common/v1      # 系统域枚举
│   │       ├── admin/v1       # 系统后台接口
│   │       └── app/v1         # 系统通用端接口
│   └── gen/go                 # 与 proto 目录同构的生成结果
├── service
│   ├── base
│   └── system/{admin,app}
└── server
    ├── base
    └── system/{admin,app}
```

前端保留管理后台和基础应用壳子：

```text
frontend/admin/src/{api,rpc,views}/
  base/
  system/admin/

frontend/app/src/
  api/{base,system}/       # 手写 service；system 下只调用 system.app
  rpc/system/app/v1/       # 生成的 system.app 类型
  pages/                   # 基础页面
  pagesMember/             # 壳子分包
```

`frontend/admin` 是唯一业务前端入口；`frontend/app` 只保留框架、基础页面以及 `base`、`system/app` 接口，不承载独立业务模块。

## 域归属

| 域 | 范围 |
| --- | --- |
| `base` | 登录、OAuth、文件、SSE、MCP、AI 会话与消息等跨终端基础能力。 |
| `common` | 错误、分页、基础类型和无业务归属的公共枚举。 |
| `system` | 认证、用户、角色、部门、岗位、菜单、字典、配置、日志、任务、租户和代码生成。 |

基础层不能依赖未来业务模块的协议或实现。需要新增业务时，使用独立域目录并通过组合根显式注册。

## 命名和生成约束

| 位置 | 当前约定 |
| --- | --- |
| Proto package | `system.admin.v1`、`system.app.v1`，公共包为 `base.v1`、`common.v1`。 |
| Go import alias | `systemadminv1`、`systemappv1`、`basev1`、`commonv1`。 |
| 管理后台 RPC | `src/rpc/system/admin/v1`。 |
| 手写前端 API | `src/api/{base,system}/...`。 |

接口变更在 `backend` 目录通过 `make api`、`make openapi`、`make ts` 生成。`api/gen/go`、前端 `src/rpc` 和 OpenAPI 均不可手工改写。

## 注册与菜单连带

新增服务或移动域时，除了修改 Proto 与 `service`，还要同步检查：

1. `server/system/{admin,app}/register.go` 的 HTTP、gRPC 和工具注册。
2. `internal/cmd/server` 的 Wire 装配是否包含新依赖。
3. `backend/api/buf*.yaml` 的生成输入和 TypeScript 端过滤规则。
4. 管理后台 `src/api`、`src/views`、动态菜单组件路径和 `sql/default-data.sql` 中的菜单/API 权限数据。
5. 代码生成器的目标域和输出路径。

HTTP 路径以 Proto 的显式 `google.api.http` 为准。目录迁移不等于可以任意改变已有 HTTP 路径、OpenAPI operation 或权限数据。
