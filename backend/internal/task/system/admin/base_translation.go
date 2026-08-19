package admin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin/dto"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"

	kratosErrors "github.com/go-kratos/kratos/v3/errors"
	"github.com/go-kratos/kratos/v3/log"
	"github.com/liujitcn/gorm-kit/repository"
	"github.com/liujitcn/kratos-core/errorsx"
	"github.com/liujitcn/kratos-kit/transport/cron"
	cronTransport "github.com/liujitcn/kratos-kit/transport/cron"
	"gorm.io/gorm"
)

const (
	// BaseTranslationTaskName 是统一机器翻译任务的稳定调用目标。
	BaseTranslationTaskName = "system.admin.BaseTranslation"
)

var _ cron.TaskExec = (*BaseTranslationTask)(nil)

// BaseTranslationTask 执行菜单、字典、字典项和系统配置的机器翻译任务。
type BaseTranslationTask struct {
	translationCase *biz.BaseTranslationCase
	menuRepo        *data.BaseMenuRepository
	dictRepo        *data.BaseDictRepository
	dictItemRepo    *data.BaseDictItemRepository
	configRepo      *data.BaseConfigRepository
	mu              sync.Mutex
}

type translationIndex map[adminv1.TranslationTargetType]map[dto.TranslationKey]*models.BaseI18N

func (i translationIndex) get(targetType adminv1.TranslationTargetType, targetID int64, locale string) *models.BaseI18N {
	return i[targetType][dto.TranslationKey{TargetID: targetID, Locale: locale}]
}

func (i translationIndex) set(row *models.BaseI18N) {
	targetType := adminv1.TranslationTargetType(row.TargetType)
	rows := i[targetType]
	if rows == nil {
		rows = make(map[dto.TranslationKey]*models.BaseI18N)
		i[targetType] = rows
	}
	rows[dto.TranslationKey{TargetID: row.TargetID, Locale: row.Locale}] = row
}

// NewBaseTranslationTask 创建统一机器翻译任务执行器。
func NewBaseTranslationTask(
	translationCase *biz.BaseTranslationCase,
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
	return task
}

// Task 返回交由 base_job 统一调度的任务定义。
func (t *BaseTranslationTask) Task() cronTransport.Task {
	return cronTransport.Task{Name: BaseTranslationTaskName, Exec: t}
}

