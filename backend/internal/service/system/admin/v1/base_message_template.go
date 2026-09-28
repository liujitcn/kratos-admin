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

// BaseMessageTemplateService 消息 Provider 模板管理服务。
type BaseMessageTemplateService struct {
	adminv1.UnimplementedBaseMessageTemplateServiceServer
	baseMessageTemplateCase *biz.BaseMessageTemplateCase
}

// NewBaseMessageTemplateService 创建消息 Provider 模板管理服务。
func NewBaseMessageTemplateService(baseMessageTemplateCase *biz.BaseMessageTemplateCase) *BaseMessageTemplateService {
	return &BaseMessageTemplateService{baseMessageTemplateCase: baseMessageTemplateCase}
}

// PageBaseMessageTemplate 分页查询消息 Provider 模板。
func (s *BaseMessageTemplateService) PageBaseMessageTemplate(ctx context.Context, req *adminv1.PageBaseMessageTemplateRequest) (*adminv1.PageBaseMessageTemplateResponse, error) {
	result, err := s.baseMessageTemplateCase.PageBaseMessageTemplate(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("PageBaseMessageTemplate %v", err))
		return nil, errorsx.WrapInternal(err, "查询消息模板失败")
	}
	return result, nil
}

// GetBaseMessageTemplate 查询消息 Provider 模板详情。
func (s *BaseMessageTemplateService) GetBaseMessageTemplate(ctx context.Context, req *adminv1.GetBaseMessageTemplateRequest) (*adminv1.BaseMessageTemplateForm, error) {
	result, err := s.baseMessageTemplateCase.GetBaseMessageTemplate(ctx, req.GetId())
	if err != nil {
		log.Error(fmt.Sprintf("GetBaseMessageTemplate %v", err))
		return nil, errorsx.WrapInternal(err, "查询消息模板失败")
	}
	return result, nil
}

// CreateBaseMessageTemplate 创建消息 Provider 模板。
func (s *BaseMessageTemplateService) CreateBaseMessageTemplate(ctx context.Context, req *adminv1.CreateBaseMessageTemplateRequest) (*emptypb.Empty, error) {
	err := s.baseMessageTemplateCase.CreateBaseMessageTemplate(ctx, req.GetBaseMessageTemplate())
	if err != nil {
		log.Error(fmt.Sprintf("CreateBaseMessageTemplate %v", err))
		return nil, errorsx.WrapInternal(err, "创建消息模板失败")
	}
	return new(emptypb.Empty), nil
}

// UpdateBaseMessageTemplate 更新消息 Provider 模板。
func (s *BaseMessageTemplateService) UpdateBaseMessageTemplate(ctx context.Context, req *adminv1.UpdateBaseMessageTemplateRequest) (*emptypb.Empty, error) {
	err := s.baseMessageTemplateCase.UpdateBaseMessageTemplate(ctx, req.GetBaseMessageTemplate())
	if err != nil {
		log.Error(fmt.Sprintf("UpdateBaseMessageTemplate %v", err))
		return nil, errorsx.WrapInternal(err, "更新消息模板失败")
	}
	return new(emptypb.Empty), nil
}

// DeleteBaseMessageTemplate 删除消息 Provider 模板。
func (s *BaseMessageTemplateService) DeleteBaseMessageTemplate(ctx context.Context, req *adminv1.DeleteBaseMessageTemplateRequest) (*emptypb.Empty, error) {
	err := s.baseMessageTemplateCase.DeleteBaseMessageTemplate(ctx, req.GetId())
	if err != nil {
		log.Error(fmt.Sprintf("DeleteBaseMessageTemplate %v", err))
		return nil, errorsx.WrapInternal(err, "删除消息模板失败")
	}
	return new(emptypb.Empty), nil
}
