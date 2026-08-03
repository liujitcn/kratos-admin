package admin

import (
	"context"
	"fmt"
	"sync"
	"time"

	systemadminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	coreTask "github.com/liujitcn/kratos-admin/backend/core/pkg/task"
	adminbiz "github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin/v1/dto"
	_const "github.com/liujitcn/kratos-admin/backend/internal/const"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"

	"github.com/go-kratos/kratos/v3/log"
	"github.com/liujitcn/gorm-kit/repository"
)

const (
	// BaseTranslationTaskName 是统一机器翻译任务的稳定调用目标。
	BaseTranslationTaskName = "system.admin.BaseTranslation"
)

// BaseTranslationTask 执行菜单、字典、字典项和系统配置的机器翻译草稿任务。
type BaseTranslationTask struct {
	translationCase         *adminbiz.BaseTranslationCase
	menuRepo                *data.BaseMenuRepository
	menuTranslationRepo     *data.BaseMenuTranslationRepository
	dictRepo                *data.BaseDictRepository
	dictTranslationRepo     *data.BaseDictTranslationRepository
	dictItemRepo            *data.BaseDictItemRepository
	dictItemTranslationRepo *data.BaseDictItemTranslationRepository
	configRepo              *data.BaseConfigRepository
	configTranslationRepo   *data.BaseConfigTranslationRepository
	mu                      sync.Mutex
}

// NewBaseTranslationTask 创建统一机器翻译任务执行器。
func NewBaseTranslationTask(
	translationCase *adminbiz.BaseTranslationCase,
	menuRepo *data.BaseMenuRepository,
	menuTranslationRepo *data.BaseMenuTranslationRepository,
	dictRepo *data.BaseDictRepository,
	dictTranslationRepo *data.BaseDictTranslationRepository,
	dictItemRepo *data.BaseDictItemRepository,
	dictItemTranslationRepo *data.BaseDictItemTranslationRepository,
	configRepo *data.BaseConfigRepository,
	configTranslationRepo *data.BaseConfigTranslationRepository,
) *BaseTranslationTask {
	return &BaseTranslationTask{
		translationCase:         translationCase,
		menuRepo:                menuRepo,
		menuTranslationRepo:     menuTranslationRepo,
		dictRepo:                dictRepo,
		dictTranslationRepo:     dictTranslationRepo,
		dictItemRepo:            dictItemRepo,
		dictItemTranslationRepo: dictItemTranslationRepo,
		configRepo:              configRepo,
		configTranslationRepo:   configTranslationRepo,
	}
}

// Task 返回交由 base_job 统一调度的任务定义。
func (t *BaseTranslationTask) Task() coreTask.Task {
	return coreTask.Task{Name: BaseTranslationTaskName, Exec: t}
}

// Exec 兼容不带上下文的任务执行接口。
func (t *BaseTranslationTask) Exec(_ map[string]string) ([]string, error) {
	return t.ExecContext(context.Background(), nil)
}

// ExecContext 执行统一机器翻译任务并保留服务生命周期上下文。
func (t *BaseTranslationTask) ExecContext(ctx context.Context, _ map[string]string) ([]string, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if !t.translationCase.DraftEnabled() {
		return []string{"机器翻译草稿功能未启用"}, nil
	}
	locales, err := t.translationCase.EditableLocales(ctx)
	if err != nil {
		return nil, err
	}
	if len(locales) == 0 {
		return []string{"没有启用的目标语言"}, nil
	}

	translatedCount := 0
	failedCount := 0
	var firstErr error
	err = t.translateMenus(ctx, locales, &translatedCount, &failedCount, &firstErr)
	if err != nil {
		return nil, err
	}
	err = t.translateDicts(ctx, locales, &translatedCount, &failedCount, &firstErr)
	if err != nil {
		return nil, err
	}
	err = t.translateDictItems(ctx, locales, &translatedCount, &failedCount, &firstErr)
	if err != nil {
		return nil, err
	}
	err = t.translateConfigs(ctx, locales, &translatedCount, &failedCount, &firstErr)
	if err != nil {
		return nil, err
	}

	output := []string{fmt.Sprintf("生成机器译文 %d 条", translatedCount)}
	if failedCount > 0 {
		return output, fmt.Errorf("机器翻译失败 %d 条: %w", failedCount, firstErr)
	}
	return output, nil
}

