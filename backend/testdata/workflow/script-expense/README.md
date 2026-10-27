# 脚本报销闭环

先在脚本管理中创建并启用以下 JAVASCRIPT 脚本，再将 definition.json 中的示例编号替换成实际编号。共享脚本库没有租户字段，流程定义、实例与执行记录仍按租户隔离。

| 示例编号 | 文件 | 用途 |
| --- | --- | --- |
| 101 | before-submit.js | 非正金额拒绝申请，不创建实例 |
| 102 | calculate.js | 计算 variables.total |
| 103 | approvers.js | 财务审批人，替换为本租户真实用户 |
| 104 | approved.js | 通过后请求 expense.approve 动作 |
| 105 | terminal.js | 驳回、撤回后请求 expense.close 动作 |

将 definition.json 的基本信息、表单与节点配置录入流程定义并启用。启用时服务端重建脚本快照，客户端无需提交 script_snapshots。1000 元及以下自动通过；大于 1000 元进入动态财务审批。脚本更新不会改变已发布定义或既有实例，应发布新的定义版本。

宿主必须先注册 expense.approve 和 expense.close，参见 docs/工作流接入指南.md。故意让动作首次返回错误，可在审批详情检查 FAILED、已完成动作进度与重试；分钟扫描或人工重试成功后状态变为 SUCCEEDED，审批结果不回滚。不要用旧式终结回调再次执行同一扣款或记账副作用。

测试命令（backend 目录）：

```sh
go test ./internal/biz/system/admin -run 'TestWorkflowScript' -count=1
```

## 无需宿主注册的通知示例

[notify.js](notify.js) 可直接作为已通过、已驳回或已撤回钩子脚本：在脚本管理创建启用的 JAVASCRIPT 脚本，将真实脚本 ID 绑定对应终态钩子，启用流程后提交审批。它输出内置 `workflow.notify`，复用现有站内信模块，向申请人发布 `WORKFLOW` 类别通知，无需 `RegisterAction`。受众必须是当前租户启用用户。测试使用 fake Publisher，不向真实用户发消息。

`approved.js` 中的 `expense.approve` 仍是宿主自定义业务动作，需按接入指南注册并实现事务幂等；不能把该文件当作开箱即用的记账能力。若只需验证内置通知，将 approved/rejected/withdrawn 钩子都替换为 notify.js 的脚本 ID 即可。
