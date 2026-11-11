import service from "@liujitcn/kratos-admin-core/request";
import type {
  AiFile,
  AiFileService,
  DeleteAiFileRequest,
  PageAiFileRequest,
  PageAiFileResponse,
  RenameAiFileRequest,
  RestoreAiFileRequest,
  TrashAiFileRequest
} from "@liujitcn/kratos-admin-system/rpc/base/v1/ai_file";
import type { Empty } from "@liujitcn/kratos-admin-system/rpc/google/protobuf/empty";

const AI_FILE_URL = "/v1/base/ai/file";

/** AI 助手文件服务。 */
export class AiFileServiceImpl implements AiFileService {
  /** 分页查询 AI 文件。 */
  PageAiFile(request: PageAiFileRequest): Promise<PageAiFileResponse> {
    return service<PageAiFileRequest, PageAiFileResponse>({ url: AI_FILE_URL, method: "get", params: request });
  }

  /** 重命名 AI 文件。 */
  RenameAiFile(request: RenameAiFileRequest): Promise<AiFile> {
    return service<RenameAiFileRequest, AiFile>({ url: `${AI_FILE_URL}/${request.id}/name`, method: "put", data: request });
  }

  /** 将 AI 文件移入回收站。 */
  TrashAiFile(request: TrashAiFileRequest): Promise<AiFile> {
    return service<TrashAiFileRequest, AiFile>({ url: `${AI_FILE_URL}/${request.id}/trash`, method: "put", data: request });
  }

  /** 从回收站恢复 AI 文件。 */
  RestoreAiFile(request: RestoreAiFileRequest): Promise<AiFile> {
    return service<RestoreAiFileRequest, AiFile>({ url: `${AI_FILE_URL}/${request.id}/restore`, method: "put", data: request });
  }

  /** 彻底删除 AI 文件。 */
  DeleteAiFile(request: DeleteAiFileRequest): Promise<Empty> {
    return service<DeleteAiFileRequest, Empty>({ url: `${AI_FILE_URL}/${request.id}`, method: "delete", params: request });
  }
}

export const defAiFileService = new AiFileServiceImpl();
