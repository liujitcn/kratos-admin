# Proto 与 HTTP 契约

本文记录当前仓库新增或修改接口时必须保持一致的契约规则。

## 目录和 package

| 目录 | package | 用途 |
| --- | --- | --- |
| `api/proto/base/v1` | `base.v1` | admin 与 app 共用能力。 |
| `api/proto/system/admin/v1` | `system.admin.v1` | 管理后台。 |
| `api/proto/system/app/v1` | `system.app.v1` | 应用端。 |
| `api/proto/system/common/v1` | `system.common.v1` | System 两端共享类型。 |

`common.v1` 来自 Buf 依赖，不在本仓库重复维护。目录、Proto package、Go import 别名和前端 RPC 层级必须一致。

## HTTP 路径

- 公共接口：`/api/v1/base/<resource>`。
- 管理端：`/api/v1/admin/<resource>`。
- 应用端：`/api/v1/app/<resource>`。
- SSE：`/events/{stream}`。
- MCP：`/mcp/{terminal}`。

资源名使用小写 kebab-case；列表使用集合路径，详情和动作使用路径参数。兼容旧接口时使用 `additional_bindings`，并明确删除时机。

## 字段和注释

- 每个 `message` 必须有中文注释。
- 每个字段同时提供中文尾注释和 `(gnostic.openapi.v3.property).description`，语义保持一致。
- 字段按业务或表结构顺序排列，编号连续；公共审计字段沿用项目现有编号。
- 请求格式约束写入 `buf.validate`；唯一性、存在性、权限和状态流转由 biz 校验。
- 枚举引用真实 package，避免复制等价枚举。

## 同步范围

一次接口变更至少核对：

1. Proto 的 `google.api.http` 和 `buf.validate`。
2. `make api openapi` 生成的 Go 与 OpenAPI。
3. 后端 service、biz 和注册代码。
4. `make ts` 或 `make ts-app` 生成的前端 RPC 类型，以及消费端人工维护的请求封装。
5. `migration/assets/<version>/<database-type>/<feature>.up.sql` 中的菜单、按钮和服务方法权限。

生成目录不得手工修改。完整流程见 [new-feature.md](new-feature.md)。
