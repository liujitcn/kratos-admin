package base

import (
	"context"
	"fmt"

	basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"
	biz "github.com/liujitcn/kratos-admin/backend/internal/biz/base"
	"github.com/liujitcn/kratos-core/errorsx"

	"github.com/go-kratos/kratos/v3/log"
	"google.golang.org/protobuf/types/known/emptypb"
)

// NotificationService 当前用户站内信服务。
type NotificationService struct {
	basev1.UnimplementedNotificationServiceServer
	notificationCase *biz.NotificationCase
}

// NewNotificationService 创建当前用户站内信服务。
func NewNotificationService(notificationCase *biz.NotificationCase) *NotificationService {
	return &NotificationService{notificationCase: notificationCase}
}

// PageNotification 分页查询当前用户收件箱。
func (s *NotificationService) PageNotification(ctx context.Context, req *basev1.PageNotificationRequest) (*basev1.PageNotificationResponse, error) {
	result, err := s.notificationCase.PageNotification(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("PageNotification %v", err))
		return nil, errorsx.WrapInternal(err, "查询站内信失败")
	}
	return result, nil
}

// ListNotificationCategories 查询消息分类列表。
func (s *NotificationService) ListNotificationCategories(ctx context.Context, req *basev1.ListNotificationCategoriesRequest) (*basev1.ListNotificationCategoriesResponse, error) {
	result, err := s.notificationCase.ListNotificationCategories(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("ListNotificationCategories %v", err))
		return nil, errorsx.WrapInternal(err, "查询消息分类失败")
	}
	return result, nil
}

// GetNotification 查询当前用户消息详情。
func (s *NotificationService) GetNotification(ctx context.Context, req *basev1.GetNotificationRequest) (*basev1.Notification, error) {
	result, err := s.notificationCase.GetNotification(ctx, req.GetId())
	if err != nil {
		log.Error(fmt.Sprintf("GetNotification %v", err))
		return nil, errorsx.WrapInternal(err, "查询站内信详情失败")
	}
	return result, nil
}

// GetNotificationSummary 查询当前用户未读汇总。
func (s *NotificationService) GetNotificationSummary(ctx context.Context, _ *basev1.GetNotificationSummaryRequest) (*basev1.NotificationSummary, error) {
	result, err := s.notificationCase.GetNotificationSummary(ctx)
	if err != nil {
		log.Error(fmt.Sprintf("GetNotificationSummary %v", err))
		return nil, errorsx.WrapInternal(err, "查询站内信未读汇总失败")
	}
	return result, nil
}

// MarkNotificationRead 标记消息为已读。
func (s *NotificationService) MarkNotificationRead(ctx context.Context, req *basev1.MarkNotificationReadRequest) (*emptypb.Empty, error) {
	err := s.notificationCase.MarkNotificationRead(ctx, req.GetIds())
	if err != nil {
		log.Error(fmt.Sprintf("MarkNotificationRead %v", err))
		return nil, errorsx.WrapInternal(err, "标记站内信已读失败")
	}
	return new(emptypb.Empty), nil
}

// MarkNotificationUnread 标记消息为未读。
func (s *NotificationService) MarkNotificationUnread(ctx context.Context, req *basev1.MarkNotificationUnreadRequest) (*emptypb.Empty, error) {
	err := s.notificationCase.MarkNotificationUnread(ctx, req.GetIds())
	if err != nil {
		log.Error(fmt.Sprintf("MarkNotificationUnread %v", err))
		return nil, errorsx.WrapInternal(err, "标记站内信未读失败")
	}
	return new(emptypb.Empty), nil
}

// MarkAllNotificationRead 标记水位线之前的全部消息为已读。
func (s *NotificationService) MarkAllNotificationRead(ctx context.Context, req *basev1.MarkAllNotificationReadRequest) (*emptypb.Empty, error) {
	err := s.notificationCase.MarkAllNotificationRead(ctx, req.GetBeforeDeliveryId())
	if err != nil {
		log.Error(fmt.Sprintf("MarkAllNotificationRead %v", err))
		return nil, errorsx.WrapInternal(err, "全部标记已读失败")
	}
	return new(emptypb.Empty), nil
}

// ArchiveNotification 归档消息。
func (s *NotificationService) ArchiveNotification(ctx context.Context, req *basev1.ArchiveNotificationRequest) (*emptypb.Empty, error) {
	err := s.notificationCase.ArchiveNotification(ctx, req.GetId())
	if err != nil {
		log.Error(fmt.Sprintf("ArchiveNotification %v", err))
		return nil, errorsx.WrapInternal(err, "归档站内信失败")
	}
	return new(emptypb.Empty), nil
}

// RestoreNotification 恢复已归档消息。
func (s *NotificationService) RestoreNotification(ctx context.Context, req *basev1.RestoreNotificationRequest) (*emptypb.Empty, error) {
	err := s.notificationCase.RestoreNotification(ctx, req.GetId())
	if err != nil {
		log.Error(fmt.Sprintf("RestoreNotification %v", err))
		return nil, errorsx.WrapInternal(err, "恢复站内信失败")
	}
	return new(emptypb.Empty), nil
}

// DeleteNotification 从个人收件箱删除消息。
func (s *NotificationService) DeleteNotification(ctx context.Context, req *basev1.DeleteNotificationRequest) (*emptypb.Empty, error) {
	err := s.notificationCase.DeleteNotification(ctx, req.GetId())
	if err != nil {
		log.Error(fmt.Sprintf("DeleteNotification %v", err))
		return nil, errorsx.WrapInternal(err, "删除站内信失败")
	}
	return new(emptypb.Empty), nil
}