// Exec 扫描所有资源并补齐缺失的机器译文。
func (t *BaseTranslationTask) Exec(ctx context.Context, _ map[string]string) ([]string, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.translationCase.Translator == nil {
		return []string{"机器翻译功能未启用"}, nil
	}
	state, err := t.translationCase.LocaleState(ctx)
	if err != nil {
		return nil, err
	}
	locales := state.EditableLocales()
	if len(locales) == 0 {
		return []string{"没有启用的目标语言"}, nil
	}
	var translations translationIndex
	translations, err = t.loadTranslationIndex(ctx)
	if err != nil {
		return nil, err
	}

	translatedCount := 0
	failedCount := 0
	var firstErr error
	menuQuery := t.menuRepo.Query(ctx).BaseMenu
	var menus []*models.BaseMenu
	menus, err = t.menuRepo.List(ctx, repository.Order(menuQuery.ID.Asc()))
	if err != nil {
		return nil, err
	}
	menuIDs := make([]int64, 0, len(menus))
	for _, menu := range menus {
		menuIDs = append(menuIDs, menu.ID)
	}
	t.translateIDs(ctx, state, translations, adminv1.TranslationTargetType_TRANSLATION_TARGET_TYPE_BASE_MENU, menuIDs, locales, "菜单", &translatedCount, &failedCount, &firstErr)

	dictQuery := t.dictRepo.Query(ctx).BaseDict
	var dicts []*models.BaseDict
	dicts, err = t.dictRepo.List(ctx, repository.Order(dictQuery.ID.Asc()))
	if err != nil {
		return nil, err
	}
	dictIDs := make([]int64, 0, len(dicts))
	for _, dict := range dicts {
		dictIDs = append(dictIDs, dict.ID)
	}
	t.translateIDs(ctx, state, translations, adminv1.TranslationTargetType_TRANSLATION_TARGET_TYPE_BASE_DICT, dictIDs, locales, "字典", &translatedCount, &failedCount, &firstErr)

	dictItemQuery := t.dictItemRepo.Query(ctx).BaseDictItem
	var dictItems []*models.BaseDictItem
	dictItems, err = t.dictItemRepo.List(ctx, repository.Order(dictItemQuery.ID.Asc()))
	if err != nil {
		return nil, err
	}
	dictItemIDs := make([]int64, 0, len(dictItems))
	for _, item := range dictItems {
		dictItemIDs = append(dictItemIDs, item.ID)
	}
	t.translateIDs(ctx, state, translations, adminv1.TranslationTargetType_TRANSLATION_TARGET_TYPE_BASE_DICT_ITEM, dictItemIDs, locales, "字典项", &translatedCount, &failedCount, &firstErr)

	configQuery := t.configRepo.Query(ctx).BaseConfig
	var configs []*models.BaseConfig
	configs, err = t.configRepo.List(ctx, repository.Order(configQuery.ID.Asc()))
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
	t.translateIDs(ctx, state, translations, adminv1.TranslationTargetType_TRANSLATION_TARGET_TYPE_BASE_CONFIG_NAME, configNameIDs, locales, "系统配置名称", &translatedCount, &failedCount, &firstErr)
	t.translateIDs(ctx, state, translations, adminv1.TranslationTargetType_TRANSLATION_TARGET_TYPE_BASE_CONFIG_VALUE, configValueIDs, locales, "系统配置值", &translatedCount, &failedCount, &firstErr)

	output := []string{fmt.Sprintf("生成机器译文 %d 条", translatedCount)}
	if failedCount > 0 {
		return output, fmt.Errorf("机器翻译失败 %d 条: %w", failedCount, firstErr)
	}
	return output, nil
}

func (t *BaseTranslationTask) loadTranslationIndex(ctx context.Context) (translationIndex, error) {
	query := t.translationCase.Query(ctx).BaseI18N
	rows, err := t.translationCase.List(ctx, repository.Order(query.ID.Asc()))
	if err != nil {
		return nil, err
	}
	index := make(translationIndex)
	for _, row := range rows {
		index.set(row)
	}
	return index, nil
}

// translateOneWithState 使用已读取的语言状态生成单个资源译文。
func (t *BaseTranslationTask) translateOneWithState(ctx context.Context, state *dto.LocaleState, translations translationIndex, targetType adminv1.TranslationTargetType, targetID int64, sourceLocale string, targetLocale string, sourceText string) error {
	if t.translationCase.Translator == nil {
		return errorsx.PermissionDenied("机器翻译功能未启用")
	}
	if targetID <= 0 {
		return errorsx.InvalidArgument("目标资源ID不能为空")
	}
	if sourceLocale == "" {
		sourceLocale = state.Primary
	}
	if !state.IsEnabled(sourceLocale) || !state.IsEditable(targetLocale) || sourceLocale == targetLocale {
		return errorsx.InvalidArgument("源语言和目标语言必须是不同的已启用语言")
	}
	var err error
	var row *models.BaseI18N
	if translations == nil {
		query := t.translationCase.Query(ctx).BaseI18N
		row, err = t.translationCase.Find(ctx,
			repository.Where(query.TargetType.Eq(int32(targetType))),
			repository.Where(query.TargetID.Eq(targetID)),
			repository.Where(query.Locale.Eq(targetLocale)),
		)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
	} else {
		row = translations.get(targetType, targetID, targetLocale)
	}
	if row != nil && row.Name != "" {
		return errorsx.Conflict("已有非空译文，不允许被机器翻译覆盖")
	}
	if sourceText == "" {
		var source *dto.TranslationDraftSource
		source, err = t.translationSource(ctx, targetType, targetID)
		if err != nil {
			return err
		}
		sourceText = source.Text
	}
	if sourceText == "" {
		return errorsx.InvalidArgument("待翻译源文不能为空")
	}
	var translated string
	translated, err = t.translationCase.TranslateText(ctx, sourceText, sourceLocale, targetLocale)
	if err != nil {
		return errorsx.Internal("生成翻译失败").WithCause(err)
	}
	if row == nil {
		row = &models.BaseI18N{TargetType: int32(targetType), TargetID: targetID, Locale: targetLocale, Name: translated}
		if err = t.translationCase.Create(ctx, row); err != nil {
			return err
		}
		if translations != nil {
			translations.set(row)
		}
		return nil
	}
	row.Name = translated
	return t.translationCase.UpdateByID(ctx, row)
}

