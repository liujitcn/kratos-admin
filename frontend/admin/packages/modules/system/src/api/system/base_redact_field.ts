import service from "@liujitcn/kratos-admin-core/request";
import type { BaseRedactFieldForm, BaseRedactFieldService, DeleteBaseRedactFieldRequest, GetBaseRedactFieldRequest, OptionBaseRedactFieldRequest, PageBaseRedactFieldRequest, PageBaseRedactFieldResponse, SetBaseRedactFieldStatusRequest, UpdateBaseRedactFieldRequest } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_redact_field";
import type { SelectOptionResponse } from "@liujitcn/kratos-admin-system/rpc/common/v1/common";
import type { Empty } from "@liujitcn/kratos-admin-system/rpc/google/protobuf/empty";

const BASE_REDACT_FIELD_URL = "/v1/admin/base/redact-field";

/** Admin脱敏字段目录服务。 */
export class BaseRedactFieldServiceImpl implements BaseRedactFieldService {
  /** 查询脱敏字段选项。 */
  OptionBaseRedactField(request: OptionBaseRedactFieldRequest = { keyword: "", operation: "" }): Promise<SelectOptionResponse> { return service<OptionBaseRedactFieldRequest, SelectOptionResponse>({ url: `${BASE_REDACT_FIELD_URL}/option`, method: "get", params: request }); }
  /** 查询脱敏字段分页列表。 */
  PageBaseRedactField(request: PageBaseRedactFieldRequest): Promise<PageBaseRedactFieldResponse> { return service<PageBaseRedactFieldRequest, PageBaseRedactFieldResponse>({ url: BASE_REDACT_FIELD_URL, method: "get", params: request }); }
  /** 查询脱敏字段详情。 */
  GetBaseRedactField(request: GetBaseRedactFieldRequest): Promise<BaseRedactFieldForm> { return service<GetBaseRedactFieldRequest, BaseRedactFieldForm>({ url: `${BASE_REDACT_FIELD_URL}/${request.id}`, method: "get" }); }
  /** 更新脱敏字段。 */
  UpdateBaseRedactField(request: UpdateBaseRedactFieldRequest): Promise<Empty> { return service<BaseRedactFieldForm | undefined, Empty>({ url: `${BASE_REDACT_FIELD_URL}/${request.base_redact_field?.id ?? ""}`, method: "put", data: request.base_redact_field }); }
  /** 删除脱敏字段。 */
  DeleteBaseRedactField(request: DeleteBaseRedactFieldRequest): Promise<Empty> { return service<DeleteBaseRedactFieldRequest, Empty>({ url: `${BASE_REDACT_FIELD_URL}/${request.id}`, method: "delete" }); }
  /** 设置脱敏字段状态。 */
  SetBaseRedactFieldStatus(request: SetBaseRedactFieldStatusRequest): Promise<Empty> { return service<SetBaseRedactFieldStatusRequest, Empty>({ url: `${BASE_REDACT_FIELD_URL}/${request.id}/status`, method: "put", data: request }); }
}

/** defBaseRedactFieldService 脱敏字段目录服务实例。 */
export const defBaseRedactFieldService = new BaseRedactFieldServiceImpl();
