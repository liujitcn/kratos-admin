package admin

import (
	"context"
	"fmt"
	"sync"

	systemadminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	coreQueue "github.com/liujitcn/kratos-admin/backend/core/pkg/queue"
	coreTask "github.com/liujitcn/kratos-admin/backend/core/pkg/task"
	adminbiz "github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin/v1/dto"
	_const "github.com/liujitcn/kratos-admin/backend/internal/const"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"

	kratosErrors "github.com/go-kratos/kratos/v3/errors"
	"github.com/go-kratos/kratos/v3/log"
	"github.com/liujitcn/gorm-kit/repository"
	queueData "github.com/liujitcn/kratos-kit/queue/data"
)

const (
	// BaseTranslationTaskName 是统一机器翻译任务的稳定调用目标。
	BaseTranslationTaskName = "system.admin.BaseTranslation"
)

// BaseTranslationTask 执行菜单、字典、字典项和系统配置的机器翻译任务。
type BaseTranslationTask struct {
	translationCase *adminbiz.BaseTranslationCase
	menuRepo        *data.BaseMenuRepository
	dictRepo        *data.BaseDictRepository
	dictItemRepo    *data.BaseDictItemRepository
	configRepo      *data.BaseConfigRepository
	mu              sync.Mutex
}

// NewBaseTranslationTask 创建统一机器翻译任务执行器。
func NewBaseTranslationTask(
	translationCase *adminbiz.BaseTranslationCase,
	menuRepo *data.BaseMenuRepository,
	dictRepo *data.BaseDictRepository,
	dictItemRepo *data.BaseDictItemRepository,
	configRepo *data.BaseConfigRepository,
) *BaseTranslationTask {
	task := &BaseTranslationTask{
		translationCase: translationCase,
		menuRepo:        menuRepo,
		dictRepo:        dictRepo,
		dictItemRepo:    dictItemRepo,
		configRepo:      configRepo,
	}
	translationCase.RegisterQueueConsumer(_const.TRANSLATION, task.consumeTranslation)
	return task
}

// Task 返回交由 base_job 统一调度的任务定义。
func (t *BaseTranslationTask) Task() coreTask.Task {
	return coreTask.Task{Name: BaseTranslationTaskName, Exec: t}
}

// Exec 兼容不带上下文的任务执行接口。
func (t *BaseTranslationTask) Exec(_ map[string]string) ([]string, error) {
	return t.ExecContext(context.Background(), nil)
}

