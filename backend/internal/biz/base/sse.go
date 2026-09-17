package biz

import (
	"context"
	"fmt"

	basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"
	"github.com/liujitcn/kratos-core/biz"
	_const "github.com/liujitcn/kratos-core/const"
	"github.com/liujitcn/kratos-core/errorsx"
	coresse "github.com/liujitcn/kratos-core/sse"
	"github.com/liujitcn/kratos-kit/database/gorm"

	"google.golang.org/protobuf/types/known/emptypb"
)

// SseCase 处理 SSE 公共业务。
type SseCase struct {
	*biz.BaseCase
	sse *coresse.SSE
}

// NewSseCase 创建 SSE 业务实例。
func NewSseCase(baseCase *biz.BaseCase, sse *coresse.SSE) *SseCase {
	return &SseCase{
		BaseCase: baseCase,
		sse:      sse,
	}
}

// SubscribeSse 订阅 SSE 事件流。
func (h *SseCase) SubscribeSse(ctx context.Context, req *basev1.SubscribeSseRequest) (*emptypb.Empty, error) {
	authInfo, err := h.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}
	channelID := req.GetChannelId()
	switch req.GetStream() {
	case "base.notification":
		channelID = fmt.Sprintf("%d:%s", authInfo.TenantId, channelID)
	case "system.admin.ops-monitoring", "system.admin.runtime-console":
		if (authInfo.RoleCode != _const.BASE_ROLE_CODE_SUPER && authInfo.RoleCode != _const.BASE_ROLE_CODE_ADMIN) || authInfo.TenantCode != gorm.DefaultTenantCode {
			return nil, errorsx.PermissionDenied("只有平台管理员可以订阅运维实时数据")
		}
	}
	err = h.sse.Serve(ctx, req.GetStream(), channelID)
	if err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}
