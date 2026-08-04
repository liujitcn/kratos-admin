# Proto 与 HTTP 契约

本文记录当前仓库新增或修改接口时必须保持一致的契约规则。

## 目录和 package

| 目录 | package | 用途 |
| --- | --- | --- |
| `api/proto/base/v1` | `base.v1` | 管理端与应用端共用能力。 |
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

资源名使用小写 kebab-case；列表使用集合路径，详情和动作使用路径参数。接口路径修改直接同步 Proto、权限脚本和前端请求，不保留旧路径或用于兼容的 `additional_bindings`。

## 接口修改策略

- 本仓库不提供 service、message、field、enum 或 HTTP 路径的向后兼容。
- 每次契约修改都按一次完整变更处理，可以直接调整名称、结构、顺序和编号；必须在同一变更中同步所有后端调用、前端调用、数据库已有数据与初始化数据、权限脚本、OpenAPI 和生成产物。
- 不保留旧 service 方法、旧 message 字段、旧 enum 名称、旧 enum 编号或旧 HTTP 路径；禁止使用 `additional_bindings`、`allow_alias` 等兼容手段。协议确需为同一 RPC 映射多个 HTTP 方法时，`additional_bindings` 只按协议语义使用。

## 字段和注释

- 每个 `message` 必须有中文注释。
- 每个字段同时提供中文尾注释和 `(gnostic.openapi.v3.property).description`，语义保持一致。
- 字段按业务或表结构顺序排列，编号连续；公共审计字段沿用项目现有编号。
- 请求格式约束写入 `buf.validate`；唯一性、存在性、权限和状态流转由 biz 校验。
- 枚举引用真实 package，避免复制等价枚举。
- 枚举位置按调用方范围收敛：只被一个业务 Proto 文件使用的 enum，放在该文件中，紧随对应 service 定义并与 message 同级；同一 package 跨多个业务文件使用、且不属于单一业务核心的共享类型，放在该 package 的 `common.proto`；同一端多个业务文件使用但有明确核心 service/message 归属的 enum，放在主文件，其他文件显式 import；同时被 `system.admin.v1` 和 `system.app.v1` 使用的 enum，才放入更上层 shared enum 文件。Proto 不支持在 service 体内直接声明 enum，因此“当前 service 下”按当前业务文件的 service 定义之后实现，不使用非法嵌套。

## 枚举命名与修改

- enum 类型名使用 PascalCase/TitleCase；缩写按单词处理，例如 `AiMessageStatus`，不要写成 `AIMessageStatus`。
- enum 值使用 UPPER_SNAKE_CASE。
- package 级别 enum 的每个值都使用完整类型名前缀，前缀转换为 UPPER_SNAKE_CASE。例如 `BaseConfigType` 使用 `BASE_CONFIG_TYPE_TEXT`，不要使用无前缀的 `TEXT`；只有仅被一个 message 使用的嵌套 enum，非零值才可以省略类型前缀。
- 第一个值必须为数值 `0`，默认命名为 `<ENUM_TYPE>_UNSPECIFIED`，且不表达任何有效业务状态。只有领域确实需要“未知”业务状态时才使用 `<ENUM_TYPE>_UNKNOWN`，不得使用 `UNKNOWN_BCT` 这类缩写零值。
- 新增 enum 的正数值按顺序分配；枚举调整按一次完整变更处理，不保留旧名称或旧编号兼容。
- enum 和每个 enum 值都必须有中文注释，注释说明业务语义而不是重复标识符。
- 禁止使用 `allow_alias`，不为旧 ProtoJSON 名称提供兼容别名。
- 修改 enum 类型名、值名称、顺序或编号后，必须在同一变更中同步业务引用、数据库已有数据和初始化数据、Go、OpenAPI 及各前端 TypeScript 产物；禁止手改生成目录。

示例：

```proto
enum BaseConfigType {
  BASE_CONFIG_TYPE_UNSPECIFIED = 0;
  BASE_CONFIG_TYPE_TEXT = 1;
  BASE_CONFIG_TYPE_IMAGE = 2;
}
```

详细背景和官方来源见 [enum 命名调研](../../docs/research-enum-naming.md)。

## 同步范围

一次接口变更至少核对：

1. Proto 的 `google.api.http` 和 `buf.validate`。
2. `make api openapi` 生成的 Go 与 OpenAPI。
3. 后端 service、biz 和注册代码。
4. `make ts`、`make ts-uni-app` 或 `make ts-taro-app` 生成的前端 RPC 类型，以及消费端人工维护的请求封装。
5. `migration/assets/<version>/<database-type>/<feature>.up.sql` 中的菜单、按钮和服务方法权限。

生成目录不得手工修改。完整流程见 [new-feature.md](new-feature.md)。