// ExecContext 扫描所有资源并补齐缺失的机器译文。
func (t *BaseTranslationTask) ExecContext(ctx context.Context, _ map[string]string) ([]string, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if !t.translationCase.DraftEnabled() {
		return []string{"机器翻译功能未启用"}, nil
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
	menuQuery := t.menuRepo.Query(ctx).BaseMenu
	menus, err := t.menuRepo.List(ctx, repository.Order(menuQuery.ID.Asc()))
	if err != nil {
		return nil, err
	}
	menuIDs := make([]int64, 0, len(menus))
	for _, menu := range menus {
		menuIDs = append(menuIDs, menu.ID)
	}
	t.translateIDs(ctx, systemadminv1.TranslationTargetType_TRANSLATION_TARGET_TYPE_BASE_MENU, menuIDs, locales, "菜单", &translatedCount, &failedCount, &firstErr)

	dictQuery := t.dictRepo.Query(ctx).BaseDict
	dicts, err := t.dictRepo.List(ctx, repository.Order(dictQuery.ID.Asc()))
	if err != nil {
		return nil, err
	}
	dictIDs := make([]int64, 0, len(dicts))
	for _, dict := range dicts {
		dictIDs = append(dictIDs, dict.ID)
	}
	t.translateIDs(ctx, systemadminv1.TranslationTargetType_TRANSLATION_TARGET_TYPE_BASE_DICT, dictIDs, locales, "字典", &translatedCount, &failedCount, &firstErr)

	dictItemQuery := t.dictItemRepo.Query(ctx).BaseDictItem
	dictItems, err := t.dictItemRepo.List(ctx, repository.Order(dictItemQuery.ID.Asc()))
	if err != nil {
		return nil, err
	}
	dictItemIDs := make([]int64, 0, len(dictItems))
	for _, item := range dictItems {
		dictItemIDs = append(dictItemIDs, item.ID)
	}
	t.translateIDs(ctx, systemadminv1.TranslationTargetType_TRANSLATION_TARGET_TYPE_BASE_DICT_ITEM, dictItemIDs, locales, "字典项", &translatedCount, &failedCount, &firstErr)

	configQuery := t.configRepo.Query(ctx).BaseConfig
	configs, err := t.configRepo.List(ctx, repository.Order(configQuery.ID.Asc()))
	if err != nil {
		return nil, err
	}
	configNameIDs := make([]int64, 0, len(configs))
	configValueIDs := make([]int64, 0, len(configs))
	for _, config := range configs {
		configNameIDs = append(configNameIDs, config.ID)
		if t.translationCase.CanTranslateConfigValue(config.Type) {
			configValueIDs = append(configValueIDs, config.ID)
		}
	}
	t.translateIDs(ctx, systemadminv1.TranslationTargetType_TRANSLATION_TARGET_TYPE_BASE_CONFIG_NAME, configNameIDs, locales, "系统配置名称", &translatedCount, &failedCount, &firstErr)
	t.translateIDs(ctx, systemadminv1.TranslationTargetType_TRANSLATION_TARGET_TYPE_BASE_CONFIG_VALUE, configValueIDs, locales, "系统配置值", &translatedCount, &failedCount, &firstErr)

	output := []string{fmt.Sprintf("生成机器译文 %d 条", translatedCount)}
	if failedCount > 0 {
		return output, fmt.Errorf("机器翻译失败 %d 条: %w", failedCount, firstErr)
	}
	return output, nil
}

// consumeTranslation 消费新增或更新资源的机器翻译消息。
func (t *BaseTranslationTask) consumeTranslation(message queueData.Message) error {
	request, err := coreQueue.Decode[dto.TranslationQueueMessage](message)
	if err != nil {
		return fmt.Errorf("解析机器翻译队列消息失败: %w", err)
	}
	if request == nil || request.TargetID <= 0 {
		return nil
	}
	return t.translateResource(context.Background(), request.TargetType, request.TargetID)
}

// translateResource 为单个翻译目标生成所有启用语言的机器译文。
func (t *BaseTranslationTask) translateResource(ctx context.Context, targetType systemadminv1.TranslationTargetType, targetID int64) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	locales, err := t.translationCase.EditableLocales(ctx)
	if err != nil {
		return err
	}
	if targetType == systemadminv1.TranslationTargetType_TRANSLATION_TARGET_TYPE_BASE_CONFIG_VALUE {
		config, findErr := t.configRepo.FindByID(ctx, targetID)
		if findErr != nil {
			return ignoreTranslationClientError(findErr)
		}
		if !t.translationCase.CanTranslateConfigValue(config.Type) {
			return nil
		}
	}
	for _, localeValue := range locales {
		err = t.translationCase.TranslateResource(ctx, targetType, targetID, localeValue)
		if err != nil && ignoreTranslationClientError(err) != nil {
			return err
		}
	}
	return nil
}

// translateIDs 批量调用统一翻译入口并统计结果。
func (t *BaseTranslationTask) translateIDs(ctx context.Context, targetType systemadminv1.TranslationTargetType, targetIDs []int64, locales []string, resourceName string, translatedCount, failedCount *int, firstErr *error) {
	var err error
	for _, targetID := range targetIDs {
		for _, localeValue := range locales {
			err = t.translationCase.TranslateResource(ctx, targetType, targetID, localeValue)
			if err == nil {
				*translatedCount = *translatedCount + 1
				continue
			}
			if ignoreTranslationClientError(err) == nil {
				continue
			}
			*failedCount = *failedCount + 1
			if *firstErr == nil {
				*firstErr = err
			}
			log.Error("机器翻译任务执行失败", "target", resourceName, "target_id", targetID, "locale", localeValue, "error", err)
		}
	}
}

// ignoreTranslationClientError 将资源不存在、参数无效和已有译文视为无需重试。
func ignoreTranslationClientError(err error) error {
	if err == nil {
		return nil
	}
	structured := kratosErrors.FromError(err)
	if structured != nil && structured.Code < 500 {
		return nil
	}
	return err
}
