package core

import (
	"context"
	"strings"

	"github.com/liujitcn/gorm-kit/repository"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	_const "github.com/liujitcn/kratos-core/const"
	coredata "github.com/liujitcn/kratos-core/data"
	"github.com/liujitcn/kratos-core/resource/openapi"
	"github.com/liujitcn/kratos-core/resource/openapi/dto"
	"github.com/liujitcn/kratos-kit/database/gorm"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
)

// APIStoreAdapter 将 Admin 的 API 生成仓储适配为 Core 资源接口。
type APIStoreAdapter struct {
	apiRepository         *data.BaseAPIRepository
	translationRepository *data.BaseAPII18NRepository
	openAPI               *openapi.OpenAPI
}

// NewAPIStoreAdapter 使用数据库客户端创建一次内部仓储，供 Core 复用 API 资源存储。
func NewAPIStoreAdapter(databases map[string]*gorm.Client, openAPI *openapi.OpenAPI) (*APIStoreAdapter, error) {
	provider, err := data.NewData(databases)
	if err != nil {
		return nil, err
	}
	return &APIStoreAdapter{
		apiRepository:         data.NewBaseAPIRepository(provider),
		translationRepository: data.NewBaseAPII18NRepository(provider),
		openAPI:               openAPI,
	}, nil
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
	records := make([]*models.BaseAPI, 0, len(items)+1)
	const sseOperation = "/base.v1.SseService/SubscribeSse"
	sseIncluded := false
	for _, item := range items {
		if item == nil {
			continue
		}
		if item.Operation == sseOperation {
			sseIncluded = true
		}
		record := &models.BaseAPI{
			ID:             item.ID,
			ToolName:       item.ToolName,
			ToolPrompts:    item.ToolPrompts,
			ServiceName:    item.ServiceName,
			ServiceDesc:    item.ServiceDesc,
			Desc:           item.Desc,
			Operation:      item.Operation,
			Method:         item.Method,
			Path:           item.Path,
			TenantResponse: s.hasTenantResponse(ctx, item.Path, item.Method),
			McpStatus:      item.McpStatus,
			AgentStatus:    item.AgentStatus,
		}
		if previous := existingByOperation[item.Operation]; previous != nil {
			record.ID = previous.ID
			record.McpStatus = previous.McpStatus
			record.AgentStatus = previous.AgentStatus
			record.ToolPrompts = previous.ToolPrompts
		}
		records = append(records, record)
	}
	if !sseIncluded {
		sseRecord := &models.BaseAPI{
			ToolName:    "base_v1_sse_service_subscribe_sse",
			ToolPrompts: "[]",
			ServiceName: "base.v1.SseService",
			ServiceDesc: "Base SSE服务",
			Desc:        "订阅SSE事件流",
			Operation:   sseOperation,
			Method:      "GET",
			Path:        "/events/{stream}",
			McpStatus:   int32(_const.STATUS_STATUS_ENABLE),
			AgentStatus: int32(_const.STATUS_STATUS_ENABLE),
		}
		if previous := existingByOperation[sseOperation]; previous != nil {
			sseRecord.ID = previous.ID
			sseRecord.McpStatus = previous.McpStatus
			sseRecord.AgentStatus = previous.AgentStatus
			sseRecord.ToolPrompts = previous.ToolPrompts
		}
		records = append(records, sseRecord)
	}
	query := s.apiRepository.Query(ctx).BaseAPI
	err = s.apiRepository.Delete(ctx, repository.Unscoped(), repository.Where(query.ID.Gt(0)))
	if err != nil {
		return err
	}
	return s.apiRepository.BatchCreate(ctx, records)
}

// hasTenantResponse 判断接口响应中是否包含租户字段消息。
func (s *APIStoreAdapter) hasTenantResponse(ctx context.Context, path, method string) bool {
	if s.openAPI == nil || !strings.EqualFold(method, "GET") {
		return false
	}
	document, err := s.openAPI.GetOperation(ctx, path, method)
	if err != nil {
		return false
	}
	for _, response := range document.Responses {
		if response != nil && schemaContainsTenantMessage(response.Body) {
			return true
		}
	}
	return false
}

// schemaContainsTenantMessage 判断字段树是否引用包含 tenant_id 的 Proto 消息。
func schemaContainsTenantMessage(schema *dto.OpenAPISchema) bool {
	if schema == nil {
		return false
	}
	name := schema.Ref
	if index := strings.LastIndex(name, "/"); index >= 0 {
		name = name[index+1:]
	}
	if name != "" {
		messageType, err := protoregistry.GlobalTypes.FindMessageByName(protoreflect.FullName(name))
		if err == nil {
			tenantField := messageType.Descriptor().Fields().ByName("tenant_id")
			if tenantField != nil && tenantField.Kind() == protoreflect.Int64Kind {
				return true
			}
		}
	}
	for _, child := range schema.Children {
		if schemaContainsTenantMessage(child) {
			return true
		}
	}
	return false
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
