import { t } from '@liujitcn/kratos-taro-app-core'

const PERSISTED_MESSAGE_PREFIX = '__I18N__:'
const LEGACY_PERSISTED_MESSAGE_PREFIX = '__I18N__'

/** 将持久化的国际化消息标记按当前语言渲染，兼容历史纯文本记录。 */
export function resolvePersistedMessage(value?: string): string {
  if (value === '系统') return t('system.notification.sender.system')
  if (!value) return ''
  const prefix = value.startsWith(PERSISTED_MESSAGE_PREFIX)
    ? PERSISTED_MESSAGE_PREFIX
    : value.startsWith(LEGACY_PERSISTED_MESSAGE_PREFIX)
      ? LEGACY_PERSISTED_MESSAGE_PREFIX
      : ''
  if (!prefix) return value
  const marker = value.split(/\r?\n/, 1)[0].slice(prefix.length)
  const separator = marker.indexOf('?')
  const key = separator < 0 ? marker : marker.slice(0, separator)
  if (!key) return value
  const params: Record<string, string> = {}
  if (separator >= 0) {
    new URLSearchParams(marker.slice(separator + 1)).forEach((paramValue, name) => {
      params[name] = paramValue
    })
  }
  const localized = t(key, params)
  return localized === key || localized === t('common.message.unknown') ? value : localized
}
