# 脱敏全页面测试报告

## 结论

本次以本地管理端为测试对象，覆盖全部可访问管理页面首屏烟测，并覆盖当前数据库中配置的出库、入库脱敏策略。

- 管理页面烟测：51 个路由；初测 49 个通过、2 个失败，修复后失败页面对应接口均已回归通过。
- 脱敏规则模板：10 个。
- 出库策略：54 条，其中 39 条为实际按规则处理，15 条为保留原值。
- 入库策略：21 条，其中测试策略 `2004` 已停用。
- 已验证手机号、邮箱、证件号入库保护和出库掩码。
- 已验证姓名使用通用 `string_mask` 规则时出库掩码生效；唯一姓名字段已禁止配置入库脱敏，避免页面配置与运行时行为不一致。

## 测试方法

- 浏览器：本机 Chrome + Playwright。浏览器插件通道不可用，因此使用已安装 Chrome 完成页面烟测。
- 页面流程：恢复登录会话，遍历系统管理模块路由，检查页面身份、首屏内容、页面异常和接口 5xx。
- 接口流程：读取全部脱敏策略，逐条调用对应的列表、树、详情接口，检查实际字段值。
- 数据库流程：核对业务表掩码值、`base_redact_storage_value` 旁表密文记录和策略配置。
- 测试数据：使用姓名、备注和本地生成的测试账号；不使用真实个人信息。

## 全页面烟测

### 结果

初测通过 49 个页面。以下两个页面首次打开时接口返回 500，页面显示“系统内部错误”：

| 页面 | 接口 | 结果 |
| --- | --- | --- |
| `/base/data-access-log` | `PageBaseDataAccessLog` | 失败 |
| `/base/login-log` | `PageBaseLoginLog` | 失败 |

后端日志显示请求参数只有 `page_num` 和 `page_size`，查询结果包含 `tenant_id=0` 的全局日志；入库脱敏物化回调把合法的全局记录误判为“脱敏数据租户ID不能为空”。数据库中 `base_login_log` 有 9 条、`base_data_access_log` 有 50 条零租户记录。

修复后回调允许零租户全局记录跳过租户入库策略。两个接口在仍不传 `tenant_id` 的情况下分别返回 15 条和 20 条首屏数据，HTTP 状态均为 200，默认租户的跨租户查看能力保持不变。

### 已烟测路由

覆盖 AI、基础管理、日志审计、数据备份/归档、权限管理、数据脱敏、消息中心、个人中心、开发工具等 51 个路由，包括：

`/ai/chat`、`/base/api`、`/base/api-log`、`/base/area`、`/base/backup-management/*`、`/base/config`、`/base/dashboard`、`/base/data-access-log`、`/base/dept`、`/base/dict`、`/base/file`、`/base/i18n-custom`、`/base/job`、`/base/language`、`/base/login-log`、`/base/login-policy`、`/base/menu`、`/base/message`、`/base/migration`、`/base/oauth-client`、`/base/online-session`、`/base/operation-log`、`/base/permission-log`、`/base/policy-evaluation-log`、`/base/post`、`/base/redact-output-policy`、`/base/redact-rule`、`/base/redact-storage-policy`、`/base/role`、`/base/tenant-project-grant`、`/base/tenant-project`、`/base/tenant`、`/base/user`、`/message/inbox`、`/profile`、`/tool/api-doc`、`/tool/cache`、`/tool/code-gen/*`、`/tool/ops-monitoring`、`/tool/runtime-log`。

## 入库脱敏

### 手机号、邮箱、证件号

新增本地测试用户后，业务表和旁表结果如下：

- 测试用户 ID：5。
- `base_user.phone` 存储为手机号掩码。
- `base_user.email` 存储为邮箱掩码。
- `base_user.id_code` 存储为等长 `X` 掩码。
- `base_redact_storage_value` 生成 3 条密文记录，分别对应策略 ID `2001`、`2002`、`2003`。
- 列表、分页、详情接口均返回对应掩码值，原始值没有出现在响应中。

### 姓名通用规则

租户 2 配置了 `base_user.user_name` 的通用 `string_mask` 入库策略 ID `2004`。新增姓名测试用户 ID 6 后：

- 业务表仍保存原始姓名 `林晓测试用户`。
- `base_redact_storage_value` 没有生成策略 ID `2004` 的旁表密文。
- 数据库运行时跳过了该字段的入库保护。
- 同一用户的分页出库接口返回 `林*****`，出库通用掩码生效。

修复后，字段列表会排除单列或复合唯一索引中的字段，创建、更新和重新启用策略时也会再次校验。`base_user.user_name`、`base_user.user_code` 已不再出现在可配置字段中；策略 `2004` 已停用，重新启用返回 `INVALID_ARGUMENT: 唯一索引字段不支持入库脱敏`。

