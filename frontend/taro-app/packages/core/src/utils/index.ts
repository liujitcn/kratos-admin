import Taro from '@tarojs/taro'

const configuredStaticUrl =
  process.env.VITE_APP_STATIC_URL || process.env.VITE_APP_API_URL || ''

/** 将后端数据文件地址切换到当前构建配置的静态资源域名。 */
function rewriteDataFileOrigin(src: string, staticOrigin: string): string {
  if (!staticOrigin) return src
  const match = src.match(/^https?:\/\/[^/]+(\/data(?:\/|$).*)$/)
  return match ? `${staticOrigin}${match[1]}` : src
}

/** 日期格式化函数。 */
export function formatDate(date: Date, format = 'YYYY-MM-DD HH:mm:ss'): string {
  const year = String(date.getFullYear())
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  const hours = String(date.getHours()).padStart(2, '0')
  const minutes = String(date.getMinutes()).padStart(2, '0')
  const seconds = String(date.getSeconds()).padStart(2, '0')
  return format
    .replace('YYYY', year)
    .replace('MM', month)
    .replace('DD', day)
    .replace('HH', hours)
    .replace('mm', minutes)
    .replace('ss', seconds)
}

/** 将分单位金额格式化为元。 */
export function formatPrice(price: number): string {
  return price ? (price / 100).toFixed(2) : '0.00'
}

/** 格式化后端静态资源地址。 */
export function formatSrc(src: string): string {
  const browserOrigin =
    process.env.TARO_ENV === 'h5' && typeof window !== 'undefined' && window.location?.origin
      ? window.location.origin
      : ''
  const staticOrigin = browserOrigin || configuredStaticUrl.replace(/\/$/, '')
  if (!src) return src
  if (/^https?:\/\//.test(src)) return rewriteDataFileOrigin(src, staticOrigin)
  if (!staticOrigin) return src
  return src.startsWith('/')
    ? `${staticOrigin}${src}`
    : `${staticOrigin}/${src.replace(/^\/+/, '')}`
}

/**
 * 微信小程序原生胶囊按钮（右上角 ⋯ / ◯）右侧避让宽度（px）。
 * 自定义导航栏（navigationStyle: 'custom'）在 weapp 下仍会显示原生胶囊，
 * 页头右侧若放置可交互图标/按钮需预留该宽度，避免与胶囊重叠；
 * 非 weapp 平台返回 0，无需特殊处理。
 */
export function getMpCapsuleRightGap(): number {
  if (process.env.TARO_ENV !== 'weapp') return 0
  try {
    const menu = Taro.getMenuButtonBoundingClientRect()
    const { windowWidth } = Taro.getSystemInfoSync()
    return Math.max(0, Math.round(windowWidth - menu.left + 12))
  } catch {
    return 0
  }
}
