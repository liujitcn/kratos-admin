import service from "@liujitcn/kratos-admin-core/request";
import type { Empty } from "@liujitcn/kratos-admin-system/rpc/google/protobuf/empty";
import type {
  BaseI18nService,
  DraftBaseI18nRequest,
  DraftBaseI18nResponse,
  UpdateBaseI18nRequest
} from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_i18n";

const BASE_I18N_URL = "/v1/admin/base/i18n";

/** BaseI18nServiceImpl Admin 单条动态翻译服务。 */
export class BaseI18nServiceImpl implements BaseI18nService {
  /** DraftBaseI18n 翻译请求中的单个文本。 */
  DraftBaseI18n(request: DraftBaseI18nRequest): Promise<DraftBaseI18nResponse> {
    return service<DraftBaseI18nRequest, DraftBaseI18nResponse>({
      url: `${BASE_I18N_URL}/draft`,
      method: "post",
      data: request
    });
  }

  /** UpdateBaseI18n 修改或新增单条翻译信息。 */
  UpdateBaseI18n(request: UpdateBaseI18nRequest): Promise<Empty> {
    return service<UpdateBaseI18nRequest, Empty>({
      url: BASE_I18N_URL,
      method: "put",
      data: request
    });
  }
}

export const defBaseI18nService = new BaseI18nServiceImpl();
