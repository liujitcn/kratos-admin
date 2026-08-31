package admin

import (
	"context"
	"fmt"

	"github.com/go-kratos/kratos/v3/log"
	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	biz "github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin"
	"github.com/liujitcn/kratos-core/errorsx"
)

// BaseApiLogService 提供 API 访问日志的只读管理接口。
type BaseApiLogService struct {
	adminv1.UnimplementedBaseApiLogServiceServer                       // 提供未实现 RPC 的默认返回。
	auditLogCase                                 *biz.BaseAuditLogCase // 负责租户隔离和日志查询业务。
}

// NewBaseApiLogService 创建 API 访问日志管理服务。
func NewBaseApiLogService(auditLogCase *biz.BaseAuditLogCase) *BaseApiLogService {
	return &BaseApiLogService{auditLogCase: auditLogCase}
}

// PageBaseApiLog 查询 API 访问日志分页列表。
func (s *BaseApiLogService) PageBaseApiLog(ctx context.Context, req *adminv1.PageBaseApiLogRequest) (*adminv1.PageBaseApiLogResponse, error) {
	page, err := s.auditLogCase.PageBaseApiLog(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("PageBaseApiLog %v", err))
		return nil, errorsx.WrapInternal(err, "查询 API 访问日志分页列表失败")
	}
	return page, nil
}

// GetBaseApiLog 查询 API 访问日志详情。
func (s *BaseApiLogService) GetBaseApiLog(ctx context.Context, req *adminv1.GetBaseApiLogRequest) (*adminv1.BaseApiLog, error) {
	item, err := s.auditLogCase.GetBaseApiLog(ctx, req.GetId())
	if err != nil {
		log.Error(fmt.Sprintf("GetBaseApiLog %v", err))
		return nil, errorsx.WrapInternal(err, "查询 API 访问日志详情失败")
	}
	return item, nil
}
