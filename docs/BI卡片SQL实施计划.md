# BI 卡片 SQL 与公共指标实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans，按任务顺序在当前会话完成。

**Goal:** 卡片持久化参数化 SQL，公共指标公式统一定义、原子同步，并阻止旧发布口径继续取数。

**Architecture:** 复用现有 BI 表、Service、Case 与 Repository。领域编译器生成执行定义，数据源业务统一同步派生卡片；预览执行在事务外，元数据更新在一个事务内。

**Tech Stack:** Go 1.27.3、Kratos、GORM/gen、Proto、Vue、Element Plus、现有 CodeEditor。

**Spec:** [BI卡片SQL与公共指标详细设计](BI卡片SQL与公共指标详细设计.md)

## 全局约束

- 不增加数据集/指标表或新依赖；只允许编辑指标公式，完整 SQL 只读。
- 保留租户、项目、宿主连接隔离及 15 秒、1000 行、48 KiB 限制。
- 生成产物只由项目命令生成；初始化统一维护 v0.0.1；不清除现有资产。
- 使用当前检出，不提交、合并或发布；已有设计文档改动保留。

## 重点检查

- 删除并重建同编码指标后，旧预览和发布快照不能重获执行资格。
- 同一预览并发/重复提交只能应用一次，失败写入必须整体回滚。
- 系统级联更新保留被更新卡片的原始来源和人工预览，不借用操作者的来源。
- 所有仪表盘响应及工作台必须通过展示 DTO，不能泄漏 SQL、公式或参数上下文。
- 在途取数、显式历史版本、密码和画布更新分别按实际执行修订校验。

## 任务 1：契约和存储

**Files:** `backend/api/proto/bi/admin/v1/*.proto`、开发库 BI 表、`backend/migration/assets/v0.0.1/mysql`。

**Interfaces:** 固定指标 `code/name/type/expression/unit/revision`；新增 `GetBiCard`、`PreviewBiCardSql`、`PreviewBiSourceModel`；保存请求包含预期源/卡片版本、预览 ID 和候选指标。

- [x] 扩展开发库字段和索引，保持已有行，生成 GORM 模型和查询。
- [x] 修改 Proto 和共享领域类型，移除查询级 aggregation，同步全部调用方和 fixture。
- [x] 统一生成 API、OpenAPI、Wire 和前端 RPC；最后核对权限/路由一致。

## 任务 2：受控公式和执行定义

**Files:** 新增 `backend/internal/biz/bi/admin/query`；修改 `dto/source.go`、`dto/query.go`、`sql_source.go`。

**Interfaces:** `query.Parse(expression string, fields map[string]bi.FieldType) (*query.Expression, error)`；`Expression.Canonical() string`、`Expression.SQL(quote func(string) string) string`；`dto.CardExecution` 为唯一持久化执行定义。

- [x] 先写并运行表达式拒绝注入/裸字段/聚合嵌套/超界、精确除法和规范化测试，确认失败。
- [x] 实现标准库 scanner 表达式树、类型校验和安全渲染，运行领域测试至通过。
- [x] 生成稳定 SQL、参数来源、修订、结果字段和哈希；执行时绑定身份、筛选、相对时间与 range_days。
- [x] 验证 MySQL/PostgreSQL 方言和参数顺序，以及十进制/NULL 结果约定。

## 任务 3：公共模型同步

**Files:** `bi_source.go`、领域同步实现、`internal/data/bi_*.go`、相应测试。

**Interfaces:** 数据源业务统一提供编译、执行定义检查、候选修订计算和事务内同源卡片同步；不反向依赖 CardCase。

- [x] 测试修订单调、同名唯一、结构与画布变更、删除引用冲突。
- [x] 实现模型预览与保存；事务按源→预览→卡片→仪表盘顺序锁定，校验全部后统一写入。
- [x] 测试多卡同步保留各自筛选/分组/来源、写入回滚和并发过期。
- [x] 补充模型管理权限，源编码不可变及删除/连接变更依赖检查。

## 任务 4：卡片预览、保存与执行记录

**Files:** `bi_card.go`、`bi_query.go`、`bi_query_record.go`、现有 Service 及测试。

**Interfaces:** `PreviewBiCardSql` 冻结精确候选；`SaveBiCard` 原子消费预览并返回同步摘要；历史记录不变。

