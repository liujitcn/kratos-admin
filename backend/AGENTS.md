# Codex 规则（backend）

## 提交与发布
- 提交顺序：本次任务全部改动完成后统一执行一次生成与测试（`go test ./...` 必须通过）→ 检查更新 `README.md` → `git add -A` + 中文提交信息 + push 当前分支同名远程分支；改动过程中间不要逐次编辑逐次编译/测试，避免重复全量扫描。
- 本项目不编写 `_test.go`；临时测试用完即删。`go test ./...` 用于验证编译与依赖完整性。

## 新增业务流程
- 完整流程见 [docs/new-feature.md](docs/new-feature.md)，新增业务前必须先读。
- 核心顺序：需要新表则先按 `configs/data.yaml` 确认连接后把表结构真正建到开发库 → `make gorm-gen` → proto 契约 → `make gen` → service/biz → 前端；不需要新表则从 proto 开始。
- 表结构变更、菜单权限（`base_menu`）、接口权限（`base_api`）脚本统一新增版本目录到 `backend/migration/assets/vX.Y.Z/mysql`；需要区分数据源时再增加数据源子目录。目录内使用同一功能名的 `<feature>.up.sql` 和 `<feature>.description.md`，与代码同一次改动完成，禁止后补。
- `api/gen`、`internal/data/gen` 生成产物禁止手改或手工添加文件；生成一律走 `make gen`、`make gorm-gen`、`make wire`。

## 注释
- 人工注释统一中文；每个新增或修改的方法必须补充中文方法注释。
- proto：每个 `message` 必须有中文注释；字段必须有中文尾注释（如 `];  // 数量`）并同步补 `(gnostic.openapi.v3.property).description`，语义一致，不在字段上方重复写同义注释。

## 代码约束
- DTO、VO、查询结果承载结构、聚合分组键等数据承载类型统一放对应模块 `dto` 目录，禁止定义在 Case 文件中。
- 优先复用 `kratos-kit`、`go-utils`、`gorm-kit` 及其子模块的已有能力；新增本地实现前必须先检查这三个库，确认无合适方案且现有能力明显不适用时才允许新增，并保持风格一致。
- Case 依赖：仅当前 Case 的主 Repo 和 `*biz.BaseCase` 允许匿名嵌入；其他 Repo 必须具名字段，例如 `baseUserRepo *data.BaseUserRepo`。
- DTO 与 models 转换优先使用 `mapper` 包工具方法，不在业务代码写冗余类型转换。

## 数据库查询
- 查询条件先 `Query(ctx)` 取查询对象，再用 `[]repo.QueryOption` 收敛，按需 `append` 后以 `opts...` 传入 `List/Page/Find`；方法内只有一个查询对象时变量名直接用 `query`、`opts`。
- 禁止手写运行时 SQL（`.Raw`、`db.Exec`、`gorm.Expr`、`NewUnsafeFieldRaw`、字符串式 `Where/Select/Joins/Order/Table` 等，由 `make lint` 的 forbidigo 强制）；聚合用 gorm/gen 类型化能力，如 `query.Amount.Sum().IfNull(0).As("amount")`、`query.ID.Count()`。
- 复杂查询拆成“查主表编号或明细 → `IN` 查关联 → Go 侧去重、合并、分组、排序、计算”；JSON 字段筛选默认 Go 侧解析过滤，确需数据库侧 JSON 函数时在代码附近说明原因。
- 确有数据库特性无法用 gorm/gen 表达时，先说明原因、影响范围和参数安全策略，封装到极小范围并加 `//nolint:forbidigo` 说明。

## 错误处理
- 顶层 `reason` 只用 6 类冻结集合：`INVALID_ARGUMENT / UNAUTHENTICATED / PERMISSION_DENIED / RESOURCE_NOT_FOUND / CONFLICT / INTERNAL_ERROR`，未经确认禁止新增。
- 对外业务错误必须用 `github.com/liujitcn/kratos-core/pkg/errorsx` 构造，禁止直接返回 `errors.New/fmt.Errorf`；repo 层返原始错误，biz 层负责分类与 `message/metadata/cause`，service 层只 `log.Errorf("方法名 %v", err)` 并以 `errorsx.WrapInternal(err, "xxx失败")` 兜底透传。
- 场景映射、errorsx 方法、metadata 键等细则见 [docs/errors.md](docs/errors.md)，修改错误处理相关代码前必须先读。

## 数据库命名
- 表名、字段名全小写下划线，使用有意义的英文名词，同一概念命名一致。
- 唯一索引 `unique_表名`；普通索引 `idx_表名_字段1_字段2`；GORM tag 中同样遵守，例如 `uniqueIndex:unique_order,priority:1`。

## proto 与 HTTP 路径
- package 带版本号并与目录对齐（`system.admin.v1`、`shop.app.v1` 等）；Go import 用真实包名别名（`shopadminv1` 等）；TS import 带 `/v1/` 层级。
- HTTP 路径遵循 RESTful，格式 `/api/v1/{terminal}/{module}/{resource}`；本仓库不保留旧接口兼容路径，路径修改时同步更新 Proto、权限脚本和前端请求。
- service、message、field、enum 和 HTTP 路径均不做向后兼容；每次契约修改都必须同步所有调用方、数据库数据、权限脚本和生成产物，禁止使用 `additional_bindings`、`allow_alias` 等兼容手段。协议确需为同一 RPC 映射多个 HTTP 方法时，`additional_bindings` 只按协议语义使用。
- 字段与命名细则见 [docs/api.md](docs/api.md)，新增或修改 proto 字段前必须先读
- proto 的 `google.api.http`、对应版本目录 `up.sql` 的 `base_api.path`、前端请求地址三处必须一致，禁止只改其中一处。
- 枚举归属按实际调用方决定：只被一个业务 Proto 文件使用的 enum，放在该文件中，紧随对应 service 定义并与 message 同级；同一 package 跨多个业务文件使用、且不属于单一业务核心的共享类型，放在该 package 的 `common.proto`；同一端多个业务文件使用但有明确核心 service/message 归属的 enum，放在主文件，其他文件显式 import；同时被 `system.admin.v1` 和 `system.app.v1` 使用的 enum，才放入更上层 shared enum 文件。Proto 不支持在 service 体内直接声明 enum，不得为了“放在 service 下”改成非法嵌套结构。
