import service from "@liujitcn/kratos-admin-core/request";
import type {
  BaseScriptForm,
  BaseScriptService,
  CreateBaseScriptRequest,
  DeleteBaseScriptRequest,
  GetBaseScriptRequest,
  PageBaseScriptRequest,
  PageBaseScriptResponse,
  SetBaseScriptStatusRequest,
  TestRunBaseScriptRequest,
  TestRunBaseScriptResponse,
  UpdateBaseScriptRequest
} from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_script";
import type { Empty } from "@liujitcn/kratos-admin-system/rpc/google/protobuf/empty";

const BASE_SCRIPT_URL = "/v1/admin/base/script";

/** Admin脚本管理服务。 */
export class BaseScriptServiceImpl implements BaseScriptService {
  /** 查询脚本分页列表。 */
  PageBaseScript(request: PageBaseScriptRequest): Promise<PageBaseScriptResponse> {
    return service<PageBaseScriptRequest, PageBaseScriptResponse>({ url: BASE_SCRIPT_URL, method: "get", params: request });
  }

  /** 查询脚本详情。 */
  GetBaseScript(request: GetBaseScriptRequest): Promise<BaseScriptForm> {
    return service<GetBaseScriptRequest, BaseScriptForm>({ url: `${BASE_SCRIPT_URL}/${request.id}`, method: "get" });
  }

  /** 创建脚本。 */
  CreateBaseScript(request: CreateBaseScriptRequest): Promise<Empty> {
    return service<BaseScriptForm | undefined, Empty>({ url: BASE_SCRIPT_URL, method: "post", data: request.base_script });
  }

  /** 更新脚本。 */
  UpdateBaseScript(request: UpdateBaseScriptRequest): Promise<Empty> {
    return service<BaseScriptForm | undefined, Empty>({ url: `${BASE_SCRIPT_URL}/${request.base_script?.id ?? ""}`, method: "put", data: request.base_script });
  }

  /** 删除脚本。 */
  DeleteBaseScript(request: DeleteBaseScriptRequest): Promise<Empty> {
    return service<DeleteBaseScriptRequest, Empty>({ url: `${BASE_SCRIPT_URL}/${request.id}`, method: "delete" });
  }

  /** 设置脚本状态。 */
  SetBaseScriptStatus(request: SetBaseScriptStatusRequest): Promise<Empty> {
    return service<SetBaseScriptStatusRequest, Empty>({ url: `${BASE_SCRIPT_URL}/${request.id}/status`, method: "put", data: request });
  }

  /** 试运行脚本。 */
  TestRunBaseScript(request: TestRunBaseScriptRequest): Promise<TestRunBaseScriptResponse> {
    return service<TestRunBaseScriptRequest, TestRunBaseScriptResponse>({ url: `${BASE_SCRIPT_URL}/test-run`, method: "post", data: request });
  }
}

export const defBaseScriptService = new BaseScriptServiceImpl();