- [x] 测试跨用户/项目、普通预览冒用、重复消费、修改后未重预览被拒绝。
- [x] 实现详情权限、公式预览和事务保存，保留原始问数与最后人工预览。
- [x] 展示保存使用当前派生字段校验；取数执行保存 SQL 并在返回前复核修订。
- [x] 调整 AI 提示、Adapter、查询哈希和记录分类，验证固定指标查询闭环。

## 任务 5：发布快照与响应过滤

**Files:** `bi_dashboard.go`、`dto/dashboard.go`、工作台及对应测试。

**Interfaces:** 内部快照 schemaVersion=4，独立 allowlist 展示布局；`requires_republish` 动态判断。

- [x] 测试当前/历史发布过期整盘拒绝、在途更新丢弃、响应中无内部执行内容。
- [x] 发布固定当前执行定义和来源，系统级联不要求旧人工预览哈希一致。
- [x] 列表/详情/保存/发布/取数/工作台共用安全展示转换，取数前后验证整份快照。

## 任务 6：管理端、文档与收尾

**Files:** BI 模块 API、卡片/模型/仪表盘组件、语言包、测试、README 和初始化权限数据。

- [x] 移除聚合选择，复用 CodeEditor 实现公共公式编辑、SQL 只读、影响预览和版本冲突保留输入。
- [x] 数据源保存先显示模型影响，仪表盘展示待重新发布状态。
- [x] 补充关键交互行为测试、四语言文案和权限，更新文档的实际状态。
- [x] 执行设计中的生成、i18n、菜单排序、Go 全量测试/lint、前端 lint/typecheck/BI 测试并处理失败。
- [x] 独立复核全部未提交改动与安全边界，交付变更及实际验证范围。

## 交付验证记录

- GORM、API、OpenAPI、Wire 和管理端 RPC 已通过项目命令生成；菜单排序与国际化检查通过。
- 后端 `go test ./...` 通过，`make -C backend lint` 为 `0 issues`。
- 管理端 BI 行为测试 52/52、类型检查 7/7、生产构建 5/5 通过；oxlint 退出 0，保留 System 精度常量与 Core triple-slash 两条既有警告。
- 管理端全量 `pnpm test` 在 System 测试阶段有 4 项既有失败：引用缺失的 uni-app `pagesMember/ai/index.vue`、`pagesMember/message/index.vue`、`pagesMember/message/detail.vue`。未创建无关页面或修改这些测试；BI 测试单独运行通过。
- MySQL、PostgreSQL 17、TiDB v8.5.8、Doris 3.0.8 的真实驱动验证通过，覆盖租户条件、IN 参数、日期周期覆盖、精确金额/计数除法和 NULL；验证使用的临时库、账号、容器及临时测试均已清理。
- 独立只读审查已完成，4 项重要发现均已补回归、观察失败并修复通过：列表重新发布读取完整布局；Adapter 冻结并执行有效行数上限；跨源迟到响应与状态隔离；保存结果未知时冻结原请求并重试相同预览。未遗留本轮审查的次要建议。
- 源行数上限变更的后续回归已从失败修复至通过：降低及提高上限后，重新发布取数、展示保存、公式预览保留原查询身份，结果继续遵守实际执行上限；同步覆盖默认上限、硬上限和 Adapter 边界。
- 同页列表改名原有空布局问题一并修复，读取完整详情后合并名称，继续使用列表中的草稿版本校验，避免丢失卡片和覆盖并发编辑。
- 生成阶段的明文数据库临时配置已删除；开发库仅调整本任务字段、容量与索引，现存行保留。未提交、合并或发布。

## 交付边界

- 只支持受控公共指标公式；完整 SQL 只读。维度、筛选和关联继续使用现有模型配置，不新增任意 SQL 编辑器或指标管理表。
- 旧指标结构、缺少执行定义的旧卡片和布局 v3 不自动转换；启用前需重新建模、保存符合新契约的卡片并重新发布仪表盘。已有 v0.0.1 环境须按现有流程补齐新增菜单授权，初始化脚本不会自动重放。
- 本次完成独立代码审查及 SFC 行为验证；未进行浏览器内人工页面验收。审查后的修复使用失败到通过的回归和完整检查验证。
