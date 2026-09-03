import service from "@/utils/request";
import type { LanguageService, OptionLanguageRequest, OptionLanguageResponse } from "@/rpc/base/v1/language";

const LANGUAGE_URL = "/v1/base/language";

/** 语言公共服务。 */
export class LanguageServiceImpl implements LanguageService {
  /** 查询当前支持的语言选项。 */
  OptionLanguage(request: OptionLanguageRequest): Promise<OptionLanguageResponse> {
    return service<OptionLanguageRequest, OptionLanguageResponse>({
      url: LANGUAGE_URL,
      method: "get",
      params: request,
      headers: { Authorization: "no-auth" }
    });
  }
}

/** 主语言公共服务。 */
export const defLanguageService = new LanguageServiceImpl();
