/** 匹配逻辑路由，并提取冒号参数。 */
export function matchLogicalPath(pattern, path) {
  const patternParts = pattern.replace(/^\/+|\/+$/g, '').split('/')
  const pathParts = path.replace(/^\/+|\/+$/g, '').split('/')
  if (patternParts.length !== pathParts.length) return
  const params = {}
  for (let index = 0; index < patternParts.length; index += 1) {
    const patternPart = patternParts[index]
    const pathPart = pathParts[index]
    if (patternPart.startsWith(':')) {
      params[patternPart.slice(1)] = decodeURIComponent(pathPart)
    } else if (patternPart !== pathPart) {
      return
    }
  }
  return params
}

/** 解析逻辑路由 query。 */
export function parseLogicalQuery(query) {
  return query
    .split('&')
    .filter(Boolean)
    .reduce((result, item) => {
      const [key, value = ''] = item.split('=', 2)
      result[decodeURIComponent(key)] = decodeURIComponent(value)
      return result
    }, {})
}
