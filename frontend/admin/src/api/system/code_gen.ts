import service from "@/utils/request";
import {
  type CodeGenService,
  type CodeGenTask,
  type GetCodeGenTaskRequest,
  type PreviewCodeGenRequest,
  type PreviewCodeGenResponse,
  type RestoreCodeGenRequest,
  type StartCodeGenTaskRequest,
  type StartCodeGenTaskResponse
} from "@/rpc/system/admin/v1/code_gen";
import type { Empty } from "@/rpc/google/protobuf/empty";

const CODE_GEN_TABLE_URL = "/v1/admin/code-gen/table";
const CODE_GEN_TASK_URL = "/v1/admin/code-gen/task";
const CODE_GEN_RESTORE_URL = "/v1/admin/code-gen/restore";

/** Admin代码生成执行服务。 */
export class CodeGenServiceImpl implements CodeGenService {
  /** 预览代码生成文件。 */
  PreviewCodeGen(request: PreviewCodeGenRequest): Promise<PreviewCodeGenResponse> {
    return service<PreviewCodeGenRequest, PreviewCodeGenResponse>({
      url: `${CODE_GEN_TABLE_URL}/${request.table_id}/preview`,
      method: "post",
      data: request
    });
  }

  /** 启动异步代码生成任务。 */
  StartCodeGenTask(request: StartCodeGenTaskRequest): Promise<StartCodeGenTaskResponse> {
    return service<StartCodeGenTaskRequest, StartCodeGenTaskResponse>({
      url: CODE_GEN_TASK_URL,
      method: "post",
      data: request
    });
  }

  /** 还原代码生成结果。 */
  RestoreCodeGen(request: RestoreCodeGenRequest): Promise<Empty> {
    return service<RestoreCodeGenRequest, Empty>({
      url: CODE_GEN_RESTORE_URL,
      method: "post",
      data: request
    });
  }

  /** 查询异步代码生成任务进度。 */
  GetCodeGenTask(request: GetCodeGenTaskRequest): Promise<CodeGenTask> {
    return service<GetCodeGenTaskRequest, CodeGenTask>({
      url: `${CODE_GEN_TASK_URL}/${request.task_id}`,
      method: "get"
    });
  }
}

export const defCodeGenService = new CodeGenServiceImpl();
