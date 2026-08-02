import Taro from '@tarojs/taro'
import type { FileInfo, MultiUploadFileResponse } from '../rpc/base/v1/file'
import { getLocaleRequestHeaders, t } from '../locales'
import { formatSrc } from './index'
import { getRequestAccessToken, requestBaseURL } from './http'

/** 上传单个文件。 */
export async function uploadFile(fileType: string, filePath: string): Promise<FileInfo> {
  const token = await getRequestAccessToken()
  const response = await Taro.uploadFile({
    url: `${requestBaseURL}/v1/base/file`,
    name: 'file',
    filePath,
    formData: { fileType },
    header: {
      ...getLocaleRequestHeaders(),
      'source-client': 'miniapp',
      ...(token ? { Authorization: token } : {}),
    },
  })
  if (response.statusCode !== 200) throw new Error(t('core.file.uploadFailed'))
  return JSON.parse(response.data) as FileInfo
}

/** 并发上传文件列表。 */
export async function uploadFileList(fileType: string, filePaths: string[]): Promise<FileInfo[]> {
  const results = await Promise.allSettled(filePaths.map((filePath) => uploadFile(fileType, filePath)))
  return results.flatMap((result) => (result.status === 'fulfilled' ? [result.value] : []))
}

/** 使用微信小程序 files 参数上传多个文件。 */
export async function multiUploadFile(
  fileType: string,
  files: Array<{ path: string; name?: string }> = [],
): Promise<FileInfo[]> {
  const uploaded = await uploadFileList(
    fileType,
    files.map((file) => file.path),
  )
  const response: MultiUploadFileResponse = { files: uploaded }
  return response.files
}

/** 从 URL 构建文件元信息。 */
export function getFileInfo(url: string): FileInfo {
  const fullName = url.split(/[\\/]/).pop() || ''
  if (!fullName) return { name: '', extname: '', url }
  const dotIndex = fullName.lastIndexOf('.')
  if (dotIndex > 0) {
    return { name: fullName, extname: fullName.slice(dotIndex), url: formatSrc(url) }
  }
  return { name: fullName, extname: '', url }
}