## 出库脱敏

39 条实际规则策略中：

- 24 条有样本并通过，其中 5 条多语言截断样本在修复后通过。
- 15 条因为业务表没有可用记录或详情 ID 不存在，暂时只能标记为无样本/阻塞。

### 已通过的规则

| 规则 | 页面/接口样本 | 结果 |
| --- | --- | --- |
| `phone_mask` | 用户列表、分页、详情 | 手机号按前三后四掩码 |
| `email_mask` | 新增测试用户列表、分页、详情 | 邮箱本地部分掩码，域名保留 |
| `fixed_length_mask` | 新增测试用户列表、分页、详情 | 证件号替换为等长 `X` |
| `string_mask` | 租户 2 用户名、操作日志资源名 | `admin -> a****`、`测试 -> 测*` |
| `ip_mask` | API 日志、登录日志、策略日志 | `::1 -> ::x` |
| `url_mask` | 当前无文件记录，未获得有效样本 | 待补样本 |
| `truncate_text` | 中文、日文和 ASCII 文本 | 按 Unicode 字符安全截断 |

### 已修复：多语言文本截断破坏 UTF-8

初测时以下页面返回了非法字符 `�`：

- 部门详情/部门树：`注册用户 -> 注册用�...`
- 国际化自定义翻译列表/详情：`すべてのプロジェクト -> すべて�...`
- 角色分页：`默认租户管理员权限模板 -> 默认租�...`

根因是依赖 `kratos-kit/redact` 的 `Truncate` 按字节切割字符串。现已改为按 `[]rune` 截断，并补充中文、日文、短文本和零长度测试。使用本地 `go.work` 加载修复后的模块回归时，上述接口均不再返回 `�`；正式集成仍需发布新的 `github.com/liujitcn/kratos-kit/redact` 版本并更新 `kratos-admin/backend/go.mod`。

### 暂无有效样本的页面

以下策略已经配置，但当前数据库没有能触发实际字段的记录，不能据此判定规则失败：

- AI 会话摘要：`ai_session.summary`
- 文件链接：`base_file.link_url`
- 消息正文：`base_message.content`
- 权限日志目标名：`base_permission_log.target_name`
- 岗位备注：`base_post.remark`
- 租户项目备注：`base_tenant_project.remark`
- OAuth 客户端 IP 白名单：`oauth_client.ip_whitelist`
- 数据访问日志资源 ID：当前记录资源 ID 为空
- 第三方账号标识：当前没有第三方账号记录

对应详情接口中，文件、消息、权限日志、OAuth 客户端分别返回 404，角色详情使用内置角色 ID 时返回保护性 409。这些属于测试数据或受保护资源限制，未计入脱敏实现失败。

## 问题清单

### P1 已修复：唯一姓名字段的入库脱敏配置不生效

复现：

1. 配置租户 2 的 `base_user.user_name` 使用 `string_mask`。
2. 新增姓名为 `林晓测试用户` 的用户。
3. 查询 `base_user` 和 `base_redact_storage_value`。

结果：业务表保留原姓名，旁表没有密文；但出库接口返回 `林*****`。

修复：采用“唯一字段仅支持出库脱敏”的边界。字段列表、创建、更新和重新启用均拒绝唯一索引字段，复合唯一索引同样生效；姓名仍可通过通用 `string_mask` 规则执行出库脱敏。

### P1 已修复：两个日志页面首屏返回 500

`/base/data-access-log` 和 `/base/login-log` 首屏会查询到 `tenant_id=0` 的全局日志，后端脱敏物化阶段此前错误要求所有记录租户ID大于零，因而返回 500。

修复：零租户全局记录不应用租户入库策略，正租户记录仍按各自租户策略恢复。回归验证两个接口不传租户筛选均返回 200。

### P1 已修复：TRUNCATE 规则按字节截断，破坏中文和日文

修复：`kratos-kit/redact` 改为按 Unicode 字符截断，并补充中、日文回归测试。本地联调通过；发布新模块版本后需更新 Admin 依赖。

### P2：多个规则缺少有效样本

文件、消息、OAuth、AI 会话、权限日志等页面当前没有可用数据，不能完成字段级脱敏断言。建议准备统一的“脱敏测试租户”和安全测试夹具，优先使用姓名、备注、资源名等非敏感字段，并统一绑定 `string_mask`。

## 后续工作

1. 发布新的 `github.com/liujitcn/kratos-kit/redact` 版本，并更新 Admin 后端依赖。
2. 为文件、消息、AI 会话、OAuth、权限日志建立姓名、备注、资源名测试夹具。
3. 发布依赖后重跑 51 页面烟测和 54 条出库策略矩阵。
