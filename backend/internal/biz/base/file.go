package biz

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"path"
	"path/filepath"
	"strings"
	"time"

	basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	"github.com/liujitcn/kratos-core/biz"
	"github.com/liujitcn/kratos-core/errorsx"

	"github.com/go-kratos/kratos/v3/log"
	"github.com/liujitcn/go-utils/id"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

const maxFileContentBytes = 20 << 20

var allowedFileExtensions = map[string]struct{}{
	".bmp": {}, ".csv": {}, ".doc": {}, ".docx": {}, ".gif": {}, ".jpeg": {}, ".jpg": {},
	".mp3": {}, ".mp4": {}, ".pdf": {}, ".png": {}, ".ppt": {}, ".pptx": {}, ".txt": {},
	".wav": {}, ".webp": {}, ".xls": {}, ".xlsx": {},
}

// FileCase 处理文件上传下载业务。
type FileCase struct {
	*biz.BaseCase
	baseFileRepo *data.BaseFileRepository
}

// NewFileCase 创建文件业务实例。
func NewFileCase(baseCase *biz.BaseCase, baseFileRepo *data.BaseFileRepository) *FileCase {
	return &FileCase{BaseCase: baseCase, baseFileRepo: baseFileRepo}
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
func (c *FileCase) MultiUploadFile(ctx context.Context, req *basev1.MultiUploadFileRequest) (*basev1.MultiUploadFileResponse, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}
	files := make([]*basev1.FileInfo, 0)
	uploadFiles := req.GetFiles()
	if len(uploadFiles) > 20 {
		return nil, errorsx.InvalidArgument("单次上传文件数量不能超过 20 个")
	}
	for _, item := range uploadFiles {
		var objectPath string
		objectPath, err = tenantFilePath(authInfo.TenantId, item.GetPath())
		if err != nil {
			return nil, err
		}
		if err = validateFileContent(item.GetName(), item.GetContent()); err != nil {
			return nil, err
		}
		var url string
		url, err = c.OSS.UploadByByte(item.GetName(), objectPath, item.GetContent())
		if err != nil {
			return nil, errorsx.Internal("文件上传失败").WithCause(err)
		}
		if err = c.recordUploadedFile(ctx, authInfo.TenantId, authInfo.UserId, item.GetName(), item.GetExtname(), item.GetContent(), url); err != nil {
			_ = c.OSS.DeleteFile(url)
			return nil, err
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
func (c *FileCase) UploadFile(ctx context.Context, req *basev1.UploadFileRequest) (*basev1.FileInfo, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}
	file := req.GetFile()
	var objectPath string
	objectPath, err = tenantFilePath(authInfo.TenantId, file.GetPath())
	if err != nil {
		return nil, err
	}
	if err = validateFileContent(file.GetName(), file.GetContent()); err != nil {
		return nil, err
	}
	var url string
	url, err = c.OSS.UploadByByte(file.GetName(), objectPath, file.GetContent())
	if err != nil {
		return nil, errorsx.Internal("文件上传失败").WithCause(err)
	}
	if err = c.recordUploadedFile(ctx, authInfo.TenantId, authInfo.UserId, file.GetName(), file.GetExtname(), file.GetContent(), url); err != nil {
		_ = c.OSS.DeleteFile(url)
		return nil, err
	}
	return &basev1.FileInfo{
		Url:     url,
		Name:    file.GetName(),
		Extname: file.GetExtname(),
	}, nil
}

// recordUploadedFile 保存上传成功后的文件元数据。
func (c *FileCase) recordUploadedFile(ctx context.Context, tenantID, userID int64, fileName, extension string, content []byte, objectPath string) error {
	if c.baseFileRepo == nil {
		return errorsx.Internal("文件元数据仓储未配置")
	}
	cleanPath := strings.TrimPrefix(objectPath, "/")
	directory := path.Dir(cleanPath)
	if directory == "." {
		directory = ""
	}
	hash := sha256.Sum256(content)
	now := time.Now()
	entity := &models.BaseFile{
		TenantID:      tenantID,
		Provider:      0,
		FileDirectory: directory,
		FileGUID:      id.NewGUIDv7NoHyphen(),
		SaveFileName:  path.Base(cleanPath),
		FileName:      fileName,
		Extension:     extension,
		MimeType:      http.DetectContentType(content),
		Size:          int64(len(content)),
		LinkURL:       objectPath,
		ContentHash:   hex.EncodeToString(hash[:]),
		CreatedBy:     userID,
		UpdatedBy:     userID,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	var err error
	err = c.baseFileRepo.Create(ctx, entity)
	if err != nil {
		return errorsx.Internal("保存文件元数据失败").WithCause(err)
	}
	return nil
}

// DownloadFile 下载文件内容。
func (c *FileCase) DownloadFile(ctx context.Context, req *basev1.DownloadFileRequest) (*wrapperspb.BytesValue, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}
	var objectPath string
	objectPath, err = tenantFilePath(authInfo.TenantId, req.GetPath())
	if err != nil {
		return nil, err
	}
	var fileByte []byte
	fileByte, err = c.OSS.GetFileByte(objectPath)
	if err != nil {
		return nil, errorsx.Internal("文件下载失败").WithCause(err)
	}
	return &wrapperspb.BytesValue{Value: fileByte}, nil
}

// validateFilePath 校验文件路径不能逃逸对象存储根目录。
func validateFilePath(filePath string) error {
	normalized := strings.ReplaceAll(filePath, `\`, "/")
	if normalized == "" || len(normalized) > 256 || strings.ContainsRune(normalized, 0) {
		return errorsx.InvalidArgument("文件路径不合法")
	}
	for _, segment := range strings.Split(normalized, "/") {
		if segment == ".." {
			return errorsx.InvalidArgument("文件路径不合法")
		}
	}
	return nil
}

// tenantFilePath 将对象路径绑定到当前租户目录，阻止跨租户读取或写入。
func tenantFilePath(tenantID int64, filePath string) (string, error) {
	if tenantID <= 0 {
		return "", errorsx.PermissionDenied("无法识别文件所属租户")
	}
	err := validateFilePath(filePath)
	if err != nil {
		return "", err
	}
	normalized := "/" + strings.TrimPrefix(strings.ReplaceAll(filePath, `\`, "/"), "/")
	prefix := fmt.Sprintf("/tenant/%d/", tenantID)
	if strings.HasPrefix(normalized, "/tenant/") {
		if !strings.HasPrefix(normalized, prefix) {
			return "", errorsx.PermissionDenied("文件不属于当前租户")
		}
		return normalized, nil
	}
	return prefix + strings.TrimPrefix(normalized, "/"), nil
}

// validateFileContent 校验上传内容大小并拒绝可直接执行或嵌入的危险类型。
func validateFileContent(fileName string, content []byte) error {
	if len(content) == 0 || len(content) > maxFileContentBytes {
		return errorsx.InvalidArgument("文件内容不能为空且不能超过 20 MB")
	}
	if len(fileName) > 255 || strings.ContainsAny(fileName, "\r\n") {
		return errorsx.InvalidArgument("文件名不合法")
	}
	extension := strings.ToLower(filepath.Ext(fileName))
	if _, ok := allowedFileExtensions[extension]; !ok {
		return errorsx.InvalidArgument("文件扩展名不在允许范围内")
	}
	detectedType := http.DetectContentType(content)
	switch detectedType {
	case "text/html; charset=utf-8", "image/svg+xml", "application/x-httpd-php", "application/x-shockwave-flash":
		return errorsx.InvalidArgument("不允许上传可执行或可嵌入脚本的文件")
	default:
		return nil
	}
}
