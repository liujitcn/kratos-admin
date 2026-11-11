package base

import (
	"context"

	"github.com/go-kratos/kratos/v3/log"
	basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"
	biz "github.com/liujitcn/kratos-admin/backend/internal/biz/base"
	"github.com/liujitcn/kratos-core/errorsx"
	"google.golang.org/protobuf/types/known/emptypb"
)

// AiFileService 提供 AI 文件列表、回收站和元数据管理接口。
type AiFileService struct {
	basev1.UnimplementedAiFileServiceServer
	aiFileCase *biz.AiFileCase
}

// NewAiFileService 创建 AI 文件服务。
func NewAiFileService(aiFileCase *biz.AiFileCase) *AiFileService {
	return &AiFileService{aiFileCase: aiFileCase}
}

// PageAiFile 分页查询 AI 文件。
func (s *AiFileService) PageAiFile(ctx context.Context, req *basev1.PageAiFileRequest) (*basev1.PageAiFileResponse, error) {
	response, err := s.aiFileCase.PageAiFile(ctx, req)
	if err != nil {
		log.Error("PageAiFile", "error", err)
		return nil, errorsx.WrapInternal(err, "分页查询AI文件失败")
	}
	return response, nil
}

// RenameAiFile 重命名 AI 文件。
func (s *AiFileService) RenameAiFile(ctx context.Context, req *basev1.RenameAiFileRequest) (*basev1.AiFile, error) {
	response, err := s.aiFileCase.RenameAiFile(ctx, req)
	if err != nil {
		log.Error("RenameAiFile", "error", err)
		return nil, errorsx.WrapInternal(err, "重命名AI文件失败")
	}
	return response, nil
}

// TrashAiFile 将 AI 文件移入回收站。
func (s *AiFileService) TrashAiFile(ctx context.Context, req *basev1.TrashAiFileRequest) (*basev1.AiFile, error) {
	response, err := s.aiFileCase.TrashAiFile(ctx, req)
	if err != nil {
		log.Error("TrashAiFile", "error", err)
		return nil, errorsx.WrapInternal(err, "移入AI文件回收站失败")
	}
	return response, nil
}

// RestoreAiFile 从回收站恢复 AI 文件。
func (s *AiFileService) RestoreAiFile(ctx context.Context, req *basev1.RestoreAiFileRequest) (*basev1.AiFile, error) {
	response, err := s.aiFileCase.RestoreAiFile(ctx, req)
	if err != nil {
		log.Error("RestoreAiFile", "error", err)
		return nil, errorsx.WrapInternal(err, "恢复AI文件失败")
	}
	return response, nil
}

// DeleteAiFile 彻底删除 AI 文件。
func (s *AiFileService) DeleteAiFile(ctx context.Context, req *basev1.DeleteAiFileRequest) (*emptypb.Empty, error) {
	response, err := s.aiFileCase.DeleteAiFile(ctx, req)
	if err != nil {
		log.Error("DeleteAiFile", "error", err)
		return nil, errorsx.WrapInternal(err, "彻底删除AI文件失败")
	}
	return response, nil
}