// translateIDs 批量调用统一翻译入口并统计结果。
func (t *BaseTranslationTask) translateIDs(ctx context.Context, state *dto.LocaleState, translations translationIndex, targetType adminv1.TranslationTargetType, targetIDs []int64, locales []string, resourceName string, translatedCount, failedCount *int, firstErr *error) {
	var err error
	for _, targetID := range targetIDs {
		for _, localeValue := range locales {
			err = t.translateOneWithState(ctx, state, translations, targetType, targetID, state.Primary, localeValue, "")
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

// translationSource 读取允许外发的资源源文。
func (t *BaseTranslationTask) translationSource(ctx context.Context, targetType adminv1.TranslationTargetType, targetID int64) (*dto.TranslationDraftSource, error) {
	source := &dto.TranslationDraftSource{TargetType: targetType, TargetID: targetID}
	var err error
	switch targetType {
	case adminv1.TranslationTargetType_TRANSLATION_TARGET_TYPE_BASE_MENU:
		menu, findErr := t.menuRepo.FindByID(ctx, targetID)
		err = findErr
		if err == nil {
			var metadata dto.MenuMetadata
			err = json.Unmarshal([]byte(menu.Meta), &metadata)
			if err == nil {
				source.Text = metadata.Title
			}
		}
	case adminv1.TranslationTargetType_TRANSLATION_TARGET_TYPE_BASE_DICT:
		dict, findErr := t.dictRepo.FindByID(ctx, targetID)
		err = findErr
		if err == nil {
			source.Text = dict.Name
		}
	case adminv1.TranslationTargetType_TRANSLATION_TARGET_TYPE_BASE_DICT_ITEM:
		item, findErr := t.dictItemRepo.FindByID(ctx, targetID)
		err = findErr
		if err == nil {
			source.Text = item.Label
		}
	case adminv1.TranslationTargetType_TRANSLATION_TARGET_TYPE_BASE_CONFIG_NAME,
		adminv1.TranslationTargetType_TRANSLATION_TARGET_TYPE_BASE_CONFIG_VALUE:
		config, findErr := t.configRepo.FindByID(ctx, targetID)
		err = findErr
		if err == nil {
			if targetType == adminv1.TranslationTargetType_TRANSLATION_TARGET_TYPE_BASE_CONFIG_VALUE && !isTranslatableConfigType(config.Type) {
				return nil, errorsx.InvalidArgument("图片、字典和布尔配置值不支持翻译")
			}
			if targetType == adminv1.TranslationTargetType_TRANSLATION_TARGET_TYPE_BASE_CONFIG_NAME {
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

// isTranslatableConfigType 判断配置值是否支持机器翻译。
func isTranslatableConfigType(configType int32) bool {
	return configType == int32(adminv1.BaseConfigType_BASE_CONFIG_TYPE_TEXT) || configType == int32(adminv1.BaseConfigType_BASE_CONFIG_TYPE_RICH_TEXT)
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
