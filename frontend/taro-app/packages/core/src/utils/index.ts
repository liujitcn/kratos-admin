const configuredStaticUrl = process.env.VITE_APP_STATIC_URL || ''

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
  if (!src || /^https?:\/\//.test(src)) return src
  const browserOrigin =
    typeof window !== 'undefined' && window.location?.origin ? window.location.origin : ''
  const staticOrigin = browserOrigin || configuredStaticUrl.replace(/\/$/, '')
  if (!staticOrigin) return src
  return src.startsWith('/')
    ? `${staticOrigin}${src}`
    : `${staticOrigin}/${src.replace(/^\/+/, '')}`
}
