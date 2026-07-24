import { http } from '@/utils/http'
import type {
  AiToolService,
  ListAiShortcutRequest,
  ListAiShortcutResponse,
} from '@/rpc/base/v1/ai_tool'

const AI_SHORTCUT_URL = '/v1/base/ai/shortcut'

/** AI 助手工具服务。 */
export class AiToolServiceImpl implements AiToolService {
  /** 查询 AI 助手快捷入口列表。 */
  ListAiShortcut(request: ListAiShortcutRequest): Promise<ListAiShortcutResponse> {
    return http<ListAiShortcutResponse>({
      url: AI_SHORTCUT_URL,
      method: 'GET',
      authMode: 'required',
      data: request,
    })
  }
}

export const defAiToolService = new AiToolServiceImpl()
