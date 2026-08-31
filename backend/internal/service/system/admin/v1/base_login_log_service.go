package admin

import (
	"context"
	"fmt"

	"github.com/go-kratos/kratos/v3/log"
	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	biz "github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin"
	"github.com/liujitcn/kratos-core/errorsx"
)

// BaseLoginLogService 提供登录日志的只读管理接口。
type BaseLoginLogService struct {
	adminv1.UnimplementedBaseLoginLogServiceServer                       // 提供未实现 RPC 的默认返回。
	auditLogCase                                   *biz.BaseAuditLogCase // 负责租户隔离和日志查询业务。
}

// NewBaseLoginLogService 创建登录日志管理服务。
func NewBaseLoginLogService(auditLogCase *biz.BaseAuditLogCase) *BaseLoginLogService {
	return &BaseLoginLogService{auditLogCase: auditLogCase}
}

// PageBaseLoginLog 查询登录日志分页列表。
func (s *BaseLoginLogService) PageBaseLoginLog(ctx context.Context, req *adminv1.PageBaseLoginLogRequest) (*adminv1.PageBaseLoginLogResponse, error) {
	page, err := s.auditLogCase.PageBaseLoginLog(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("PageBaseLoginLog %v", err))
		return nil, errorsx.WrapInternal(err, "查询登录日志分页列表失败")
	}
	return page, nil
}

// GetBaseLoginLog 查询登录日志详情。
func (s *BaseLoginLogService) GetBaseLoginLog(ctx context.Context, req *adminv1.GetBaseLoginLogRequest) (*adminv1.BaseLoginLog, error) {
	item, err := s.auditLogCase.GetBaseLoginLog(ctx, req.GetId())
	if err != nil {
		log.Error(fmt.Sprintf("GetBaseLoginLog %v", err))
		return nil, errorsx.WrapInternal(err, "查询登录日志详情失败")
	}
	return item, nil
}