// translateMenus 扫描缺失或待处理的菜单译文。
func (t *BaseTranslationTask) translateMenus(ctx context.Context, locales []string, translatedCount, failedCount *int, firstErr *error) error {
	query := t.menuRepo.Query(ctx).BaseMenu
	menus, err := t.menuRepo.List(ctx, repository.Order(query.ID.Asc()))
	if err != nil {
		return err
	}
	translationQuery := t.menuTranslationRepo.Query(ctx).BaseMenuTranslation
	rows, err := t.menuTranslationRepo.List(ctx, repository.Order(translationQuery.ID.Asc()))
	if err != nil {
		return err
	}
	rowMap := make(map[dto.TranslationKey]*models.BaseMenuTranslation, len(rows))
	for _, row := range rows {
		rowMap[dto.TranslationKey{ResourceID: row.MenuID, Locale: row.Locale}] = row
	}
	for _, menu := range menus {
		if menu.Meta == "" {
			continue
		}
		for _, localeValue := range locales {
			row := rowMap[dto.TranslationKey{ResourceID: menu.ID, Locale: localeValue}]
			if !menuTranslationNeedsDraft(row) {
				continue
			}
			err = t.generateDraft(ctx, systemadminv1.TranslationResourceType_TRANSLATION_RESOURCE_TYPE_MENU, menu.ID, systemadminv1.BaseConfigTranslationField_BASE_CONFIG_TRANSLATION_FIELD_UNSPECIFIED, localeValue)
			if err != nil {
				*failedCount = *failedCount + 1
				if *firstErr == nil {
					*firstErr = err
				}
				log.Error("菜单翻译任务执行失败", "resource_id", menu.ID, "locale", localeValue, "error", err)
				continue
			}
			*translatedCount = *translatedCount + 1
		}
	}
	return nil
}

// translateDicts 扫描缺失或待处理的字典译文。
func (t *BaseTranslationTask) translateDicts(ctx context.Context, locales []string, translatedCount, failedCount *int, firstErr *error) error {
	query := t.dictRepo.Query(ctx).BaseDict
	dicts, err := t.dictRepo.List(ctx, repository.Order(query.ID.Asc()))
	if err != nil {
		return err
	}
	translationQuery := t.dictTranslationRepo.Query(ctx).BaseDictTranslation
	rows, err := t.dictTranslationRepo.List(ctx, repository.Order(translationQuery.ID.Asc()))
	if err != nil {
		return err
	}
	rowMap := make(map[dto.TranslationKey]*models.BaseDictTranslation, len(rows))
	for _, row := range rows {
		rowMap[dto.TranslationKey{ResourceID: row.DictID, Locale: row.Locale}] = row
	}
	for _, dict := range dicts {
		if dict.Name == "" {
			continue
		}
		for _, localeValue := range locales {
			row := rowMap[dto.TranslationKey{ResourceID: dict.ID, Locale: localeValue}]
			if !dictTranslationNeedsDraft(row) {
				continue
			}
			err = t.generateDraft(ctx, systemadminv1.TranslationResourceType_TRANSLATION_RESOURCE_TYPE_DICT, dict.ID, systemadminv1.BaseConfigTranslationField_BASE_CONFIG_TRANSLATION_FIELD_UNSPECIFIED, localeValue)
			if err != nil {
				*failedCount = *failedCount + 1
				if *firstErr == nil {
					*firstErr = err
				}
				log.Error("字典翻译任务执行失败", "resource_id", dict.ID, "locale", localeValue, "error", err)
				continue
			}
			*translatedCount = *translatedCount + 1
		}
	}
	return nil
}

// translateDictItems 扫描缺失或待处理的字典项译文。
func (t *BaseTranslationTask) translateDictItems(ctx context.Context, locales []string, translatedCount, failedCount *int, firstErr *error) error {
	query := t.dictItemRepo.Query(ctx).BaseDictItem
	items, err := t.dictItemRepo.List(ctx, repository.Order(query.ID.Asc()))
	if err != nil {
		return err
	}
	translationQuery := t.dictItemTranslationRepo.Query(ctx).BaseDictItemTranslation
	rows, err := t.dictItemTranslationRepo.List(ctx, repository.Order(translationQuery.ID.Asc()))
	if err != nil {
		return err
	}
	rowMap := make(map[dto.TranslationKey]*models.BaseDictItemTranslation, len(rows))
	for _, row := range rows {
		rowMap[dto.TranslationKey{ResourceID: row.DictItemID, Locale: row.Locale}] = row
	}
	for _, item := range items {
		if item.Label == "" {
			continue
		}
		for _, localeValue := range locales {
			row := rowMap[dto.TranslationKey{ResourceID: item.ID, Locale: localeValue}]
			if !dictItemTranslationNeedsDraft(row) {
				continue
			}
			err = t.generateDraft(ctx, systemadminv1.TranslationResourceType_TRANSLATION_RESOURCE_TYPE_DICT_ITEM, item.ID, systemadminv1.BaseConfigTranslationField_BASE_CONFIG_TRANSLATION_FIELD_UNSPECIFIED, localeValue)
			if err != nil {
				*failedCount = *failedCount + 1
				if *firstErr == nil {
					*firstErr = err
				}
				log.Error("字典项翻译任务执行失败", "resource_id", item.ID, "locale", localeValue, "error", err)
				continue
			}
			*translatedCount = *translatedCount + 1
		}
	}
	return nil
}

