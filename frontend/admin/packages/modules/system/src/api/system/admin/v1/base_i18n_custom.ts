import service from "@liujitcn/kratos-admin-core/request";
import type { Empty } from "@liujitcn/kratos-admin-system/rpc/google/protobuf/empty";
import type {
  BaseI18nCustomForm,
  BaseI18nCustomService,
  CreateBaseI18nCustomRequest,
  DeleteBaseI18nCustomRequest,
  GetBaseI18nCustomRequest,
  PageBaseI18nCustomRequest,
  PageBaseI18nCustomResponse,
  SetBaseI18nCustomStatusRequest,
  UpdateBaseI18nCustomRequest
} from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_i18n_custom";

const BASE_I18N_CUSTOM_URL = "/v1/admin/base/i18n-custom";

/** Admin前端国际化自定义翻译服务。 */
export class BaseI18nCustomServiceImpl implements BaseI18nCustomService {
  /** 查询国际化自定义翻译分页列表。 */
  PageBaseI18nCustom(request: PageBaseI18nCustomRequest): Promise<PageBaseI18nCustomResponse> {
    return service<PageBaseI18nCustomRequest, PageBaseI18nCustomResponse>({
      url: BASE_I18N_CUSTOM_URL,
      method: "get",
      params: request
    });
  }

  /** 查询国际化自定义翻译详情。 */
  GetBaseI18nCustom(request: GetBaseI18nCustomRequest): Promise<BaseI18nCustomForm> {
    return service<GetBaseI18nCustomRequest, BaseI18nCustomForm>({
      url: `${BASE_I18N_CUSTOM_URL}/${request.id}`,
      method: "get"
    });
  }

  /** 创建国际化自定义翻译。 */
  CreateBaseI18nCustom(request: CreateBaseI18nCustomRequest): Promise<Empty> {
    return service<BaseI18nCustomForm | undefined, Empty>({
      url: BASE_I18N_CUSTOM_URL,
      method: "post",
      data: request.i18n_custom
    });
  }

  /** 更新国际化自定义翻译。 */
  UpdateBaseI18nCustom(request: UpdateBaseI18nCustomRequest): Promise<Empty> {
    return service<BaseI18nCustomForm | undefined, Empty>({
      url: `${BASE_I18N_CUSTOM_URL}/${request.i18n_custom?.id ?? ""}`,
      method: "put",
      data: request.i18n_custom
    });
  }

  /** 删除国际化自定义翻译。 */
  DeleteBaseI18nCustom(request: DeleteBaseI18nCustomRequest): Promise<Empty> {
    return service<DeleteBaseI18nCustomRequest, Empty>({
      url: `${BASE_I18N_CUSTOM_URL}/${request.id}`,
      method: "delete"
    });
  }

  /** 设置国际化自定义翻译状态。 */
  SetBaseI18nCustomStatus(request: SetBaseI18nCustomStatusRequest): Promise<Empty> {
    return service<SetBaseI18nCustomStatusRequest, Empty>({
      url: `${BASE_I18N_CUSTOM_URL}/${request.id}/status`,
      method: "put",
      data: request
    });
  }
}

export const defBaseI18nCustomService = new BaseI18nCustomServiceImpl();
