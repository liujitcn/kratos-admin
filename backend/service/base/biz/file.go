package biz

import (
	"fmt"
	"slices"
	"strings"

	basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"
	"github.com/liujitcn/kratos-admin/backend/pkg/errorsx"

	"github.com/go-kratos/kratos/v3/log"
	_string "github.com/liujitcn/go-utils/string"
	"github.com/liujitcn/kratos-kit/oss"
	"github.com/liujitcn/kratos-kit/sdk"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// FileCase 处理文件上传下载业务。
type FileCase struct {
	oss.OSS
}

// NewFileCase 创建文件业务实例。
func NewFileCase(
	oss oss.OSS,
) *FileCase {
	sdk.Runtime.SetOSS(oss)
	return &FileCase{
		OSS: oss,
	}
}

// DeleteFile 删除单个旧文件。
func (c *FileCase) DeleteFile(oldFile string, newFile string) {
	// 新旧文件不一致时，删除历史文件资源。
	if newFile == "" || oldFile != newFile {
		err := validateFilePath(oldFile)
		if err != nil {
			log.Error(fmt.Sprintf("DeleteFile %v", err))
			return
		}
		err = c.OSS.DeleteFile(oldFile)
		// 删除单个旧文件失败时，只记录日志不阻断调用方流程。
		if err != nil {
			log.Error(fmt.Sprintf("DeleteFile %v", err))
		}
	}
}

// MultiUploadFile 批量上传文件。
func (c *FileCase) MultiUploadFile(req *basev1.MultiUploadFileRequest) (*basev1.MultiUploadFileResponse, error) {
	files := make([]*basev1.FileInfo, 0)
	uploadFiles := req.GetFiles()
	for _, item := range uploadFiles {
		err := validateFilePath(item.GetPath())
		if err != nil {
			return nil, err
		}
		var url string
		url, err = c.UploadByByte(item.GetName(), item.GetPath(), item.GetContent())
		if err != nil {
			return nil, errorsx.Internal("文件上传失败").WithCause(err)
		}
		files = append(files, &basev1.FileInfo{
			Url:     url,
			Name:    item.GetName(),
			Extname: item.GetExtname(),
		})
	}
	return &basev1.MultiUploadFileResponse{Files: files}, nil
}

// UploadFile 上传单个文件。
func (c *FileCase) UploadFile(req *basev1.UploadFileRequest) (*basev1.FileInfo, error) {
	file := req.GetFile()
	err := validateFilePath(file.GetPath())
	if err != nil {
		return nil, err
	}
	var url string
	url, err = c.UploadByByte(file.GetName(), file.GetPath(), file.GetContent())
	if err != nil {
		return nil, errorsx.Internal("文件上传失败").WithCause(err)
	}
	return &basev1.FileInfo{
		Url:     url,
		Name:    file.GetName(),
		Extname: file.GetExtname(),
	}, nil
}

// DownloadFile 下载文件内容。
func (c *FileCase) DownloadFile(req *basev1.DownloadFileRequest) (*wrapperspb.BytesValue, error) {
	err := validateFilePath(req.GetPath())
	if err != nil {
		return nil, err
	}
	var fileByte []byte
	fileByte, err = c.GetFileByte(req.GetPath())
	if err != nil {
		return nil, errorsx.Internal("文件下载失败").WithCause(err)
	}
	return &wrapperspb.BytesValue{Value: fileByte}, nil
}

// MultiDeleteFileByString 按字符串数组配置删除历史文件。
func (c *FileCase) MultiDeleteFileByString(oldFile string, newFile []string) {
	c.MultiDeleteFile(_string.ConvertJsonStringToStringArray(oldFile), newFile)
}

// MultiDeleteFile 批量删除不再使用的旧文件。
func (c *FileCase) MultiDeleteFile(oldFile, newFile []string) {
	for _, item := range oldFile {
		// 新文件列表为空或未包含旧文件时，删除当前旧文件。
		if len(newFile) == 0 || !slices.Contains(newFile, item) {
			err := validateFilePath(item)
			if err != nil {
				log.Error(fmt.Sprintf("MultiDeleteFile %v", err))
				continue
			}
			err = c.OSS.DeleteFile(item)
			// 单个旧文件删除失败时，只记录日志继续处理剩余文件。
			if err != nil {
				log.Error(fmt.Sprintf("MultiDeleteFile %v", err))
			}
		}
	}
}

// validateFilePath 校验文件路径不能逃逸对象存储根目录。
func validateFilePath(filePath string) error {
	normalized := strings.ReplaceAll(filePath, `\`, "/")
	if normalized == "" || strings.ContainsRune(normalized, 0) {
		return errorsx.InvalidArgument("文件路径不合法")
	}
	for _, segment := range strings.Split(normalized, "/") {
		if segment == ".." {
			return errorsx.InvalidArgument("文件路径不合法")
		}
	}
	return nil
}
