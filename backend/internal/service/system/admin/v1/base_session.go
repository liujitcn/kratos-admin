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

// BaseSessionService 提供 Admin 会话管理接口。
type BaseSessionService struct {
	adminv1.UnimplementedBaseSessionServiceServer                      // 提供未实现 RPC 的默认返回。
	baseSessionCase                               *biz.BaseSessionCase // 负责当前会话查询和令牌撤销。
}

// NewBaseSessionService 创建 Admin 会话管理服务。
func NewBaseSessionService(baseSessionCase *biz.BaseSessionCase) *BaseSessionService {
	return &BaseSessionService{baseSessionCase: baseSessionCase}
}

// GetCurrentBaseSession 查询当前用户会话。
func (s *BaseSessionService) GetCurrentBaseSession(ctx context.Context, req *adminv1.GetCurrentBaseSessionRequest) (*adminv1.BaseSession, error) {
	res, err := s.baseSessionCase.GetCurrentBaseSession(ctx)
	if err != nil {
		log.Error(fmt.Sprintf("GetCurrentBaseSession %v", err))
		return nil, errorsx.WrapInternal(err, "查询当前会话失败")
	}
	return res, nil
}

// RevokeAllBaseSessions 撤销当前用户的全部会话。
func (s *BaseSessionService) RevokeAllBaseSessions(ctx context.Context, req *adminv1.RevokeAllBaseSessionsRequest) (*emptypb.Empty, error) {
	err := s.baseSessionCase.RevokeAllBaseSessions(ctx)
	if err != nil {
		log.Error(fmt.Sprintf("RevokeAllBaseSessions %v", err))
		return nil, errorsx.WrapInternal(err, "撤销全部会话失败")
	}
	return new(emptypb.Empty), nil
}
