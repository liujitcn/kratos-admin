package core

import (
	"context"

	"github.com/liujitcn/gorm-kit/repository"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	coredata "github.com/liujitcn/kratos-core/data"
)

// APIStoreAdapter 将 Admin 的 API 生成仓储适配为 Core 资源接口。
type APIStoreAdapter struct {
	apiRepository         *data.BaseAPIRepository
	translationRepository *data.BaseAPII18NRepository
}

// NewAPIStoreAdapter 创建 Core API 资源存储适配器。
func NewAPIStoreAdapter(
	apiRepository *data.BaseAPIRepository,
	translationRepository *data.BaseAPII18NRepository,
) *APIStoreAdapter {
	return &APIStoreAdapter{
		apiRepository:         apiRepository,
		translationRepository: translationRepository,
	}
}

// ReplaceAll 替换 API 快照并保留已有工具运行时配置。
func (s *APIStoreAdapter) ReplaceAll(ctx context.Context, items []*coredata.APIRecord) error {
	var existing []*models.BaseAPI
	var err error
	if len(items) > 0 {
		existing, err = s.apiRepository.List(ctx)
		if err != nil {
			return err
		}
	}
	existingByOperation := make(map[string]*models.BaseAPI, len(existing))
	for _, item := range existing {
		if item == nil || item.Operation == "" {
			continue
		}
		if _, exists := existingByOperation[item.Operation]; !exists {
			existingByOperation[item.Operation] = item
		}
	}
	records := make([]*models.BaseAPI, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		record := &models.BaseAPI{
			ID:          item.ID,
			ToolName:    item.ToolName,
			ToolPrompts: item.ToolPrompts,
			ServiceName: item.ServiceName,
			ServiceDesc: item.ServiceDesc,
			Desc:        item.Desc,
			Operation:   item.Operation,
			Method:      item.Method,
			Path:        item.Path,
			McpStatus:   item.McpStatus,
			AgentStatus: item.AgentStatus,
		}
		if previous := existingByOperation[item.Operation]; previous != nil {
			record.ID = previous.ID
			record.McpStatus = previous.McpStatus
			record.AgentStatus = previous.AgentStatus
			record.ToolPrompts = previous.ToolPrompts
		}
		records = append(records, record)
	}
	query := s.apiRepository.Query(ctx).BaseAPI
	err = s.apiRepository.Delete(ctx, repository.Unscoped(), repository.Where(query.ID.Gt(0)))
	if err != nil {
		return err
	}
	return s.apiRepository.BatchCreate(ctx, records)
}

// ReplaceAllTranslations 替换 API 多语言快照。
func (s *APIStoreAdapter) ReplaceAllTranslations(ctx context.Context, items []*coredata.APITranslationRecord) error {
	records := make([]*models.BaseAPII18N, 0, len(items))
	for _, item := range items {
		if item == nil || item.Locale == "" {
			continue
		}
		records = append(records, &models.BaseAPII18N{
			Operation:   item.Operation,
			Locale:      item.Locale,
			ToolPrompts: item.ToolPrompts,
			ServiceDesc: item.ServiceDesc,
			Desc:        item.Desc,
		})
	}
	query := s.translationRepository.Query(ctx).BaseAPII18N
	err := s.translationRepository.Delete(ctx, repository.Where(query.ID.Gt(0)))
	if err != nil {
		return err
	}
	return s.translationRepository.BatchCreate(ctx, records)
}

// ListForPolicy 查询权限重建所需的 API 字段。
func (s *APIStoreAdapter) ListForPolicy(ctx context.Context) ([]*coredata.APIPolicyRecord, error) {
	items, err := s.apiRepository.List(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]*coredata.APIPolicyRecord, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		result = append(result, &coredata.APIPolicyRecord{Operation: item.Operation, Method: item.Method})
	}
	return result, nil
}

var _ coredata.APIStore = (*APIStoreAdapter)(nil)
