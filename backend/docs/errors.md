# 错误处理

对外错误由 `backend/core/pkg/errorsx` 构造，顶层 `reason` 只使用以下六类：

| reason | HTTP | 使用场景 |
| --- | --- | --- |
| `INVALID_ARGUMENT` | 400 | 请求格式或业务参数无效。 |
| `UNAUTHENTICATED` | 401 | 未登录、Token 或凭据无效。 |
| `PERMISSION_DENIED` | 403 | 已认证但无权操作。 |
| `RESOURCE_NOT_FOUND` | 404 | 目标资源不存在。 |
| `CONFLICT` | 409 | 唯一约束、子资源、状态或受保护资源冲突。 |
| `INTERNAL_ERROR` | 500 | 未分类的内部故障。 |

## 分层职责

- Repository 返回 GORM、MySQL 或依赖库的原始错误。
- biz 把可预期错误转换成 `errorsx`，补充用户可读中文消息、metadata 和 cause。
- service 记录方法名与错误，并用 `errorsx.WrapInternal(err, "操作失败")` 兜底；已分类错误会原样透传。
- 前端以 HTTP code 和 `reason` 判断认证、权限等流程，直接展示服务端 `message`，不解析错误文本推断类型。

## 构造方法

基础方法：`InvalidArgument`、`Unauthenticated`、`PermissionDenied`、`ResourceNotFound`、`Conflict`、`Internal`。

冲突方法：

| 方法 | `conflict_type` |
| --- | --- |
| `UniqueConflict` | `unique_violation` |
| `HasChildrenConflict` | `has_children` |
| `StateConflict` | `state_conflict` |
| `ProtectedResourceConflict` | `protected_resource` |

可用 metadata 键只有 `conflict_type`、`resource`、`field`、`constraint`、`child_resource`、`current_state`、`expected_state`。新增键或 reason 前需要先扩展公共契约和前端消费逻辑。

`WrapInternal` 会把 `gorm.ErrRecordNotFound` 转成 `RESOURCE_NOT_FOUND`，其他未分类错误转成 `INTERNAL_ERROR`。MySQL 1062 可先用 `IsMySQLDuplicateKey` 识别，再返回 `UniqueConflict`。

不得直接向客户端返回数据库错误文本、SQL、密钥或第三方响应详情。
