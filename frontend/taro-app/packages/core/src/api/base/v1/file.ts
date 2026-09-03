import Taro from '@tarojs/taro'
import type {
  DownloadFileRequest,
  FileInfo,
  MultiUploadFileResponse,
} from '../../../rpc/base/v1/file'
import { uploadFile, uploadFileList } from '../../../utils/file'
import { getLocaleRequestHeaders, t } from '../../../locales'
import { getRequestAccessToken, requestBaseURL } from '../../../utils/http'

const FILE_URL = '/v1/base/file'

/** 文件服务。 */
export class FileServiceImpl {
  /** 上传多个浏览器文件。 */
  async MultiUploadFile(files: File[], fileType: string): Promise<MultiUploadFileResponse> {
    if (process.env.TARO_ENV !== 'h5') {
      throw new Error(t('core.file.temporary_path_required'))
    }
    const paths = files.map((file) => URL.createObjectURL(file))
    try {
      return { files: await uploadFileList(fileType, paths) }
    } finally {
      paths.forEach((path) => URL.revokeObjectURL(path))
    }
  }

  /** 上传单个浏览器文件。 */
  async UploadFile(file: File, fileType: string): Promise<FileInfo> {
    if (process.env.TARO_ENV !== 'h5') {
      throw new Error(t('core.file.temporary_path_required'))
    }
    const path = URL.createObjectURL(file)
    try {
      return await uploadFile(fileType, path)
    } finally {
      URL.revokeObjectURL(path)
    }
  }

  /** 下载并打开后端文件。 */
  async DownloadFile(file: string, fileName: string): Promise<void> {
    const request: DownloadFileRequest = { name: fileName, path: file }
    const query = Object.entries(request)
      .map(([key, value]) => `${encodeURIComponent(key)}=${encodeURIComponent(String(value))}`)
      .join('&')
    const token = await getRequestAccessToken()
    const response = await Taro.downloadFile({
      url: `${requestBaseURL}${FILE_URL}?${query}`,
      header: {
        ...getLocaleRequestHeaders(),
        Authorization: token,
        'source-client': 'miniapp',
      },
    })
    if (response.statusCode !== 200) throw new Error(t('core.file.download_failed'))
    await Taro.openDocument({ filePath: response.tempFilePath, showMenu: true })
  }
}

/** 默认文件服务。 */
export const defFileService = new FileServiceImpl()
