package admin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"github.com/liujitcn/go-utils/translator"
	systemadminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	coreQueue "github.com/liujitcn/kratos-admin/backend/core/pkg/queue"
	coreTask "github.com/liujitcn/kratos-admin/backend/core/pkg/task"
	adminbiz "github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin/v1/dto"
	_const "github.com/liujitcn/kratos-admin/backend/internal/const"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	backendI18n "github.com/liujitcn/kratos-admin/backend/internal/i18n"

	kratosErrors "github.com/go-kratos/kratos/v3/errors"
	"github.com/go-kratos/kratos/v3/log"
	"github.com/liujitcn/gorm-kit/repository"
	"github.com/liujitcn/kratos-admin/backend/core/pkg/errorsx"
	queueData "github.com/liujitcn/kratos-kit/queue/data"
	"gorm.io/gorm"
)

const translationDraftMaxBytes = 2000

const (
	// BaseTranslationTaskName 是统一机器翻译任务的稳定调用目标。
	BaseTranslationTaskName = "system.admin.BaseTranslation"
)

// BaseTranslationTask 执行菜单、字典、字典项和系统配置的机器翻译任务。
type BaseTranslationTask struct {
	translationCase *adminbiz.BaseTranslationCase
	languageCase    *adminbiz.BaseLanguageCase
	draftTranslator translator.Translator
	menuRepo        *data.BaseMenuRepository
	dictRepo        *data.BaseDictRepository
	dictItemRepo    *data.BaseDictItemRepository
	configRepo      *data.BaseConfigRepository
	mu              sync.Mutex
}

