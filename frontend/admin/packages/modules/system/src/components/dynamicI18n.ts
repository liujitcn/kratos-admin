import { getEnabledBaseLanguages } from "@liujitcn/kratos-admin-system/api/system/admin/v1/base_language";
import type { I18nTargetType } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_i18n";

/** DynamicI18nValue 动态资源单语言翻译编辑状态。 */
export interface DynamicI18nValue {
  /** 语言代码。 */
  locale: string;
  text: string;
  id: number;
}

/** DynamicI18nRecord 描述三类动态资源翻译共有字段。 */
export interface DynamicI18nRecord extends Omit<DynamicI18nValue, "text"> {
	target_type: I18nTargetType;
	target_id: number;
	name: string;
}

/** DynamicLanguageOption 动态翻译编辑器显示的语言选项。 */
export interface DynamicLanguageOption {
  /** 语言代码。 */
  value: string;
  /** 语言显示名称。 */
  label: string;
}

/** 返回语言管理中启用且非主语言的翻译语言选项。 */
export function getEditableLanguageOptions(): DynamicLanguageOption[] {
  return getEnabledBaseLanguages()
    .filter(item => !item.is_primary)
    .map(item => ({ value: item.language_code, label: item.native_name || item.language_name || item.language_code }));
}

/** 返回语言管理中的语言显示名称。 */
export function getLanguageLabel(locale: string): string {
  const language = getEnabledBaseLanguages().find(item => item.language_code === locale);
  return language?.native_name || language?.language_name || locale;
}

/** normalizeDynamicI18ns 将后端翻译记录补齐为全部非主语言编辑状态。 */
export function normalizeDynamicI18ns(
  records: DynamicI18nRecord[] | undefined,
  locales: string[] = getEditableLanguageOptions().map(item => item.value)
): DynamicI18nValue[] {
  return locales.map(locale => {
    const record = records?.find(item => item.locale === locale);
    return {
      id: record?.id ?? 0,
      locale,
      text: String(record?.name ?? "")
    };
  });
}

/** serializeDynamicI18ns 将编辑状态转换回后端资源翻译结构。 */
export function serializeDynamicI18ns(
  values: DynamicI18nValue[],
  targetType: I18nTargetType,
  targetId: number
): Array<{ id: number; target_type: I18nTargetType; target_id: number; locale: string; name: string }> {
  return values.map(({ id, locale, text }) => ({ id, target_type: targetType, target_id: targetId, locale, name: text }));
}
