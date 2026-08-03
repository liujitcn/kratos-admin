import service from "@liujitcn/kratos-admin-core/request";
import { readonly, shallowRef } from "vue";
import {
	type BaseLanguage,
  type BaseLanguageForm,
	type BaseLanguageService,
  type CreateBaseLanguageRequest,
  type DeleteBaseLanguageRequest,
  type GetBaseLanguageRequest,
  type OptionBaseLanguageRequest,
  type OptionBaseLanguageResponse,
  type PageBaseLanguageRequest,
  type PageBaseLanguageResponse,
  type SetBaseLanguagePrimaryRequest,
  type SetBaseLanguageStatusRequest,
  type UpdateBaseLanguageRequest
} from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_language";
import type { Empty } from "@liujitcn/kratos-admin-system/rpc/google/protobuf/empty";

const BASE_LANGUAGE_URL = "/v1/admin/base/language";

/** Admin语言服务。 */
export class BaseLanguageServiceImpl implements BaseLanguageService {
  /** 查询语言选项。 */
  OptionBaseLanguage(request: OptionBaseLanguageRequest = {}): Promise<OptionBaseLanguageResponse> {
    return service<OptionBaseLanguageRequest, OptionBaseLanguageResponse>({
      url: `${BASE_LANGUAGE_URL}/option`,
      method: "get",
      params: request
    });
  }

  /** 查询语言分页列表。 */
  PageBaseLanguage(request: PageBaseLanguageRequest): Promise<PageBaseLanguageResponse> {
    return service<PageBaseLanguageRequest, PageBaseLanguageResponse>({
      url: BASE_LANGUAGE_URL,
      method: "get",
      params: request
    });
  }

  /** 查询语言详情。 */
  GetBaseLanguage(request: GetBaseLanguageRequest): Promise<BaseLanguageForm> {
    return service<GetBaseLanguageRequest, BaseLanguageForm>({
      url: `${BASE_LANGUAGE_URL}/${request.id}`,
      method: "get"
    });
  }

  /** 创建语言。 */
  CreateBaseLanguage(request: CreateBaseLanguageRequest): Promise<Empty> {
    return service<BaseLanguageForm | undefined, Empty>({
      url: BASE_LANGUAGE_URL,
      method: "post",
      data: request.base_language
    });
  }

  /** 更新语言。 */
  UpdateBaseLanguage(request: UpdateBaseLanguageRequest): Promise<Empty> {
    return service<BaseLanguageForm | undefined, Empty>({
      url: `${BASE_LANGUAGE_URL}/${request.base_language?.id ?? ""}`,
      method: "put",
      data: request.base_language
    });
  }

  /** 删除语言。 */
  DeleteBaseLanguage(request: DeleteBaseLanguageRequest): Promise<Empty> {
    return service<DeleteBaseLanguageRequest, Empty>({
      url: `${BASE_LANGUAGE_URL}/${request.id}`,
      method: "delete"
    });
  }

  /** 设置语言启用状态。 */
  SetBaseLanguageStatus(request: SetBaseLanguageStatusRequest): Promise<Empty> {
    return service<SetBaseLanguageStatusRequest, Empty>({
      url: `${BASE_LANGUAGE_URL}/${request.id}/status`,
      method: "put",
      data: request
    });
  }

  /** 设置主语言。 */
  SetBaseLanguagePrimary(request: SetBaseLanguagePrimaryRequest): Promise<Empty> {
    return service<SetBaseLanguagePrimaryRequest, Empty>({
      url: `${BASE_LANGUAGE_URL}/${request.id}/primary`,
      method: "put",
      data: request
    });
  }
}

export const defBaseLanguageService = new BaseLanguageServiceImpl();

const enabledBaseLanguages = shallowRef<BaseLanguage[]>([]);
let enabledBaseLanguagesRequest: Promise<BaseLanguage[]> | undefined;

/** 加载启用语言选项并在当前管理端会话内缓存。 */
export async function loadEnabledBaseLanguages(force = false): Promise<BaseLanguage[]> {
  if (force) {
    enabledBaseLanguagesRequest = undefined;
  }
  if (enabledBaseLanguagesRequest) return enabledBaseLanguagesRequest;
  if (!force && enabledBaseLanguages.value.length) return enabledBaseLanguages.value;

  enabledBaseLanguagesRequest = defBaseLanguageService
    .OptionBaseLanguage({ enabled_only: true })
    .then(response => {
      const languages = response.base_languages ?? [];
      enabledBaseLanguages.value = languages;
      return languages;
    })
    .finally(() => {
      enabledBaseLanguagesRequest = undefined;
    });
  return enabledBaseLanguagesRequest;
}

/** 返回响应式的启用语言选项。 */
export function useEnabledBaseLanguages() {
  return { languages: readonly(enabledBaseLanguages) };
}

/** 返回当前缓存的启用语言选项快照。 */
export function getEnabledBaseLanguages(): BaseLanguage[] {
  return enabledBaseLanguages.value;
}

/** 清除启用语言缓存，使下次使用时重新读取语言管理数据。 */
export function invalidateEnabledBaseLanguages() {
  enabledBaseLanguages.value = [];
  enabledBaseLanguagesRequest = undefined;
}
