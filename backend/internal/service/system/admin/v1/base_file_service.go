package admin

import (
	"context"
	"fmt"

	"github.com/go-kratos/kratos/v3/log"
	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	biz "github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin"
	"github.com/liujitcn/kratos-core/errorsx"
	"google.golang.org/protobuf/types/known/emptypb"
)

// BaseFileService 提供后台文件资产管理接口。
type BaseFileService struct {
	adminv1.UnimplementedBaseFileServiceServer
	baseFileCase *biz.BaseFileCase
}

// NewBaseFileService 创建后台文件资产管理服务。
func NewBaseFileService(baseFileCase *biz.BaseFileCase) *BaseFileService {
	return &BaseFileService{baseFileCase: baseFileCase}
}

// PageBaseFile 分页查询文件资产。
func (s *BaseFileService) PageBaseFile(ctx context.Context, req *adminv1.PageBaseFileRequest) (*adminv1.PageBaseFileResponse, error) {
	res, err := s.baseFileCase.PageBaseFile(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("PageBaseFile %v", err))
		return nil, errorsx.WrapInternal(err, "分页查询文件资产失败")
	}
	return res, nil
}

// GetBaseFile 查询文件资产详情。
func (s *BaseFileService) GetBaseFile(ctx context.Context, req *adminv1.GetBaseFileRequest) (*adminv1.BaseFile, error) {
	res, err := s.baseFileCase.GetBaseFile(ctx, req.GetId())
	if err != nil {
		log.Error(fmt.Sprintf("GetBaseFile %v", err))
		return nil, errorsx.WrapInternal(err, "查询文件资产详情失败")
	}
	return res, nil
}

// DeleteBaseFile 删除文件资产及其对象。
func (s *BaseFileService) DeleteBaseFile(ctx context.Context, req *adminv1.DeleteBaseFileRequest) (*emptypb.Empty, error) {
	if err := s.baseFileCase.DeleteBaseFile(ctx, req.GetId()); err != nil {
		log.Error(fmt.Sprintf("DeleteBaseFile %v", err))
		return nil, errorsx.WrapInternal(err, "删除文件资产失败")
	}
	return new(emptypb.Empty), nil
}
