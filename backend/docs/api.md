# proto 契约与 HTTP 接口命名细则

> 新增或修改 proto、HTTP 接口前必读。核心约束见 `backend/AGENTS.md` 的「proto 与 HTTP 路径」节。

## proto package 与目录

- package 必须带版本号，按终端或模块分层命名：`system.admin.v1`、`system.app.v1`、`base.v1`、`common.v1`。
- 文件目录必须与 package 对齐，版本号落到目录层级，例如 `api/proto/system/admin/v1/auth.proto` 对应 `package system.admin.v1;`。
- 禁止新增未分组的 `admin.v1`、`app.v1`，以及不符合当前分层的公共包命名。
- Go 生成包 import 必须使用真实包名别名：`systemadminv1`、`systemappv1`、`basev1`、`commonv1`；生成 import path 必须带 `/v1`，禁止退回不带版本层级的历史路径。
- 前端 TS 生成类型与业务 import 必须带 `/v1/` 目录层级，例如 `@/rpc/system/admin/v1/auth`；禁止新增缺少模块分组或版本层级的历史路径。

## proto 字段与注释

- 直接对应数据库表字段的字段，必须严格按表结构字段顺序排列、编号从 1 连续递增。
- 新增表字段映射必须插入到与表结构一致的位置并重排其后编号；禁止只追加到消息末尾；删除字段直接重排，不保留 `reserved`。
- 通过 ID 关联映射的 `name` 等派生字段，统一追加在全部表字段映射之后。
- 每个 `message` 必须有中文注释，直接表达业务语义，例如"用户分页查询条件""部门详情响应"。
- 每个字段必须有中文尾注释（如 `];  // 数量`），并同步补齐 `(gnostic.openapi.v3.property).description`，两者语义一致；字段尾部已有注释时不要在上方重复写语义相同的注释。

## HTTP 路径规范（RESTful）

- 路径层级统一带版本号前缀：`/api/v1/{terminal}/{module}/{resource}`。
- 迁移旧接口时，新主路径必须用 `/api/v1/...`；旧路径 `/api/...` 通过 `additional_bindings` 暂时保留。
- 优先表达"资源"而不是"动作"：标准 CRUD 用 HTTP Method 区分动作，同一资源的查/增/改/删复用同一主路径。
- RESTful 判断以"资源建模"而不是"动词翻译"为准；业务动作优先抽象为会话、令牌、绑定关系、执行记录、导出任务等资源或子资源。
- 状态切换、排序、导出、树结构、选项集、统计汇总等非标准 CRUD 场景，必须抽象成资源或子资源，禁止动词式路径。
- 资源唯一标识放 path；筛选、分页、排序、统计维度放 query 或请求参数。
- 路径统一小写英文，沿用项目现有的分隔风格；禁止驼峰、中文、空格；资源段默认沿用当前项目的单数/不可数风格，不混用复数。
- 同一业务的列表、汇总、图表、明细等接口命名保持并列、稳定、可预测，例如 `/summary`、`/trend`、`/rank`、`/metric`、`/risk`、`/todo`。
- 批量操作无法自然映射为单资源操作时，优先使用资源集合语义或明确的子资源命名。
- RPC 方法名与最终资源语义保持一致，不保留历史动作式或歧义命名。

## 三处同步

HTTP 路径确定后，以下三处必须保持一致，禁止只改其中一处：
1. proto 中的 `google.api.http` 映射
2. `sql/default-data.sql` 中的 `base_api.path`
3. 前端请求地址（`src/rpc` 生成类型与调用处）

修改或新增 proto 时，同步检查：`api/gen`、后端 service/biz、前端 RPC 类型与调用处、OpenAPI、`sql/default-data.sql` 接口权限数据。
