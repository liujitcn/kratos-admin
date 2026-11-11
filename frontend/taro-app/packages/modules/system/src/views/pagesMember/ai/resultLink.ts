/** 只接收结果定位编号，目标页仍须通过本人查询接口校验归属。 */
export function parseAiResultParams(value: unknown): Record<string, string> | undefined {
  if (!value || typeof value !== 'object') return undefined
  const input = value as Record<string, unknown>
  const params: Record<string, string> = {}
  for (const key of ['run_id', 'session_id']) {
    const id = input[key]
    if (typeof id === 'string' && /^[1-9][0-9]*$/.test(id)) params[key] = id
  }
  return Object.keys(params).length ? params : undefined
}
