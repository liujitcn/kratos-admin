# 经典工作流示例

五组配置可用于流程设计器和流程定义 API 的配置参考。每个 JSON 都包含基本信息、申请表单及 version 2 节点图，节点位置适配 Vue Flow 画布。

| 示例 | 配置 | 路径与预期行为 |
| --- | --- | --- |
| 请假串行审批 | [leave-serial.json](leave-serial.json) | 开始 → 主管 → 人事 → 结束，前一级通过后才产生下一级待办 |
| 报销金额分支 | [expense-exclusive.json](expense-exclusive.json) | 开始 → 主管 → 金额判断；≤1000 元结束，>1000 元增加财务审批 |
| 合同会签 | [contract-all.json](contract-all.json) | 开始 → 法务、财务会签（ALL）→ 结束，必须全部通过 |
| 值班或签 | [duty-any.json](duty-any.json) | 开始 → 两位值班负责人或签（ANY）→ 结束，任一通过即取消其余待办 |
| 采购并行汇聚 | [purchase-parallel.json](purchase-parallel.json) | 开始 → 财务、法务并行 → 汇聚 → 经理 → 结束，两个分支均完成才进入经理审批 |

任一待办先驳回时整单驳回，包括或签；申请人可以撤回审批中的实例。终结后不能继续办理任何已完成或已取消任务。

## 示例数据与表单

示例中的用户 ID（1、2、3、4）是占位值。导入其他环境前，必须把节点 `approvers` 中的 ID 换成本租户的真实审批人；申请人本人作为审批人时，引擎会自动跳过本人。

申请表单控制在 3～4 个字段，涵盖文本、多行文本、数字、日期和下拉选择。金额分支使用数字字段 `amount`，条件为 `amount > 1000`。`select.options` 使用逗号分隔字符串，与申请页面解析方式一致。

JSON 中的 `form_schema`、`node_config` 便于阅读，分别保存为数组和对象。通过流程定义 API 创建时，需要将这两项分别序列化为 JSON 字符串。新建定义默认保存为草稿，配置审批人后再启用。

## 使用方式

将 JSON 导入开发环境的流程设计器，或经流程定义 API 创建流程。

## 脚本流程示例

[script-expense](script-expense/README.md) 提供提交校验、变量计算、动态审批人、金额网关与三类终态业务动作的配置示例。
