import Taro from '@tarojs/taro'
import { getCurrentLocale, t } from '@liujitcn/kratos-taro-app-core'
import {
  getRequestAccessToken,
  requestBaseURL,
  sourceClient,
  http,
} from '@liujitcn/kratos-taro-app-core/utils/http'
import type {
  AiFile,
  AiFileService,
  PageAiFileRequest,
  PageAiFileResponse,
} from '../../../rpc/base/v1/ai_file'

/** AI 文件只读服务，不暴露移动端管理写操作。 */
class AiFileReadService implements Pick<AiFileService, 'PageAiFile'> {
  /** 按当前身份查询文件分类及分页。 */
  PageAiFile(request: PageAiFileRequest): Promise<PageAiFileResponse> {
    return http({ url: '/v1/base/ai/file', method: 'GET', authMode: 'required', data: request })
  }
}

/** 默认 AI 文件只读服务。 */
export const defAiFileService = new AiFileReadService()

/** 经统一鉴权文件接口预览或下载，图片与文档采用对应平台查看器。 */
export async function openAiWorkspaceFile(
  file: Pick<AiFile, 'file_name' | 'link_url' | 'mime_type' | 'extension'>,
  preview: boolean,
): Promise<string> {
  if (process.env.TARO_ENV === 'h5') {
    const bytes = await http<ArrayBuffer>({
      url: '/v1/base/file',
      method: 'GET',
      authMode: 'required',
      data: { name: file.file_name, path: file.link_url },
      responseType: 'arraybuffer',
    })
    const blob = new Blob([bytes], { type: file.mime_type || 'application/octet-stream' })
    if (preview && isAiWorkspaceTextFile(file)) return blob.text()
    const url = URL.createObjectURL(blob)
    if (preview && (file.mime_type || '').startsWith('image/')) {
      await Taro.previewImage({ current: url, urls: [url] })
    } else {
      const link = document.createElement('a')
      link.href = url
      link.download = file.file_name
      document.body.appendChild(link)
      link.click()
      link.remove()
    }
    setTimeout(() => URL.revokeObjectURL(url), 60000)
    return ''
  }

  const token = await getRequestAccessToken()
  const query = `name=${encodeURIComponent(file.file_name)}&path=${encodeURIComponent(file.link_url)}`
  const response = await Taro.downloadFile({
    url: `${requestBaseURL}/v1/base/file?${query}`,
    header: {
      Authorization: token,
      'source-client': sourceClient,
      'Accept-Language': getCurrentLocale(),
    },
  })
  if (response.statusCode !== 200) throw new Error(t('system.ai.workspace.file_unavailable'))
  if (preview && isAiWorkspaceTextFile(file) && typeof Taro.getFileSystemManager === 'function') {
    return String(Taro.getFileSystemManager().readFileSync(response.tempFilePath, 'utf8'))
  }
  if (!preview) {
    await Taro.saveFile({ tempFilePath: response.tempFilePath })
    await Taro.showToast({ icon: 'none', title: t('system.ai.workspace.file_saved') })
    return ''
  }
  if ((file.mime_type || '').startsWith('image/'))
    await Taro.previewImage({ current: response.tempFilePath, urls: [response.tempFilePath] })
  else await Taro.openDocument({ filePath: response.tempFilePath, showMenu: true })
  return ''
}

/** 文本以纯文本预览，HTML/SVG 不作为页面执行。 */
export function isAiWorkspaceTextFile(
  file: Pick<AiFile, 'extension' | 'file_name' | 'mime_type'>,
): boolean {
  const extension = (file.extension || file.file_name.split('.').pop() || '')
    .toLowerCase()
    .replace(/^\./, '')
  return (
    ['txt', 'md', 'csv', 'json', 'log', 'xml', 'yaml', 'yml', 'html', 'svg'].includes(extension) ||
    (file.mime_type || '').startsWith('text/')
  )
}
