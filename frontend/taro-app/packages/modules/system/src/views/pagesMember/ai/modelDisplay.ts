import type { AiProviderModelOption } from '../../../rpc/base/v1/ai_provider'

/** 常见模型的展示名映射；未命中时按通用规则美化原始名。 */
const MODEL_DISPLAY_NAMES: Record<string, string> = {
  'deepseek-chat': 'DeepSeek Chat',
  'deepseek-reasoner': 'DeepSeek Reasoner',
  'deepseek-v3': 'DeepSeek V3',
  'deepseek-r1': 'DeepSeek R1',
  'gpt-4o': 'GPT-4o',
  'gpt-4o-mini': 'GPT-4o mini',
  'gpt-4.1': 'GPT-4.1',
  'gpt-4.1-mini': 'GPT-4.1 mini',
  'gpt-4-turbo': 'GPT-4 Turbo',
  'o3-mini': 'O3-mini',
  'claude-3-5-sonnet': 'Claude 3.5 Sonnet',
  'claude-3-7-sonnet': 'Claude 3.7 Sonnet',
  'qwen-max': 'Qwen Max',
  'qwen-plus': 'Qwen Plus',
}

/** 需要整体大写的模型名片段。 */
const UPPERCASE_TOKENS = new Set([
  'gpt',
  'llama',
  'qwen',
  'glm',
  'kimi',
  'gemini',
  'claude',
  'doubao',
  'deepseek',
  'moonshot',
  'hunyuan',
])

/** 对未配置展示名的模型名做通用美化。 */
export function fallbackModelDisplayName(modelName: string): string {
  if (!modelName) {
    return ''
  }
  const mapped = MODEL_DISPLAY_NAMES[modelName.toLowerCase()]
  if (mapped) {
    return mapped
  }
  return modelName
    .split(/[-_]/)
    .filter(Boolean)
    .map((token) => {
      const lower = token.toLowerCase()
      if (UPPERCASE_TOKENS.has(lower)) {
        return lower.toUpperCase()
      }
      return lower.charAt(0).toUpperCase() + lower.slice(1)
    })
    .join(' ')
}

/** 从供应商模型选项中解析模型展示名称，兼容历史消息中「供应商 / 模型」格式的原始值。 */
export function resolveModelDisplayName(
  providers: AiProviderModelOption[] | undefined,
  modelName: string,
): string {
  if (!modelName) {
    return ''
  }
  const separator = ' / '
  const modelPart = modelName.includes(separator)
    ? modelName.slice(modelName.lastIndexOf(separator) + separator.length)
    : modelName
  for (const provider of providers ?? []) {
    const matched = provider.models.find((item) => item.model_name === modelPart)
    if (matched?.display_name) {
      return matched.display_name
    }
  }
  return fallbackModelDisplayName(modelPart)
}
