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

// BaseLanguageService Admin语言管理服务。
type BaseLanguageService struct {
	adminv1.UnimplementedBaseLanguageServiceServer
	baseLanguageCase *biz.BaseLanguageCase
}

// NewBaseLanguageService 创建Admin语言管理服务。
func NewBaseLanguageService(baseLanguageCase *biz.BaseLanguageCase) *BaseLanguageService {
	return &BaseLanguageService{baseLanguageCase: baseLanguageCase}
}

// OptionBaseLanguage 查询启用语言选项。
func (s *BaseLanguageService) OptionBaseLanguage(ctx context.Context, req *adminv1.OptionBaseLanguageRequest) (*adminv1.OptionBaseLanguageResponse, error) {
	res, err := s.baseLanguageCase.OptionBaseLanguage(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("OptionBaseLanguage %v", err))
		return nil, errorsx.WrapInternal(err, "查询语言选项失败")
	}
	return res, nil
}

// PageBaseLanguage 查询语言分页列表。
func (s *BaseLanguageService) PageBaseLanguage(ctx context.Context, req *adminv1.PageBaseLanguageRequest) (*adminv1.PageBaseLanguageResponse, error) {
	res, err := s.baseLanguageCase.PageBaseLanguage(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("PageBaseLanguage %v", err))
		return nil, errorsx.WrapInternal(err, "查询语言分页列表失败")
	}
	return res, nil
}

// GetBaseLanguage 查询语言详情。
func (s *BaseLanguageService) GetBaseLanguage(ctx context.Context, req *adminv1.GetBaseLanguageRequest) (*adminv1.BaseLanguageForm, error) {
	res, err := s.baseLanguageCase.GetBaseLanguage(ctx, req.GetId())
	if err != nil {
		log.Error(fmt.Sprintf("GetBaseLanguage %v", err))
		return nil, errorsx.WrapInternal(err, "查询语言失败")
	}
	return res, nil
}

// CreateBaseLanguage 创建语言。
func (s *BaseLanguageService) CreateBaseLanguage(ctx context.Context, req *adminv1.CreateBaseLanguageRequest) (*emptypb.Empty, error) {
	err := s.baseLanguageCase.CreateBaseLanguage(ctx, req.GetBaseLanguage())
	if err != nil {
		log.Error(fmt.Sprintf("CreateBaseLanguage %v", err))
		return nil, errorsx.WrapInternal(err, "创建语言失败")
	}
	return new(emptypb.Empty), nil
}

// UpdateBaseLanguage 更新语言。
func (s *BaseLanguageService) UpdateBaseLanguage(ctx context.Context, req *adminv1.UpdateBaseLanguageRequest) (*emptypb.Empty, error) {
	err := s.baseLanguageCase.UpdateBaseLanguage(ctx, req.GetBaseLanguage())
	if err != nil {
		log.Error(fmt.Sprintf("UpdateBaseLanguage %v", err))
		return nil, errorsx.WrapInternal(err, "更新语言失败")
	}
	return new(emptypb.Empty), nil
}

// DeleteBaseLanguage 删除语言。
func (s *BaseLanguageService) DeleteBaseLanguage(ctx context.Context, req *adminv1.DeleteBaseLanguageRequest) (*emptypb.Empty, error) {
	err := s.baseLanguageCase.DeleteBaseLanguage(ctx, req.GetId())
	if err != nil {
		log.Error(fmt.Sprintf("DeleteBaseLanguage %v", err))
		return nil, errorsx.WrapInternal(err, "删除语言失败")
	}
	return new(emptypb.Empty), nil
}

// SetBaseLanguageStatus 设置语言启用状态。
func (s *BaseLanguageService) SetBaseLanguageStatus(ctx context.Context, req *adminv1.SetBaseLanguageStatusRequest) (*emptypb.Empty, error) {
	err := s.baseLanguageCase.SetBaseLanguageStatus(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("SetBaseLanguageStatus %v", err))
		return nil, errorsx.WrapInternal(err, "设置语言状态失败")
	}
	return new(emptypb.Empty), nil
}

// SetBaseLanguagePrimary 设置主语言。
func (s *BaseLanguageService) SetBaseLanguagePrimary(ctx context.Context, req *adminv1.SetBaseLanguagePrimaryRequest) (*emptypb.Empty, error) {
	err := s.baseLanguageCase.SetBaseLanguagePrimary(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("SetBaseLanguagePrimary %v", err))
		return nil, errorsx.WrapInternal(err, "设置主语言失败")
	}
	return new(emptypb.Empty), nil
}
