import service from "@liujitcn/kratos-admin-core/request";
import type {
  BaseTranslationService,
  GenerateTranslationDraftRequest,
  GenerateTranslationDraftResponse
} from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_translation";

const BASE_TRANSLATION_URL = "/v1/admin/base/translation";

/** BaseTranslationServiceImpl Admin 动态翻译草稿服务。 */
export class BaseTranslationServiceImpl implements BaseTranslationService {
  /** GenerateTranslationDraft 为已保存资源生成机器翻译草稿。 */
  GenerateTranslationDraft(request: GenerateTranslationDraftRequest): Promise<GenerateTranslationDraftResponse> {
    return service<GenerateTranslationDraftRequest, GenerateTranslationDraftResponse>({
      url: `${BASE_TRANSLATION_URL}/draft`,
      method: "post",
      data: request
    });
  }
}

export const defBaseTranslationService = new BaseTranslationServiceImpl();
