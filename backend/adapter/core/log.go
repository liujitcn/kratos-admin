package core

import (
	"context"
	"errors"

	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	coredata "github.com/liujitcn/kratos-core/data"
	kitgorm "github.com/liujitcn/kratos-kit/database/gorm"
	"gorm.io/gorm"
)

// LogStoreAdapter 将 Admin 的日志生成仓储适配为 Core 审计日志接口。
type LogStoreAdapter struct {
	apiRepository    *data.BaseAPILogRepository
	policyRepository *data.BasePolicyEvaluationLogRepository
}

// NewLogStoreAdapter 使用数据库客户端创建一次内部仓储，供 Core 复用审计日志存储。
func NewLogStoreAdapter(databases map[string]*kitgorm.Client) (*LogStoreAdapter, error) {
	provider, err := data.NewData(databases)
	if err != nil {
		return nil, err
	}
	return &LogStoreAdapter{
		apiRepository:    data.NewBaseAPILogRepository(provider),
		policyRepository: data.NewBasePolicyEvaluationLogRepository(provider),
	}, nil
}

// CreateAPI 写入 API 访问日志。
func (s *LogStoreAdapter) CreateAPI(ctx context.Context, item coredata.APILogRecord) error {
	return s.apiRepository.Create(ctx, &models.BaseAPILog{
		ID:          item.ID,
		TenantID:    item.TenantID,
		TenantCode:  item.TenantCode,
		UserID:      item.UserID,
		UserName:    item.UserName,
		RequestID:   item.RequestID,
		TraceID:     item.TraceID,
		ServiceName: item.ServiceName,
		Operation:   item.Operation,
		Method:      item.Method,
		Path:        item.Path,
		StatusCode:  item.StatusCode,
		Result:      item.Result,
		ReasonCode:  item.ReasonCode,
		Reason:      item.Reason,
		LatencyMs:   item.LatencyMs,
		ClientIP:    item.ClientIP,
		UserAgent:   item.UserAgent,
		OccurredAt:  item.OccurredAt,
		CreatedAt:   item.CreatedAt,
	})
}

// ExistsAPI 判断 API 访问日志是否已写入。
func (s *LogStoreAdapter) ExistsAPI(ctx context.Context, id int64) (bool, error) {
	_, err := s.apiRepository.FindByID(ctx, id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// CreatePolicyEvaluation 写入策略评估日志。
func (s *LogStoreAdapter) CreatePolicyEvaluation(ctx context.Context, item coredata.PolicyEvaluationLogRecord) error {
	return s.policyRepository.Create(ctx, &models.BasePolicyEvaluationLog{
		ID:             item.ID,
		TenantID:       item.TenantID,
		TenantCode:     item.TenantCode,
		UserID:         item.UserID,
		UserName:       item.UserName,
		RoleID:         item.RoleID,
		RoleCode:       item.RoleCode,
		RequestID:      item.RequestID,
		TraceID:        item.TraceID,
		ClientIP:       item.ClientIP,
		Engine:         item.Engine,
		EvaluationType: item.EvaluationType,
		Resource:       item.Resource,
		Action:         item.Action,
		Project:        item.Project,
		Decision:       item.Decision,
		ReasonCode:     item.ReasonCode,
		Reason:         item.Reason,
		DurationMs:     item.DurationMs,
		CandidateCount: item.CandidateCount,
		MatchedCount:   item.MatchedCount,
		InputHash:      item.InputHash,
		OccurredAt:     item.OccurredAt,
		CreatedAt:      item.CreatedAt,
	})
}

// ExistsPolicyEvaluation 判断策略评估日志是否已写入。
func (s *LogStoreAdapter) ExistsPolicyEvaluation(ctx context.Context, id int64) (bool, error) {
	_, err := s.policyRepository.FindByID(ctx, id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

var _ coredata.LogStore = (*LogStoreAdapter)(nil)