// translateConfigs 扫描缺失或待处理的系统配置名称和值译文。
func (t *BaseTranslationTask) translateConfigs(ctx context.Context, locales []string, translatedCount, failedCount *int, firstErr *error) error {
	query := t.configRepo.Query(ctx).BaseConfig
	configs, err := t.configRepo.List(ctx, repository.Order(query.ID.Asc()))
	if err != nil {
		return err
	}
	translationQuery := t.configTranslationRepo.Query(ctx).BaseConfigTranslation
	rows, err := t.configTranslationRepo.List(ctx, repository.Order(translationQuery.ID.Asc()))
	if err != nil {
		return err
	}
	rowMap := make(map[int64]map[string]*models.BaseConfigTranslation, len(rows))
	for _, row := range rows {
		if rowMap[row.ConfigID] == nil {
			rowMap[row.ConfigID] = make(map[string]*models.BaseConfigTranslation)
		}
		rowMap[row.ConfigID][row.Locale+"\x00"+row.Field] = row
	}
	for _, config := range configs {
		fields := []systemadminv1.BaseConfigTranslationField{systemadminv1.BaseConfigTranslationField_BASE_CONFIG_TRANSLATION_FIELD_NAME}
		if t.translationCase.CanTranslateConfigValue(config.Type) {
			fields = append(fields, systemadminv1.BaseConfigTranslationField_BASE_CONFIG_TRANSLATION_FIELD_VALUE)
		}
		for _, field := range fields {
			fieldValue := "name"
			source := config.Name
			if field == systemadminv1.BaseConfigTranslationField_BASE_CONFIG_TRANSLATION_FIELD_VALUE {
				fieldValue = "value"
				source = config.Value
			}
			if source == "" {
				continue
			}
			for _, localeValue := range locales {
				row := rowMap[config.ID][localeValue+"\x00"+fieldValue]
				if !configTranslationNeedsDraft(row) {
					continue
				}
				err = t.generateDraft(ctx, systemadminv1.TranslationResourceType_TRANSLATION_RESOURCE_TYPE_CONFIG, config.ID, field, localeValue)
				if err != nil {
					*failedCount = *failedCount + 1
					if *firstErr == nil {
						*firstErr = err
					}
					log.Error("系统配置翻译任务执行失败", "resource_id", config.ID, "field", fieldValue, "locale", localeValue, "error", err)
					continue
				}
				*translatedCount = *translatedCount + 1
			}
		}
	}
	return nil
}

// generateDraft 调用统一翻译业务入口生成单条机器译文草稿。
func (t *BaseTranslationTask) generateDraft(ctx context.Context, resourceType systemadminv1.TranslationResourceType, resourceID int64, field systemadminv1.BaseConfigTranslationField, localeValue string) error {
	_, err := t.translationCase.GenerateTranslationDraft(ctx, &systemadminv1.GenerateTranslationDraftRequest{
		ResourceType: resourceType,
		ResourceId:   resourceID,
		TargetLocale: localeValue,
		Field:        field,
	})
	return err
}

// translationNeedsDraft 判断译文记录是否允许机器任务生成或覆盖。
func translationNeedsDraft(status string, reviewedBy int64, reviewedAt time.Time) bool {
	if reviewedBy != 0 || !reviewedAt.IsZero() {
		return false
	}
	return status == "" || status == _const.TRANSLATION_STATUS_PENDING
}

// menuTranslationNeedsDraft 判断菜单翻译记录是否允许机器任务处理。
func menuTranslationNeedsDraft(row *models.BaseMenuTranslation) bool {
	if row == nil {
		return true
	}
	return translationNeedsDraft(row.TranslationStatus, row.ReviewedBy, row.ReviewedAt)
}

// dictTranslationNeedsDraft 判断字典翻译记录是否允许机器任务处理。
func dictTranslationNeedsDraft(row *models.BaseDictTranslation) bool {
	if row == nil {
		return true
	}
	return translationNeedsDraft(row.TranslationStatus, row.ReviewedBy, row.ReviewedAt)
}

// dictItemTranslationNeedsDraft 判断字典项翻译记录是否允许机器任务处理。
func dictItemTranslationNeedsDraft(row *models.BaseDictItemTranslation) bool {
	if row == nil {
		return true
	}
	return translationNeedsDraft(row.TranslationStatus, row.ReviewedBy, row.ReviewedAt)
}

// configTranslationNeedsDraft 判断系统配置翻译记录是否允许机器任务处理。
func configTranslationNeedsDraft(row *models.BaseConfigTranslation) bool {
	if row == nil {
		return true
	}
	return translationNeedsDraft(row.TranslationStatus, row.ReviewedBy, row.ReviewedAt)
}
