import service from "@/utils/request";
import type { DownloadFileRequest, MultiUploadFileResponse, FileInfo } from "@/rpc/base/v1/file";

const FILE_URL = "/v1/base/file";

/** 文件服务 */
export class FileServiceImpl {
  /** 多个文件上传 */
  MultiUploadFile(files: File[], fileType: string): Promise<MultiUploadFileResponse> {
    const formData = new FormData();
    files.map(file => {
      formData.append(file.name, file);
    });
    formData.append("fileType", fileType);
    return service<FormData, MultiUploadFileResponse>({
      url: `${FILE_URL}/multi`,
      method: "post",
      data: formData,
      headers: {
        "Content-Type": "multipart/form-data"
      }
    });
  }
  /** 上传文件 */
  UploadFile(file: File, fileType: string): Promise<FileInfo> {
    const formData = new FormData();
    formData.append("file", file);
    formData.append("fileType", fileType);
    return service<FormData, FileInfo>({
      url: `${FILE_URL}`,
      method: "post",
      data: formData,
      headers: {
        "Content-Type": "multipart/form-data"
      }
    });
  }

  /** 读取文件内容并返回 Blob。 */
  async GetFileBlob(file: string, fileName: string): Promise<Blob> {
    const response = await service<DownloadFileRequest, any>({
      url: `${FILE_URL}`,
      method: "get",
      params: {
        name: fileName,
        path: file
      } as DownloadFileRequest,
      responseType: "blob"
    });
    return response.data instanceof Blob ? response.data : new Blob([response.data]);
  }

  /** 下载文件 */
  async DownloadFile(file: string, fileName: string): Promise<void> {
    try {
      // 创建下载链接
      const url = window.URL.createObjectURL(await this.GetFileBlob(file, fileName));
      const a = document.createElement("a");
      a.href = url;
      a.download = fileName;
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
      window.URL.revokeObjectURL(url);
    } catch (error) {
      console.error("下载错误:", error);
    }
  }
}

export const defFileService = new FileServiceImpl();
