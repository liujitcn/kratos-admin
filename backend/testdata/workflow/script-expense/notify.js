// 内置动作复用现有站内信模块，无需宿主注册处理器。
// 申请人来自引擎可信上下文，受众仍会校验本租户启用状态。
({actions: [{type: "workflow.notify", payload: {
  title: "报销审批结果",
  content: "报销流程已结束，流程实例：" + ctx.workflow.instance_id,
  user_ids: [ctx.workflow.applicant_user_id]
}}]})
