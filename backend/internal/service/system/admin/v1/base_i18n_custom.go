package admin

import (
	"context"
	"fmt"

	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	biz "github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin"
	"github.com/liujitcn/kratos-core/errorsx"

	"github.com/go-kratos/kratos/v3/log"
	"google.golang.org/protobuf/types/known/emptypb"
)

// BaseI18nCustomService 管理前端国际化自定义翻译。
type BaseI18nCustomService struct {
	adminv1.UnimplementedBaseI18nCustomServiceServer
	baseI18nCustomCase *biz.BaseI18nCustomCase
}

// NewBaseI18nCustomService 创建前端国际化自定义翻译服务。
func NewBaseI18nCustomService(baseI18nCustomCase *biz.BaseI18nCustomCase) *BaseI18nCustomService {
	return &BaseI18nCustomService{baseI18nCustomCase: baseI18nCustomCase}
}

// PageBaseI18nCustom 查询国际化自定义翻译分页列表。
func (s *BaseI18nCustomService) PageBaseI18nCustom(ctx context.Context, req *adminv1.PageBaseI18nCustomRequest) (*adminv1.PageBaseI18nCustomResponse, error) {
	response, err := s.baseI18nCustomCase.PageBaseI18nCustom(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("PageBaseI18nCustom %v", err))
		return nil, errorsx.WrapInternal(err, "查询国际化自定义翻译分页列表失败")
	}
	return response, nil
}

// GetBaseI18nCustom 查询国际化自定义翻译详情。
func (s *BaseI18nCustomService) GetBaseI18nCustom(ctx context.Context, req *adminv1.GetBaseI18nCustomRequest) (*adminv1.BaseI18nCustomForm, error) {
	response, err := s.baseI18nCustomCase.GetBaseI18nCustom(ctx, req.GetId())
	if err != nil {
		log.Error(fmt.Sprintf("GetBaseI18nCustom %v", err))
		return nil, errorsx.WrapInternal(err, "查询国际化自定义翻译失败")
	}
	return response, nil
}

// CreateBaseI18nCustom 创建国际化自定义翻译。
func (s *BaseI18nCustomService) CreateBaseI18nCustom(ctx context.Context, req *adminv1.CreateBaseI18nCustomRequest) (*emptypb.Empty, error) {
	err := s.baseI18nCustomCase.CreateBaseI18nCustom(ctx, req.GetI18nCustom())
	if err != nil {
		log.Error(fmt.Sprintf("CreateBaseI18nCustom %v", err))
		return nil, errorsx.WrapInternal(err, "创建国际化自定义翻译失败")
	}
	return new(emptypb.Empty), nil
}

// UpdateBaseI18nCustom 更新国际化自定义翻译。
func (s *BaseI18nCustomService) UpdateBaseI18nCustom(ctx context.Context, req *adminv1.UpdateBaseI18nCustomRequest) (*emptypb.Empty, error) {
	err := s.baseI18nCustomCase.UpdateBaseI18nCustom(ctx, req.GetI18nCustom())
	if err != nil {
		log.Error(fmt.Sprintf("UpdateBaseI18nCustom %v", err))
		return nil, errorsx.WrapInternal(err, "更新国际化自定义翻译失败")
	}
	return new(emptypb.Empty), nil
}

// DeleteBaseI18nCustom 删除国际化自定义翻译。
func (s *BaseI18nCustomService) DeleteBaseI18nCustom(ctx context.Context, req *adminv1.DeleteBaseI18nCustomRequest) (*emptypb.Empty, error) {
	err := s.baseI18nCustomCase.DeleteBaseI18nCustom(ctx, req.GetId())
	if err != nil {
		log.Error(fmt.Sprintf("DeleteBaseI18nCustom %v", err))
		return nil, errorsx.WrapInternal(err, "删除国际化自定义翻译失败")
	}
	return new(emptypb.Empty), nil
}

// SetBaseI18nCustomStatus 设置国际化自定义翻译状态。
func (s *BaseI18nCustomService) SetBaseI18nCustomStatus(ctx context.Context, req *adminv1.SetBaseI18nCustomStatusRequest) (*emptypb.Empty, error) {
	err := s.baseI18nCustomCase.SetBaseI18nCustomStatus(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("SetBaseI18nCustomStatus %v", err))
		return nil, errorsx.WrapInternal(err, "设置国际化自定义翻译状态失败")
	}
	return new(emptypb.Empty), nil
}
