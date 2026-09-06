import service from "@liujitcn/kratos-admin-core/request";
import type {
  BaseRedactStoragePolicyForm,
  BaseRedactStoragePolicyService,
  CreateBaseRedactStoragePolicyRequest,
  DeleteBaseRedactStoragePolicyRequest,
  GetBaseRedactStoragePolicyRequest,
  PageBaseRedactStoragePolicyRequest,
  PageBaseRedactStoragePolicyResponse,
  SetBaseRedactStoragePolicyStatusRequest,
  UpdateBaseRedactStoragePolicyRequest
} from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_redact_storage_policy";
import type { Empty } from "@liujitcn/kratos-admin-system/rpc/google/protobuf/empty";

const BASE_REDACT_STORAGE_POLICY_URL = "/v1/admin/base/redact-storage-policy";

/** 管理端入库脱敏策略服务。 */
export class BaseRedactStoragePolicyServiceImpl implements BaseRedactStoragePolicyService {
  /** 查询入库脱敏策略分页列表。 */
  PageBaseRedactStoragePolicy(request: PageBaseRedactStoragePolicyRequest): Promise<PageBaseRedactStoragePolicyResponse> {
    return service<PageBaseRedactStoragePolicyRequest, PageBaseRedactStoragePolicyResponse>({ url: BASE_REDACT_STORAGE_POLICY_URL, method: "get", params: request });
  }

  /** 查询入库脱敏策略详情。 */
  GetBaseRedactStoragePolicy(request: GetBaseRedactStoragePolicyRequest): Promise<BaseRedactStoragePolicyForm> {
    return service<GetBaseRedactStoragePolicyRequest, BaseRedactStoragePolicyForm>({ url: `${BASE_REDACT_STORAGE_POLICY_URL}/${request.id}`, method: "get" });
  }

  /** 批量创建入库脱敏策略。 */
  CreateBaseRedactStoragePolicy(request: CreateBaseRedactStoragePolicyRequest): Promise<Empty> {
    return service<CreateBaseRedactStoragePolicyRequest, Empty>({ url: BASE_REDACT_STORAGE_POLICY_URL, method: "post", data: request });
  }

  /** 批量更新入库脱敏策略。 */
  UpdateBaseRedactStoragePolicy(request: UpdateBaseRedactStoragePolicyRequest): Promise<Empty> {
    return service<UpdateBaseRedactStoragePolicyRequest, Empty>({ url: BASE_REDACT_STORAGE_POLICY_URL, method: "put", data: request });
  }

  /** 删除入库脱敏策略。 */
  DeleteBaseRedactStoragePolicy(request: DeleteBaseRedactStoragePolicyRequest): Promise<Empty> {
    return service<DeleteBaseRedactStoragePolicyRequest, Empty>({ url: `${BASE_REDACT_STORAGE_POLICY_URL}/${request.id}`, method: "delete" });
  }

  /** 设置入库脱敏策略状态。 */
  SetBaseRedactStoragePolicyStatus(request: SetBaseRedactStoragePolicyStatusRequest): Promise<Empty> {
    return service<SetBaseRedactStoragePolicyStatusRequest, Empty>({ url: `${BASE_REDACT_STORAGE_POLICY_URL}/${request.id}/status`, method: "put", data: request });
  }
}

/** defBaseRedactStoragePolicyService 入库脱敏策略服务实例。 */
export const defBaseRedactStoragePolicyService = new BaseRedactStoragePolicyServiceImpl();
