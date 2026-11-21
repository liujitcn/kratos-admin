package core

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strings"

	"gorm.io/gorm/clause"

	"github.com/liujitcn/gorm-kit/repository"
	basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"
	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/bi/admin/v1"
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
	menuRepository        *data.BaseMenuRepository
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
		menuRepository:        data.NewBaseMenuRepository(provider),
		openAPI:               openAPI,
	}, nil
}

// ReplaceAll 替换 API 快照、保留工具配置，并在权限重建前补齐内置 AI 和 BI 入口的接口权限。
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
	eventIncluded := false
	refinementIncluded := false
	for _, item := range items {
		if item == nil {
			continue
		}
		switch item.Operation {
		case sseOperation:
			sseIncluded = true
		case basev1.OperationAiMessageServiceListAiMessageEvent:
			eventIncluded = true
		case adminv1.OperationBiQueryServiceRefineBiQuery:
			refinementIncluded = true
		}
		record := &models.BaseAPI{
			ID:             item.ID,
			ToolName:       item.ToolName,
			ToolPrompts:    normalizeJSONArray(item.ToolPrompts),
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
			record.ToolPrompts = normalizeJSONArray(previous.ToolPrompts)
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
			sseRecord.ToolPrompts = normalizeJSONArray(previous.ToolPrompts)
		}
		records = append(records, sseRecord)
	}
	query := s.apiRepository.Query(ctx).BaseAPI
	err = s.apiRepository.Delete(ctx, repository.Unscoped(), repository.Where(query.ID.Gt(0)))
	if err != nil {
		return err
	}
	err = s.apiRepository.BatchCreate(ctx, records)
	if err != nil {
		return err
	}
	if eventIncluded {
		err = s.syncBuiltinMenuOperation(ctx, basev1.OperationAiMessageServiceListAiMessageEvent, map[int64][]string{
			90990000: {basev1.OperationAiMessageServiceSendAiMessage},
			90991000: {basev1.OperationAiMessageServiceSendAiMessage},
			99101000: {basev1.OperationAiMessageServiceSendAiMessage},
		})
		if err != nil {
			return err
		}
	}
	if refinementIncluded {
		return s.syncBuiltinMenuOperation(ctx, adminv1.OperationBiQueryServiceRefineBiQuery, map[int64][]string{
			90991000: {basev1.OperationAiMessageServiceSendAiMessage, adminv1.OperationBiQueryRecordServiceGetBiQuery},
			91020000: {adminv1.OperationBiQueryServiceAskBiQuery, adminv1.OperationBiQueryRecordServiceGetBiQuery},
		})
	}
	return nil
}

// syncBuiltinMenuOperation 仅为仍保留前置授权的内置菜单追加接口，保留现有角色和自定义权限。
func (s *APIStoreAdapter) syncBuiltinMenuOperation(ctx context.Context, operation string, required map[int64][]string) error {
	query := s.menuRepository.Query(ctx).BaseMenu
	menus, err := query.WithContext(ctx).
		Where(query.ID.In(slices.Sorted(maps.Keys(required))...), query.Status.Eq(int32(_const.STATUS_STATUS_ENABLE))).
		Clauses(clause.Locking{Strength: "UPDATE"}).Find()
	if err != nil {
		return err
	}
	for _, menu := range menus {
		var operations []string
		if err = json.Unmarshal([]byte(menu.API), &operations); err != nil {
			return fmt.Errorf("解析内置菜单 %d 权限: %w", menu.ID, err)
		}
		if slices.Contains(operations, operation) || slices.ContainsFunc(required[menu.ID], func(prerequisite string) bool {
			return !slices.Contains(operations, prerequisite)
		}) {
			continue
		}
		var payload []byte
		payload, err = json.Marshal(append(operations, operation))
		if err != nil {
			return fmt.Errorf("编码内置菜单 %d 权限: %w", menu.ID, err)
		}
		_, err = query.WithContext(ctx).Where(query.ID.Eq(menu.ID)).UpdateSimple(query.API.Value(string(payload)))
		if err != nil {
			return err
		}
	}
	return nil
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
	return slices.ContainsFunc(schema.Children, schemaContainsTenantMessage)
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
			ToolPrompts: normalizeJSONArray(item.ToolPrompts),
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

// normalizeJSONArray 为数组型 JSON 字段补齐空数组默认值。
func normalizeJSONArray(raw string) string {
	if raw == "" || raw == "null" {
		return "[]"
	}
	return raw
}
