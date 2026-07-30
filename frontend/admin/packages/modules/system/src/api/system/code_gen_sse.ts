import type { CodeGenTask } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/code_gen";
import { subscribeSseEvent, type SseStop } from "../base/sse";

const sseStreamCodeGen = "system.admin.codegen";
const sseEventCodeGenProgress = "codegen.progress";

/** SSE 取消订阅函数。 */
export type { SseStop };

/** 代码生成任务进度处理函数。 */
export type CodeGenProgressHandler = (task: CodeGenTask) => void;

/** 订阅指定代码生成任务的实时进度。 */
export function subscribeCodeGenProgress(taskId: string, handler: CodeGenProgressHandler): SseStop {
  return subscribeSseEvent(
    { stream: sseStreamCodeGen, channel_id: taskId },
    sseEventCodeGenProgress,
    raw => parseCodeGenProgress(raw, taskId),
    handler
  );
}

/** 解析并校验代码生成任务进度负载。 */
function parseCodeGenProgress(raw: string, taskId: string): CodeGenTask | null {
  if (!raw) return null;

  try {
    const task = JSON.parse(raw) as CodeGenTask;
    return task.task_id === taskId && Array.isArray(task.tables) ? task : null;
  } catch {
    return null;
  }
}
