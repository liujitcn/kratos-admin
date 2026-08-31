package admin

import (
	"context"
	"fmt"

	"github.com/go-kratos/kratos/v3/log"
	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	biz "github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin"
	"github.com/liujitcn/kratos-core/errorsx"
)

// BasePolicyEvaluationLogService 提供策略评估日志的只读管理接口。
type BasePolicyEvaluationLogService struct {
	adminv1.UnimplementedBasePolicyEvaluationLogServiceServer                       // 提供未实现 RPC 的默认返回。
	auditLogCase                                              *biz.BaseAuditLogCase // 负责租户隔离和日志查询业务。
}

// NewBasePolicyEvaluationLogService 创建策略评估日志管理服务。
func NewBasePolicyEvaluationLogService(auditLogCase *biz.BaseAuditLogCase) *BasePolicyEvaluationLogService {
	return &BasePolicyEvaluationLogService{auditLogCase: auditLogCase}
}

// PageBasePolicyEvaluationLog 查询策略评估日志分页列表。
func (s *BasePolicyEvaluationLogService) PageBasePolicyEvaluationLog(ctx context.Context, req *adminv1.PageBasePolicyEvaluationLogRequest) (*adminv1.PageBasePolicyEvaluationLogResponse, error) {
	page, err := s.auditLogCase.PageBasePolicyEvaluationLog(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("PageBasePolicyEvaluationLog %v", err))
		return nil, errorsx.WrapInternal(err, "查询策略评估日志分页列表失败")
	}
	return page, nil
}

// GetBasePolicyEvaluationLog 查询策略评估日志详情。
func (s *BasePolicyEvaluationLogService) GetBasePolicyEvaluationLog(ctx context.Context, req *adminv1.GetBasePolicyEvaluationLogRequest) (*adminv1.BasePolicyEvaluationLog, error) {
	item, err := s.auditLogCase.GetBasePolicyEvaluationLog(ctx, req.GetId())
	if err != nil {
		log.Error(fmt.Sprintf("GetBasePolicyEvaluationLog %v", err))
		return nil, errorsx.WrapInternal(err, "查询策略评估日志详情失败")
	}
	return item, nil
}
