# Codex 规则（backend）

## 交付顺序
- 完整流程见 [服务接入指南](../docs/服务接入指南.md)，新增业务前必须先读。
- 需要新表时，先按 `configs/data.yaml` 确认连接并把表结构建到开发库，再执行 `make gorm-gen`；随后按“Proto 契约 → `make gen` → service/biz → 前端 → `v0.0.1` 初始化脚本”的顺序完成。
- 本次任务全部改动完成后统一执行生成与测试：至少运行 `go test ./...`；检查 `README.md` 后再按仓库流程提交。不要在中间反复执行全量生成和测试。
- 关键业务规则和安全边界允许编写小而稳定的 `_test.go` 回归测试；临时测试用完即删。

## 模块与契约对应
- Proto 是 HTTP、gRPC、校验和前端类型的唯一契约源。`api/proto/<module>/v1` 对应 `internal/service/<module>/v1`；Biz 去掉版本目录，对应 `internal/biz/<module>`，例如 `system/admin/v1` 对应 `service/system/admin/v1` 和 `biz/system/admin`。
- 每个 Proto `service XxxService` 必须有且只有一个手写 `XxxService` 实现，文件名为 `<name>_service.go`，构造函数为 `NewXxxService`，并嵌入生成的 `UnimplementedXxxServiceServer`。Service 目录中的 `*Service` 不得脱离对应 Proto 另行创建；`*_http.pb.go` 等生成文件和传输辅助函数不属于手写 Service。
- Service 只负责请求/响应适配、调用对应 Case、日志和错误兜底；事务、租户、权限、状态和业务规则放在 Biz，数据库访问限制在 data。
- `internal/biz/<module>` 直接承载对应 Proto 模块的主 Case。主 Case 名称必须去掉 `Service` 后与 Proto 服务名一致，例如 `BaseUserService` 对应 `BaseUserCase`。
- Biz 模块根目录的 `*Case` 仅允许两类：与 Proto `service` 对应的主 Case；与单张表结构和生成 Repository 一一对应、供多个业务复用的基础数据 Case。其他领域编排、策略、DTO、VO、查询结果和聚合类型分别放到已有主 Case、领域子包或 `dto` 目录，不在模块根目录新增无 Proto/无表 Repository 归属的 Case。
- 同一 Proto Service 内需要拆分的内部能力，优先使用普通领域类型或放入该业务子包；对外注册和 Wire 只暴露 Proto 对应的主 Service/Case。

## Migration
- 本仓库新增或调整内容统一归入唯一目标初始化版本：`backend/migration/assets/v0.0.1/mysql`。表结构依赖的初始化数据、菜单、按钮、`base_api` 权限、默认数据和多语言数据，直接修改该目录下的现有脚本和说明。
- 按完整初始化状态维护迁移：一次新增功能应把其所需的全部初始化内容补齐到 `v0.0.1`，保持脚本可重复执行；不要只提交面向某个已发布版本的增量补丁。
- 不新增 `v0.0.2`、`v0.0.3` 或其他 `vX.Y.Z` 迁移目录，也不为功能新增独立增量 `.up.sql`/`.down.sql`。新增语言生成迁移时显式使用 `I18N_MIGRATION_VERSION=v0.0.1`。
- 初始化脚本中的记录按表分组，并在各组内按 `id` 升序排列。`base_config` 的 ID 统一使用 `1000-1999`：site 1 使用 `1000-1099`、site 2 使用 `1100-1199`、site 3 使用 `1200-1299`，配置项之间按十位预留空间；`base_dict` 的 ID 统一使用 `1000-1999`，按配置 `code` 字典序排列，每个配置按十位预留空间；`base_dict_item` 使用 `dict_id * 10 + 序号`。`base_menu` 使用固定 8 位 `AA BB CC DD` 编码，每级两位，同一父级子节点使用 `01-99`；根节点固定为首页 `10000000`、用户管理 `30000000`、系统管理 `91000000`、开发工具 `95000000`、移动端 `99000000`，并按 ID 顺序维护菜单。调整菜单 ID 时必须同步 `parent_id`、角色 `menus` 以及多语言数据的 `target_id`。
- SQL 使用 `<feature>.up.sql`；目录说明维护在 `v0.0.1/mysql/README.md`，已有译文维护在对应 `README.<locale>.md`，与代码同一次改动完成。已执行过 `v0.0.1` 的存量数据库不会因修改初始化脚本自动重放，验证应使用全新数据库或按开发环境流程重建。

