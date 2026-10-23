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

// BaseScriptService Admin脚本管理服务。
type BaseScriptService struct {
	adminv1.UnimplementedBaseScriptServiceServer
	baseScriptCase *biz.BaseScriptCase
}

// NewBaseScriptService 创建Admin脚本管理服务。
func NewBaseScriptService(baseScriptCase *biz.BaseScriptCase) *BaseScriptService {
	return &BaseScriptService{baseScriptCase: baseScriptCase}
}

// PageBaseScript 查询脚本分页列表。
func (s *BaseScriptService) PageBaseScript(ctx context.Context, req *adminv1.PageBaseScriptRequest) (*adminv1.PageBaseScriptResponse, error) {
	result, err := s.baseScriptCase.PageBaseScript(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("PageBaseScript %v", err))
		return nil, errorsx.WrapInternal(err, "查询脚本分页列表失败")
	}
	return result, nil
}

// GetBaseScript 查询脚本详情。
func (s *BaseScriptService) GetBaseScript(ctx context.Context, req *adminv1.GetBaseScriptRequest) (*adminv1.BaseScriptForm, error) {
	result, err := s.baseScriptCase.GetBaseScript(ctx, req.GetId())
	if err != nil {
		log.Error(fmt.Sprintf("GetBaseScript %v", err))
		return nil, errorsx.WrapInternal(err, "查询脚本失败")
	}
	return result, nil
}

// CreateBaseScript 创建脚本。
func (s *BaseScriptService) CreateBaseScript(ctx context.Context, req *adminv1.CreateBaseScriptRequest) (*emptypb.Empty, error) {
	err := s.baseScriptCase.CreateBaseScript(ctx, req.GetBaseScript())
	if err != nil {
		log.Error(fmt.Sprintf("CreateBaseScript %v", err))
		return nil, errorsx.WrapInternal(err, "创建脚本失败")
	}
	return new(emptypb.Empty), nil
}

// UpdateBaseScript 更新脚本。
func (s *BaseScriptService) UpdateBaseScript(ctx context.Context, req *adminv1.UpdateBaseScriptRequest) (*emptypb.Empty, error) {
	err := s.baseScriptCase.UpdateBaseScript(ctx, req.GetBaseScript())
	if err != nil {
		log.Error(fmt.Sprintf("UpdateBaseScript %v", err))
		return nil, errorsx.WrapInternal(err, "更新脚本失败")
	}
	return new(emptypb.Empty), nil
}

// DeleteBaseScript 删除脚本。
func (s *BaseScriptService) DeleteBaseScript(ctx context.Context, req *adminv1.DeleteBaseScriptRequest) (*emptypb.Empty, error) {
	err := s.baseScriptCase.DeleteBaseScript(ctx, req.GetId())
	if err != nil {
		log.Error(fmt.Sprintf("DeleteBaseScript %v", err))
		return nil, errorsx.WrapInternal(err, "删除脚本失败")
	}
	return new(emptypb.Empty), nil
}

// SetBaseScriptStatus 设置脚本状态。
func (s *BaseScriptService) SetBaseScriptStatus(ctx context.Context, req *adminv1.SetBaseScriptStatusRequest) (*emptypb.Empty, error) {
	err := s.baseScriptCase.SetBaseScriptStatus(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("SetBaseScriptStatus %v", err))
		return nil, errorsx.WrapInternal(err, "设置脚本状态失败")
	}
	return new(emptypb.Empty), nil
}

// TestRunBaseScript 试运行脚本。
func (s *BaseScriptService) TestRunBaseScript(ctx context.Context, req *adminv1.TestRunBaseScriptRequest) (*adminv1.TestRunBaseScriptResponse, error) {
	result, err := s.baseScriptCase.TestRunBaseScript(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("TestRunBaseScript %v", err))
		return nil, errorsx.WrapInternal(err, "试运行脚本失败")
	}
	return result, nil
}
