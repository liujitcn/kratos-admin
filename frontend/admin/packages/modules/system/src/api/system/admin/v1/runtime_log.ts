import service from "@liujitcn/kratos-admin-core/request";
import type {
  DownloadRuntimeLogFileRequest,
  ListRuntimeLogFilesRequest,
  ListRuntimeLogFilesResponse,
  OpenRuntimeConsoleRequest,
  OpenRuntimeConsoleResponse,
  ReadRuntimeLogFileRequest,
  ReadRuntimeLogFileResponse
} from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/runtime_log";

const RUNTIME_LOG_URL = "/v1/admin/runtime-log";

/** Admin运行日志服务。 */
export class RuntimeLogServiceImpl {
  /** 查询历史日志文件列表。 */
  ListRuntimeLogFiles(request: ListRuntimeLogFilesRequest): Promise<ListRuntimeLogFilesResponse> {
    return service<ListRuntimeLogFilesRequest, ListRuntimeLogFilesResponse>({
      url: `${RUNTIME_LOG_URL}/files`,
      method: "get",
      params: request
    });
  }

  /** 分页读取历史日志文件内容。 */
  ReadRuntimeLogFile(request: ReadRuntimeLogFileRequest): Promise<ReadRuntimeLogFileResponse> {
    return service<ReadRuntimeLogFileRequest, ReadRuntimeLogFileResponse>({
      url: `${RUNTIME_LOG_URL}/files/${encodeURIComponent(request.file_id)}`,
      method: "get",
      params: {
        cursor: request.cursor,
        limit: request.limit,
        keyword: request.keyword,
        levels: request.levels
      }
    });
  }

  /** 创建当前用户的实时控制台频道。 */
  OpenRuntimeConsole(request: OpenRuntimeConsoleRequest): Promise<OpenRuntimeConsoleResponse> {
    return service<OpenRuntimeConsoleRequest, OpenRuntimeConsoleResponse>({
      url: `${RUNTIME_LOG_URL}/console`,
      method: "post",
      data: request
    });
  }

  /** 下载历史日志原文件。 */
  async DownloadRuntimeLogFile(request: DownloadRuntimeLogFileRequest, fallbackName: string): Promise<void> {
    const response = await service<DownloadRuntimeLogFileRequest, any>({
      url: `${RUNTIME_LOG_URL}/files/${encodeURIComponent(request.file_id)}/download`,
      method: "get",
      responseType: "blob"
    });
    const contentDisposition = String(response.headers?.["content-disposition"] ?? "");
    const encodedName = contentDisposition.match(/filename\*=UTF-8''([^;]+)/i)?.[1];
    const plainName = contentDisposition.match(/filename="?([^";]+)"?/i)?.[1];
    let filename = fallbackName;
    if (encodedName) {
      try {
        filename = decodeURIComponent(encodedName);
      } catch {
        filename = fallbackName;
      }
    } else if (plainName) {
      filename = plainName;
    }
    const blob = response.data instanceof Blob ? response.data : new Blob([response.data]);
    const objectURL = window.URL.createObjectURL(blob);
    const anchor = document.createElement("a");
    anchor.href = objectURL;
    anchor.download = filename;
    document.body.appendChild(anchor);
    anchor.click();
    anchor.remove();
    window.URL.revokeObjectURL(objectURL);
  }
}

export const defRuntimeLogService = new RuntimeLogServiceImpl();
