import service from "@liujitcn/kratos-admin-core/request";
import type { Empty } from "@liujitcn/kratos-admin-system/rpc/google/protobuf/empty";
import type {
  BaseI18nService,
  DraftBaseTranslationRequest,
  DraftBaseTranslationResponse,
  UpdateBaseTranslationRequest
} from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_i18n";

const BASE_TRANSLATION_URL = "/v1/admin/base/translation";

/** BaseI18nServiceImpl Admin 单条动态翻译服务。 */
export class BaseI18nServiceImpl implements BaseI18nService {
  /** DraftBaseTranslation 翻译请求中的单个文本。 */
  DraftBaseTranslation(request: DraftBaseTranslationRequest): Promise<DraftBaseTranslationResponse> {
    return service<DraftBaseTranslationRequest, DraftBaseTranslationResponse>({
      url: `${BASE_TRANSLATION_URL}/draft`,
      method: "post",
      data: request
    });
  }

  /** UpdateBaseTranslation 修改或新增单条翻译信息。 */
  UpdateBaseTranslation(request: UpdateBaseTranslationRequest): Promise<Empty> {
    return service<UpdateBaseTranslationRequest, Empty>({
      url: BASE_TRANSLATION_URL,
      method: "put",
      data: request
    });
  }
}

export const defBaseI18nService = new BaseI18nServiceImpl();
