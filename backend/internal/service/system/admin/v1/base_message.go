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

// BaseMessageService 站内信管理服务。
type BaseMessageService struct {
	adminv1.UnimplementedBaseMessageServiceServer
	baseMessageCase *biz.BaseMessageCase
}

// NewBaseMessageService 创建站内信管理服务。
func NewBaseMessageService(baseMessageCase *biz.BaseMessageCase) *BaseMessageService {
	return &BaseMessageService{baseMessageCase: baseMessageCase}
}

// PageBaseMessage 分页查询消息。
func (s *BaseMessageService) PageBaseMessage(ctx context.Context, req *adminv1.PageBaseMessageRequest) (*adminv1.PageBaseMessageResponse, error) {
	result, err := s.baseMessageCase.PageBaseMessage(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("PageBaseMessage %v", err))
		return nil, errorsx.WrapInternal(err, "查询消息失败")
	}
	return result, nil
}

// GetBaseMessage 查询消息详情。
func (s *BaseMessageService) GetBaseMessage(ctx context.Context, req *adminv1.GetBaseMessageRequest) (*adminv1.BaseMessageDetail, error) {
	result, err := s.baseMessageCase.GetBaseMessage(ctx, req.GetId())
	if err != nil {
		log.Error(fmt.Sprintf("GetBaseMessage %v", err))
		return nil, errorsx.WrapInternal(err, "查询消息详情失败")
	}
	return result, nil
}

// CreateBaseMessage 创建消息草稿。
func (s *BaseMessageService) CreateBaseMessage(ctx context.Context, req *adminv1.CreateBaseMessageRequest) (*adminv1.CreateBaseMessageResponse, error) {
	id, err := s.baseMessageCase.CreateBaseMessage(ctx, req.GetBaseMessage())
	if err != nil {
		log.Error(fmt.Sprintf("CreateBaseMessage %v", err))
		return nil, errorsx.WrapInternal(err, "创建消息失败")
	}
	return &adminv1.CreateBaseMessageResponse{Id: id}, nil
}

// UpdateBaseMessage 更新消息草稿。
func (s *BaseMessageService) UpdateBaseMessage(ctx context.Context, req *adminv1.UpdateBaseMessageRequest) (*emptypb.Empty, error) {
	err := s.baseMessageCase.UpdateBaseMessage(ctx, req.GetBaseMessage())
	if err != nil {
		log.Error(fmt.Sprintf("UpdateBaseMessage %v", err))
		return nil, errorsx.WrapInternal(err, "更新消息失败")
	}
	return new(emptypb.Empty), nil
}

// DeleteBaseMessage 删除消息草稿。
func (s *BaseMessageService) DeleteBaseMessage(ctx context.Context, req *adminv1.DeleteBaseMessageRequest) (*emptypb.Empty, error) {
	err := s.baseMessageCase.DeleteBaseMessage(ctx, req.GetId())
	if err != nil {
		log.Error(fmt.Sprintf("DeleteBaseMessage %v", err))
		return nil, errorsx.WrapInternal(err, "删除消息失败")
	}
	return new(emptypb.Empty), nil
}

// PublishBaseMessage 发布消息。
func (s *BaseMessageService) PublishBaseMessage(ctx context.Context, req *adminv1.PublishBaseMessageRequest) (*emptypb.Empty, error) {
	err := s.baseMessageCase.PublishBaseMessage(ctx, req.GetId())
	if err != nil {
		log.Error(fmt.Sprintf("PublishBaseMessage %v", err))
		return nil, errorsx.WrapInternal(err, "发布消息失败")
	}
	return new(emptypb.Empty), nil
}

// CancelBaseMessageSchedule 取消消息定时发布。
func (s *BaseMessageService) CancelBaseMessageSchedule(ctx context.Context, req *adminv1.CancelBaseMessageScheduleRequest) (*emptypb.Empty, error) {
	err := s.baseMessageCase.CancelBaseMessageSchedule(ctx, req.GetId())
	if err != nil {
		log.Error(fmt.Sprintf("CancelBaseMessageSchedule %v", err))
		return nil, errorsx.WrapInternal(err, "取消消息定时发布失败")
	}
	return new(emptypb.Empty), nil
}

// RevokeBaseMessage 撤回消息。
func (s *BaseMessageService) RevokeBaseMessage(ctx context.Context, req *adminv1.RevokeBaseMessageRequest) (*emptypb.Empty, error) {
	err := s.baseMessageCase.RevokeBaseMessage(ctx, req.GetId())
	if err != nil {
		log.Error(fmt.Sprintf("RevokeBaseMessage %v", err))
		return nil, errorsx.WrapInternal(err, "撤回消息失败")
	}
	return new(emptypb.Empty), nil
}

// RetryBaseMessageDispatch 重试消息投递任务。
func (s *BaseMessageService) RetryBaseMessageDispatch(ctx context.Context, req *adminv1.RetryBaseMessageDispatchRequest) (*emptypb.Empty, error) {
	err := s.baseMessageCase.RetryBaseMessageDispatch(ctx, req.GetId())
	if err != nil {
		log.Error(fmt.Sprintf("RetryBaseMessageDispatch %v", err))
		return nil, errorsx.WrapInternal(err, "重试消息投递失败")
	}
	return new(emptypb.Empty), nil
}