## 生成与注释
- `api/gen`、`internal/data/gen`、OpenAPI、前端 `src/rpc` 和 Wire 文件均为生成产物，禁止手改或手工添加；统一使用 `make gen`、`make gorm-gen`、`make wire` 及对应模块命令生成。
- 人工注释统一中文；每个新增或修改的方法必须补充中文方法注释。
- Go 源文件禁止新增 `// Package ...` 包级注释。
- Proto 每个 `message` 必须有中文注释；字段使用中文尾注释（如 `];  // 数量`），并同步 `(gnostic.openapi.v3.property).description`，语义一致，不在字段上方重复写同义注释。

## Biz 与代码约束
- 所有 Case 必须匿名嵌入 `*biz.BaseCase`，由构造函数通过 `baseCase *biz.BaseCase` 注入并初始化；除当前 Case 的主 Repo 外，其他 Repo 必须具名字段，例如 `baseUserRepo *data.BaseUserRepo`。
- DTO、VO、查询结果承载结构、聚合分组键等数据承载类型统一放对应模块 `dto` 目录，禁止定义在 Case 文件中。
- DTO 与 models 转换优先使用 `mapper` 包工具方法，不在业务代码写冗余类型转换。
- 优先复用 `kratos-kit`、`go-utils`、`gorm-kit` 及其子模块的已有能力；新增本地实现前必须先检查这三个库，确认无合适方案且现有能力明显不适用时才允许新增，并保持风格一致。

## 数据库查询
- 查询条件先 `Query(ctx)` 取查询对象，再用 `[]repo.QueryOption` 收敛，按需 `append` 后以 `opts...` 传入 `List/Page/Find`；方法内只有一个查询对象时变量名直接用 `query`、`opts`。
- 禁止手写运行时 SQL（`.Raw`、`db.Exec`、`gorm.Expr`、`NewUnsafeFieldRaw`、字符串式 `Where/Select/Joins/Order/Table` 等，由 `make lint` 的 forbidigo 强制）；聚合使用 gorm/gen 类型化能力。
- 复杂查询拆成“查主表编号或明细 → `IN` 查关联 → Go 侧去重、合并、分组、排序、计算”；JSON 字段筛选默认 Go 侧解析，确需数据库 JSON 函数时在代码附近说明原因。
- 确有数据库特性无法用 gorm/gen 表达时，先说明原因、影响范围和参数安全策略，封装到极小范围并加 `//nolint:forbidigo` 说明。

## 错误处理
- 顶层 `reason` 只用冻结集合：`INVALID_ARGUMENT / UNAUTHENTICATED / PERMISSION_DENIED / RESOURCE_NOT_FOUND / CONFLICT / INTERNAL_ERROR`，未经确认禁止新增。
- 对外业务错误用 `github.com/liujitcn/kratos-core/errorsx` 构造；repo 层返回原始错误，biz 层负责分类与 `message/metadata/cause`，service 层记录方法错误并以 `errorsx.WrapInternal(err, "xxx失败")` 兜底。场景映射、errorsx 方法和 metadata 键见 [服务接入指南](../docs/服务接入指南.md)。

## 数据库与 Proto 命名
- 表名、字段名全小写下划线，使用有意义的英文名词，同一概念命名一致；唯一索引使用 `unique_表名`，普通索引使用 `idx_表名_字段1_字段2`，GORM tag 同样遵守。
- package 带版本号并与目录对齐（`system.admin.v1`、`shop.app.v1` 等）；Go import 使用真实包名别名，TS import 带 `/v1/` 层级。
- HTTP 路径遵循 RESTful，格式 `/api/v1/{terminal}/{module}/{resource}`；路径修改时同步更新 Proto、权限脚本和前端请求。
- service、message、field、enum 和 HTTP 路径按当前契约直接修改，不保留旧接口兼容路径；每次契约修改都同步所有调用方、数据库初始化数据、权限脚本和生成产物。`additional_bindings`、`allow_alias` 等兼容手段需有明确协议语义，未经确认不使用。
- 字段与命名细则见 [接口参数校验设计](../docs/接口参数校验设计.md)，新增或修改 Proto 字段前必须先读。
- Proto 的 `google.api.http`、`v0.0.1/mysql` 中 `base_api.path`、前端请求地址三处必须一致。
- 枚举归属按实际调用方决定：单文件使用的 enum 放在该文件；同 package 跨文件且不属于单一业务核心的共享类型放 `common.proto`；同时被 `system.admin.v1` 和 `system.app.v1` 使用的 enum 才放更上层 shared enum 文件。Proto 不支持在 service 体内声明 enum，不得改成非法嵌套结构。
