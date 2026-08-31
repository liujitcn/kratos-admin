package admin

import (
	"context"
	"fmt"

	"github.com/go-kratos/kratos/v3/log"
	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	biz "github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin"
	"github.com/liujitcn/kratos-core/errorsx"
)

// BasePermissionLogService 提供权限日志的只读管理接口。
type BasePermissionLogService struct {
	adminv1.UnimplementedBasePermissionLogServiceServer                       // 提供未实现 RPC 的默认返回。
	auditLogCase                                        *biz.BaseAuditLogCase // 负责租户隔离和日志查询业务。
}

// NewBasePermissionLogService 创建权限日志管理服务。
func NewBasePermissionLogService(auditLogCase *biz.BaseAuditLogCase) *BasePermissionLogService {
	return &BasePermissionLogService{auditLogCase: auditLogCase}
}

// PageBasePermissionLog 查询权限日志分页列表。
func (s *BasePermissionLogService) PageBasePermissionLog(ctx context.Context, req *adminv1.PageBasePermissionLogRequest) (*adminv1.PageBasePermissionLogResponse, error) {
	page, err := s.auditLogCase.PageBasePermissionLog(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("PageBasePermissionLog %v", err))
		return nil, errorsx.WrapInternal(err, "查询权限日志分页列表失败")
	}
	return page, nil
}

// GetBasePermissionLog 查询权限日志详情。
func (s *BasePermissionLogService) GetBasePermissionLog(ctx context.Context, req *adminv1.GetBasePermissionLogRequest) (*adminv1.BasePermissionLog, error) {
	item, err := s.auditLogCase.GetBasePermissionLog(ctx, req.GetId())
	if err != nil {
		log.Error(fmt.Sprintf("GetBasePermissionLog %v", err))
		return nil, errorsx.WrapInternal(err, "查询权限日志详情失败")
	}
	return item, nil
}
