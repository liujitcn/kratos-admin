# 错误处理细则

> 修改错误处理相关代码前必读。核心约束见 `backend/AGENTS.md` 的「错误处理」节。

## 设计目标

错误体系只解决 3 件事：让调用方稳定判断少量必要场景、让用户看到合适提示文案、让日志保留足够排障信息。禁止恢复"按文案拆枚举""按资源拆枚举""按场景拆枚举"的旧做法。

各部分职责：
- `reason`：稳定的一级分类（仅 6 类，冻结集合）。
- `message`：面向用户的提示文案。
- `metadata`：调用方确有需要时的二级结构化信息。
- `cause`、日志与链路信息：保留底层技术细节。

禁止把每一句中文提示、每一种资源类型或每一个细分场景都扩展成新的顶层 `reason`。

## 6 类顶层 reason（冻结集合）

| reason | HTTP | 语义 | 典型场景 |
|---|---|---|---|
| `INVALID_ARGUMENT` | 400 | 请求本身有问题 | 参数为空、格式错误、取值非法、验证码错误、角色 ID 列表为空；能通过修改请求立即解决的问题 |
| `UNAUTHENTICATED` | 401 | 身份不成立 | 用户名或密码错误、未登录、token 无效/过期、刷新令牌无效；登录场景中的"用户不存在"也归此类（避免账号探测） |
| `PERMISSION_DENIED` | 403 | 身份成立但不允许 | 账号禁用、角色禁用、无权限、不能操作超级管理员/超管角色 |
| `RESOURCE_NOT_FOUND` | 404 | 目标资源不存在 | 用户/角色/部门/定时任务/调用目标不存在；登录失败不要归此类 |
| `CONFLICT` | 409 | 资源存在但当前状态不允许 | 唯一约束冲突、存在子节点不能删除、状态不允许变更、资源受保护、重复提交或任务状态不允许变更 |
| `INTERNAL_ERROR` | 500 | 其余内部异常唯一兜底 | 数据库/Redis/RPC/第三方错误、配置错误、数据不一致、文件系统错误，以及"按当前请求内容无法直接修复"的异常 |

新增顶层 reason 必须同时满足 4 个条件，缺一不可：现有 6 类无法准确表达一级语义；调用方存在明确稳定的跨页面/跨端分支需求；该需求不能通过 `reason + metadata` 表达；该分类具备长期稳定性。禁止在 `error.proto` 中追加 `TOKEN_EXPIRED`、`USER_NOT_FOUND`、`STATE_CONFLICT`、`UNIQUE_VIOLATION` 这类旧风格或派生风格枚举。

## 分层职责

- `repo` 层、数据库访问层、第三方 SDK 层：返回原始错误，不做对外错误分类，不拼面向前端的业务提示。
- `biz` 层：负责错误分类、对外 `message`、必要的 `metadata` 与 `cause`。
- `service` 层：不做业务分类判断，只打印原始日志和统一兜底包装：
  - 日志：直接在当前方法内 `log.Errorf("方法名 %v", err)`；禁止封装新的日志 helper；禁止 `log.Error("xx err:", err.Error())` 这类产生 `err:error:` 冗余前缀的写法。
  - 兜底：`errorsx.WrapInternal(err, "xxx失败")`；若 biz 层已返回结构化错误，`WrapInternal` 直接透传，禁止二次改写已有 `reason`。

## errorsx 构造方法

- 对外业务错误禁止直接返回 `errors.New(...)`、`fmt.Errorf(...)`，必须用项目的 `pkg/errorsx` 统一构造。
- 唯一约束冲突：`errorsx.UniqueConflict(...)`；识别 MySQL 唯一键冲突用 `errorsx.IsMySQLDuplicateKey(err)`，不要在业务代码重复写错误码判断。
- 存在子资源不能删除：`errorsx.HasChildrenConflict(...)`。
- 状态不匹配：`errorsx.StateConflict(...)`。
- 受保护资源：`errorsx.ProtectedResourceConflict(...)`。
- 转换底层错误时必须保留错误链：`.WithCause(err)`；禁止只返回新的中文文案而丢掉原始错误。

## metadata 使用约束

- 默认优先 `reason + message`；只有调用方确实需要在同一 reason 下做稳定分支时才补充 `metadata`。
- 冲突类错误优先复用已定义的元数据键：`conflict_type`、`resource`、`field`、`constraint`、`child_resource`、`current_state`、`expected_state`；不要随意新增命名分散的 key。
- 禁止把整句中文提示复制进 `metadata`，禁止无限扩展 metadata 键把它变成第二套枚举系统。

## 前端协作约定

前端是否展示弹窗、表单提示、跳转登录，必须由 `code/reason/metadata` 决定；后端不要新增依赖中文 `message` 才能稳定分支的协议约定。
