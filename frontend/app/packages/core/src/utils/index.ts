const baseURL = import.meta.env.VITE_APP_STATIC_URL

/**
 * 解析静态资源访问前缀
 * @returns 静态资源访问前缀
 */
const resolveStaticOrigin = () => {
  // H5 开发环境优先走当前站点，由 Vite 代理转发静态资源。
  if (typeof window !== 'undefined' && window.location?.protocol?.startsWith('http')) {
    return window.location.origin
  }
  if (baseURL) {
    return baseURL.replace(/\/$/, '')
  }
  if (typeof window !== 'undefined' && window.location?.origin) {
    return window.location.origin
  }
  return ''
}
/**
 * 日期格式化函数
 * @param date 日期对象
 * @param format 日期格式，默认为 YYYY-MM-DD HH:mm:ss
 */
export const formatDate = (date: Date, format = 'YYYY-MM-DD HH:mm:ss') => {
  // 获取年月日时分秒，通过 padStart 补 0
  const year = String(date.getFullYear())
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  const hours = String(date.getHours()).padStart(2, '0')
  const minutes = String(date.getMinutes()).padStart(2, '0')
  const seconds = String(date.getSeconds()).padStart(2, '0')

  // 返回格式化后的结果
  return format
    .replace('YYYY', year)
    .replace('MM', month)
    .replace('DD', day)
    .replace('HH', hours)
    .replace('mm', minutes)
    .replace('ss', seconds)
}

/**
 * 金额格式化函数
 * @param price 金额
 */
export const formatPrice = (price: number) => {
  if (!price) {
    return '0.00'
  }
  return (price / 100).toFixed(2)
}

/**
 * 图片地址格式化函数
 * @param src 图片地址
 */
export const formatSrc = (src: string) => {
  if (!src) {
    return src
  }
  // 已经是绝对地址时直接返回，避免重复拼接。
  if (/^https?:\/\//.test(src)) {
    return src
  }

  const staticOrigin = resolveStaticOrigin()
  if (!staticOrigin) {
    return src
  }

  // 以 / 开头的后端静态资源路径需要固定挂到站点根路径，不能受 /app base 影响。
  if (src.startsWith('/')) {
    return `${staticOrigin}${src}`
  }

  return `${staticOrigin}/${src.replace(/^\/+/, '')}`
}
