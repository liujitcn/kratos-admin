import type { FileInfo, MultiUploadFileResponse } from '@/rpc/base/v1/file'
import { formatSrc } from '@/utils/index.ts'

// 文件上传-兼容小程序端、H5端、App端
export const uploadFile = async (fileType: string, filePath: string): Promise<FileInfo> => {
  const res = await uni.uploadFile({
    url: '/v1/base/file',
    name: 'file',
    filePath: filePath,
    formData: {
      fileType: fileType,
    },
  })
  if (res.statusCode === 200) {
    return JSON.parse(res.data) as FileInfo
  } else {
    throw new Error('上传失败')
  }
}

// 文件上传-兼容小程序端、H5端、App端
export const uploadFileList = async (
  fileType: string,
  filePaths: string[],
): Promise<FileInfo[]> => {
  const fileInfoArr: FileInfo[] = []
  // 使用 Promise.all 等待所有上传完成
  await Promise.all(
    filePaths.map(async (filePath) => {
      try {
        const res = await uni.uploadFile({
          url: '/v1/base/file',
          name: 'file',
          filePath: filePath,
          formData: {
            fileType: fileType,
          },
        })
        if (res.statusCode === 200) {
          const data = JSON.parse(res.data) as FileInfo
          fileInfoArr.push(data)
        } else {
          console.error('上传失败，状态码:', res.statusCode)
        }
      } catch (error) {
        console.error('上传异常:', error)
      }
    }),
  )
  return fileInfoArr
}

// 多文件上传-兼容小程序端、H5端、App端
export const multiUploadFile = async (fileType: string, files: any): Promise<FileInfo[]> => {
  const res = await uni.uploadFile({
    url: '/v1/base/file/multi',
    name: 'file',
    filePath: '',
    files: files,
    formData: {
      fileType: fileType,
    },
  })
  if (res.statusCode === 200) {
    const data = JSON.parse(res.data) as MultiUploadFileResponse
    return data.files
  } else {
    await uni.showToast({ icon: 'error', title: '上传失败' })
    return []
  }
}

export const getFileInfo = (url: string): FileInfo => {
  // 处理路径分隔符（兼容Windows和Unix系统）
  const parts = url.split(/[\\/]/)
  // 获取文件名部分（包含扩展名）
  const fullName = parts.pop() || ''

  if (!fullName) return { name: '', extname: '', url: url } // 空文件名处理

  // 查找最后一个点号的位置
  const dotIndex = fullName.lastIndexOf('.')

  // 排除隐藏文件（如.gitignore）和无扩展名情况
  if (dotIndex > 0) {
    return {
      name: fullName,
      extname: fullName.slice(dotIndex),
      url: formatSrc(url),
    }
  }

  // 无合法扩展名时返回完整文件名
  return { name: fullName, extname: '', url: url }
}