// NewBaseTranslationTask 创建统一机器翻译任务执行器。
func NewBaseTranslationTask(
	translationCase *adminbiz.BaseTranslationCase,
	languageCase *adminbiz.BaseLanguageCase,
	draftTranslator translator.Translator,
	menuRepo *data.BaseMenuRepository,
	dictRepo *data.BaseDictRepository,
	dictItemRepo *data.BaseDictItemRepository,
	configRepo *data.BaseConfigRepository,
) *BaseTranslationTask {
	task := &BaseTranslationTask{
		translationCase: translationCase,
		languageCase:    languageCase,
		draftTranslator: draftTranslator,
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
	if t.draftTranslator == nil {
		return []string{"机器翻译功能未启用"}, nil
	}
	locales, primaryLocale, _, err := t.languageCase.Locales(ctx)
	if err != nil {
		return nil, err
	}
	locales = editableLocales(locales, primaryLocale)
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
		if isTranslatableConfigType(config.Type) {
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
	if request.SourceLocale != "" && request.TargetLocale != "" {
		err = t.translateOne(context.Background(), request.TargetType, request.TargetID, request.SourceLocale, request.TargetLocale)
		if err != nil && ignoreTranslationClientError(err) != nil {
			return err
		}
		return nil
	}
	return t.translateResource(context.Background(), request.TargetType, request.TargetID)
}

// translateResource 为单个翻译目标生成所有启用语言的机器译文。
func (t *BaseTranslationTask) translateResource(ctx context.Context, targetType systemadminv1.TranslationTargetType, targetID int64) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	locales, primaryLocale, _, err := t.languageCase.Locales(ctx)
	if err != nil {
		return err
	}
	locales = editableLocales(locales, primaryLocale)
	if targetType == systemadminv1.TranslationTargetType_TRANSLATION_TARGET_TYPE_BASE_CONFIG_VALUE {
		config, findErr := t.configRepo.FindByID(ctx, targetID)
		if findErr != nil {
			return ignoreTranslationClientError(findErr)
		}
		if !isTranslatableConfigType(config.Type) {
			return nil
		}
	}
	for _, localeValue := range locales {
		err = t.translateOne(ctx, targetType, targetID, primaryLocale, localeValue)
		if err != nil && ignoreTranslationClientError(err) != nil {
			return err
		}
	}
	return nil
}

// translateOne 为单个资源生成指定语言的机器译文，不覆盖已有非空内容。
func (t *BaseTranslationTask) translateOne(ctx context.Context, targetType systemadminv1.TranslationTargetType, targetID int64, sourceLocale string, targetLocale string) error {
	if t.draftTranslator == nil {
		return errorsx.PermissionDenied("机器翻译功能未启用")
	}
	if targetID <= 0 {
		return errorsx.InvalidArgument("目标资源ID不能为空")
	}
	locales, primaryLocale, _, err := t.languageCase.Locales(ctx)
	if err != nil {
		return err
	}
	if sourceLocale == "" {
		sourceLocale = primaryLocale
	}
	if !containsLocale(locales, sourceLocale) || !containsLocale(locales, targetLocale) || sourceLocale == targetLocale {
		return errorsx.InvalidArgument("源语言和目标语言必须是不同的已启用语言")
	}
	source, err := t.translationSource(ctx, targetType, targetID)
	if err != nil {
		return err
	}
	if source.Text == "" {
		return errorsx.InvalidArgument("待翻译源文不能为空")
	}
	if len([]byte(source.Text)) > translationDraftMaxBytes {
		return errorsx.InvalidArgument("待翻译源文不能超过2000字节")
	}
	query := t.translationCase.Query(ctx).BaseTranslation
	row, findErr := t.translationCase.Find(ctx,
		repository.Where(query.TargetType.Eq(int32(targetType))),
		repository.Where(query.TargetID.Eq(targetID)),
		repository.Where(query.Locale.Eq(targetLocale)),
	)
	if findErr == nil && row.Name != "" {
		return errorsx.Conflict("已有非空译文，不允许被机器翻译覆盖")
	}
	if findErr != nil && !errors.Is(findErr, gorm.ErrRecordNotFound) {
		return findErr
	}
	translated, err := backendI18n.TranslateProtected(ctx, t.draftTranslator, source.Text, sourceLocale, targetLocale)
	if err != nil {
		return errorsx.Internal("生成翻译失败").WithCause(err)
	}
	if errors.Is(findErr, gorm.ErrRecordNotFound) {
		return t.translationCase.Create(ctx, &models.BaseTranslation{TargetType: int32(targetType), TargetID: targetID, Locale: targetLocale, Name: translated})
	}
	row.Name = translated
	return t.translationCase.UpdateByID(ctx, row)
}

// translateIDs 批量调用统一翻译入口并统计结果。
func (t *BaseTranslationTask) translateIDs(ctx context.Context, targetType systemadminv1.TranslationTargetType, targetIDs []int64, locales []string, resourceName string, translatedCount, failedCount *int, firstErr *error) {
	var err error
	for _, targetID := range targetIDs {
		for _, localeValue := range locales {
			err = t.translateOne(ctx, targetType, targetID, "", localeValue)
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

// translationSource 读取允许外发的资源源文。
func (t *BaseTranslationTask) translationSource(ctx context.Context, targetType systemadminv1.TranslationTargetType, targetID int64) (*dto.TranslationDraftSource, error) {
	source := &dto.TranslationDraftSource{TargetType: targetType, TargetID: targetID}
	var err error
	switch targetType {
	case systemadminv1.TranslationTargetType_TRANSLATION_TARGET_TYPE_BASE_MENU:
		menu, findErr := t.menuRepo.FindByID(ctx, targetID)
		err = findErr
		if err == nil {
			var metadata dto.MenuMetadata
			err = json.Unmarshal([]byte(menu.Meta), &metadata)
			if err == nil {
				source.Text = metadata.Title
			}
		}
	case systemadminv1.TranslationTargetType_TRANSLATION_TARGET_TYPE_BASE_DICT:
		dict, findErr := t.dictRepo.FindByID(ctx, targetID)
		err = findErr
		if err == nil {
			source.Text = dict.Name
		}
	case systemadminv1.TranslationTargetType_TRANSLATION_TARGET_TYPE_BASE_DICT_ITEM:
		item, findErr := t.dictItemRepo.FindByID(ctx, targetID)
		err = findErr
		if err == nil {
			source.Text = item.Label
		}
	case systemadminv1.TranslationTargetType_TRANSLATION_TARGET_TYPE_BASE_CONFIG_NAME,
		systemadminv1.TranslationTargetType_TRANSLATION_TARGET_TYPE_BASE_CONFIG_VALUE:
		config, findErr := t.configRepo.FindByID(ctx, targetID)
		err = findErr
		if err == nil {
			if targetType == systemadminv1.TranslationTargetType_TRANSLATION_TARGET_TYPE_BASE_CONFIG_VALUE && !isTranslatableConfigType(config.Type) {
				return nil, errorsx.InvalidArgument("图片、字典和布尔配置值不支持翻译")
			}
			if targetType == systemadminv1.TranslationTargetType_TRANSLATION_TARGET_TYPE_BASE_CONFIG_NAME {
				source.Text = config.Name
			} else {
				source.Text = config.Value
			}
		}
	default:
		return nil, errorsx.InvalidArgument("翻译目标类型无效")
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errorsx.ResourceNotFound("待翻译资源不存在").WithCause(err)
	}
	if err != nil {
		return nil, errorsx.Internal("读取待翻译资源失败").WithCause(err)
	}
	return source, nil
}

// containsLocale 判断语言代码是否位于启用语言列表中。
func containsLocale(locales []string, value string) bool {
	for _, locale := range locales {
		if locale == value {
			return true
		}
	}
	return false
}

// isTranslatableConfigType 判断配置值是否支持机器翻译。
func isTranslatableConfigType(configType int32) bool {
	return configType == int32(systemadminv1.BaseConfigType_BASE_CONFIG_TYPE_TEXT) || configType == int32(systemadminv1.BaseConfigType_BASE_CONFIG_TYPE_RICH_TEXT)
}

// editableLocales 从启用语言中排除主语言，返回可编辑的目标语言。
func editableLocales(locales []string, primaryLocale string) []string {
	result := make([]string, 0, len(locales))
	for _, locale := range locales {
		if locale != primaryLocale {
			result = append(result, locale)
		}
	}
	return result
}
