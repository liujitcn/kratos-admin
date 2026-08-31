package admin

import (
	"context"
	"fmt"

	"github.com/go-kratos/kratos/v3/log"
	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	biz "github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin"
	"github.com/liujitcn/kratos-core/errorsx"
)

// BaseDataAccessLogService 提供数据访问日志的只读管理接口。
type BaseDataAccessLogService struct {
	adminv1.UnimplementedBaseDataAccessLogServiceServer                       // 提供未实现 RPC 的默认返回。
	auditLogCase                                        *biz.BaseAuditLogCase // 负责租户隔离和日志查询业务。
}

// NewBaseDataAccessLogService 创建数据访问日志管理服务。
func NewBaseDataAccessLogService(auditLogCase *biz.BaseAuditLogCase) *BaseDataAccessLogService {
	return &BaseDataAccessLogService{auditLogCase: auditLogCase}
}

// PageBaseDataAccessLog 查询数据访问日志分页列表。
func (s *BaseDataAccessLogService) PageBaseDataAccessLog(ctx context.Context, req *adminv1.PageBaseDataAccessLogRequest) (*adminv1.PageBaseDataAccessLogResponse, error) {
	page, err := s.auditLogCase.PageBaseDataAccessLog(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("PageBaseDataAccessLog %v", err))
		return nil, errorsx.WrapInternal(err, "查询数据访问日志分页列表失败")
	}
	return page, nil
}

// GetBaseDataAccessLog 查询数据访问日志详情。
func (s *BaseDataAccessLogService) GetBaseDataAccessLog(ctx context.Context, req *adminv1.GetBaseDataAccessLogRequest) (*adminv1.BaseDataAccessLog, error) {
	item, err := s.auditLogCase.GetBaseDataAccessLog(ctx, req.GetId())
	if err != nil {
		log.Error(fmt.Sprintf("GetBaseDataAccessLog %v", err))
		return nil, errorsx.WrapInternal(err, "查询数据访问日志详情失败")
	}
	return item, nil
}
