import type { CodeGenTask } from "../rpc/system/admin/v1/code_gen";

/** 补齐 ProtoJSON 省略的零值计数，统一 HTTP 与 SSE 任务快照。 */
export function normalizeCodeGenTask(task: CodeGenTask): CodeGenTask {
  return {
    ...task,
    completed_steps: task.completed_steps ?? 0,
    total_steps: task.total_steps ?? 0,
    tables: (task.tables ?? []).map(table => ({
      ...table,
      completed_steps: table.completed_steps ?? 0,
      total_steps: table.total_steps ?? 0
    }))
  };
}
