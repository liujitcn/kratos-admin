/** 流程定义向导步骤键。 */
export const WORKFLOW_WIZARD_STEP_KEYS = ["basic", "form", "graph", "review"] as const;

/** 流程定义向导步骤键类型。 */
export type WorkflowWizardStepKey = (typeof WORKFLOW_WIZARD_STEP_KEYS)[number];

/** 流程定义向导校验状态。 */
export type WorkflowWizardStepStatus = "pending" | "success" | "error";

/** Element Plus 步骤组件使用的展示状态。 */
export type WorkflowWizardRenderStatus = "wait" | "process" | "finish" | "error";

/** 判断当前步骤是否允许进入下一步。 */
export function canAdvanceWorkflowWizard(activeStep: number, status: WorkflowWizardStepStatus): boolean {
  return activeStep < WORKFLOW_WIZARD_STEP_KEYS.length - 1 && status === "success";
}

/** 将业务校验状态映射为 Element Plus 步骤展示状态。 */
export function getWorkflowWizardStepStatus(
  step: number,
  activeStep: number,
  status: WorkflowWizardStepStatus
): WorkflowWizardRenderStatus {
  if (status === "error") return "error";
  if (step === activeStep) return "process";
  if (status === "success") return "finish";
  return "wait";
}
