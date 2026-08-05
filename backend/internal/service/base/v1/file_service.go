package base

import (
	"context"
	"fmt"

	basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"
	"github.com/liujitcn/kratos-admin/backend/core/pkg/errorsx"

	biz "github.com/liujitcn/kratos-admin/backend/internal/biz/base"

	"github.com/go-kratos/kratos/v3/log"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// FileService 文件服务
type FileService struct {
	basev1.UnimplementedFileServiceServer
	fileCase *biz.FileCase
}

// NewFileService 创建文件服务
func NewFileService(
	fileCase *biz.FileCase,
) *FileService {
	var ss = FileService{
		fileCase: fileCase}
	return &ss
}

// MultiUploadFile 多个文件上传
func (s *FileService) MultiUploadFile(ctx context.Context, req *basev1.MultiUploadFileRequest) (*basev1.MultiUploadFileResponse, error) {
	res, err := s.fileCase.MultiUploadFile(req)
	if err != nil {
		log.Error(fmt.Sprintf("MultiUploadFile %v", err))
		return nil, errorsx.WrapInternal(err, "多个文件上传失败")
	}
	return res, nil
}

// UploadFile 单个文件上传
func (s *FileService) UploadFile(ctx context.Context, req *basev1.UploadFileRequest) (*basev1.FileInfo, error) {
	res, err := s.fileCase.UploadFile(req)
	if err != nil {
		log.Error(fmt.Sprintf("UploadFile %v", err))
		return nil, errorsx.WrapInternal(err, "单个文件上传失败")
	}
	return res, nil
}

// DownloadFile 下载文件
func (s *FileService) DownloadFile(ctx context.Context, req *basev1.DownloadFileRequest) (*wrapperspb.BytesValue, error) {
	res, err := s.fileCase.DownloadFile(req)
	if err != nil {
		log.Error(fmt.Sprintf("DownloadFile %v", err))
		return nil, errorsx.WrapInternal(err, "下载文件失败")
	}
	return res, nil
}
