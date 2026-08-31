import service from "@liujitcn/kratos-admin-core/request";
import type {
  BaseFile,
  BaseFileService,
  DeleteBaseFileRequest,
  GetBaseFileRequest,
  PageBaseFileRequest,
  PageBaseFileResponse
} from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_file";
import type { Empty } from "@liujitcn/kratos-admin-system/rpc/google/protobuf/empty";

const BASE_FILE_URL = "/v1/admin/base/file";

/** Admin文件资产管理服务。 */
export class BaseFileServiceImpl implements BaseFileService {
  /** 分页查询文件资产。 */
  PageBaseFile(request: PageBaseFileRequest): Promise<PageBaseFileResponse> {
    return service<PageBaseFileRequest, PageBaseFileResponse>({
      url: BASE_FILE_URL,
      method: "get",
      params: request
    });
  }

  /** 查询文件资产详情。 */
  GetBaseFile(request: GetBaseFileRequest): Promise<BaseFile> {
    return service<GetBaseFileRequest, BaseFile>({
      url: `${BASE_FILE_URL}/${request.id}`,
      method: "get"
    });
  }

  /** 删除文件资产及其对象。 */
  DeleteBaseFile(request: DeleteBaseFileRequest): Promise<Empty> {
    return service<DeleteBaseFileRequest, Empty>({
      url: `${BASE_FILE_URL}/${request.id}`,
      method: "delete"
    });
  }
}

/** 默认文件资产管理服务实例。 */
export const defBaseFileService = new BaseFileServiceImpl();
