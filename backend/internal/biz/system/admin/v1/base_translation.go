package biz

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	systemadminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	commonv1 "github.com/liujitcn/kratos-admin/backend/core/api/gen/go/common/v1"
	"github.com/liujitcn/kratos-admin/backend/core/pkg/errorsx"
	coreLocale "github.com/liujitcn/kratos-admin/backend/core/pkg/locale"
	"github.com/liujitcn/kratos-admin/backend/internal/biz"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin/v1/dto"
	systemConfig "github.com/liujitcn/kratos-admin/backend/internal/config"
	_const "github.com/liujitcn/kratos-admin/backend/internal/const"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	translationQueue "github.com/liujitcn/kratos-admin/backend/internal/data/queue"
	backendI18n "github.com/liujitcn/kratos-admin/backend/internal/i18n"

	"github.com/go-kratos/kratos/v3/log"
	"github.com/liujitcn/go-utils/translator"
	"github.com/liujitcn/gorm-kit/repository"
	"gorm.io/gorm"
)

const translationDraftMaxBytes = 2000

// BaseTranslationCase 统一管理所有资源的翻译能力。
type BaseTranslationCase struct {
	*biz.BaseCase
	draftConfig     systemConfig.TranslationDraftConfig
	draftTranslator translator.Translator
	menuRepo        *data.BaseMenuRepository
	dictRepo        *data.BaseDictRepository
	dictItemRepo    *data.BaseDictItemRepository
	configRepo      *data.BaseConfigRepository
	translationRepo *data.BaseTranslationRepository
	languageRepo    *data.BaseLanguageRepository
	draftMu         sync.Mutex
}

// NewBaseTranslationCase 创建动态翻译业务实例。
func NewBaseTranslationCase(
	baseCase *biz.BaseCase,
	draftConfig systemConfig.TranslationDraftConfig,
	draftTranslator translator.Translator,
	menuRepo *data.BaseMenuRepository,
	dictRepo *data.BaseDictRepository,
	dictItemRepo *data.BaseDictItemRepository,
	configRepo *data.BaseConfigRepository,
	languageRepo *data.BaseLanguageRepository,
	translationRepo *data.BaseTranslationRepository,
) *BaseTranslationCase {
	translationCase := &BaseTranslationCase{
		BaseCase:        baseCase,
		draftConfig:     draftConfig,
		draftTranslator: draftTranslator,
		menuRepo:        menuRepo,
		dictRepo:        dictRepo,
		dictItemRepo:    dictItemRepo,
		configRepo:      configRepo,
		translationRepo: translationRepo,
		languageRepo:    languageRepo,
	}
	return translationCase
}

// DraftBaseTranslation 翻译请求中的单个文本，不保存翻译结果。
func (c *BaseTranslationCase) DraftBaseTranslation(ctx context.Context, req *systemadminv1.DraftBaseTranslationRequest) (*systemadminv1.DraftBaseTranslationResponse, error) {
	sourceLocale := req.GetSourceLocale()
	targetLocale := req.GetTargetLocale()
	var err error
	if sourceLocale == "" {
		sourceLocale, err = c.PrimaryLocale(ctx)
		if err != nil {
			return nil, err
		}
	}
	var text string
	text, err = c.TranslateText(ctx, req.GetSource(), sourceLocale, targetLocale)
	if err != nil {
		return nil, err
	}
	return &systemadminv1.DraftBaseTranslationResponse{
		SourceLocale: sourceLocale,
		TargetLocale: targetLocale,
		Translation:  text,
	}, nil
}

// UpdateBaseTranslation 修改或新增单个翻译信息；文本为空时补充机器译文。
func (c *BaseTranslationCase) UpdateBaseTranslation(ctx context.Context, req *systemadminv1.UpdateBaseTranslationRequest) error {
	id := req.GetId()
	text := req.GetName()
	if len([]byte(text)) > translationDraftMaxBytes {
		return errorsx.InvalidArgument("翻译文本不能超过2000字节")
	}
	query := c.translationRepo.Query(ctx).BaseTranslation
	var err error
	var row *models.BaseTranslation
	var targetType systemadminv1.TranslationTargetType
	var targetID int64
	var locale string
	if id > 0 {
		var rows []*models.BaseTranslation
		rows, err = c.translationRepo.List(ctx, repository.Where(query.ID.Eq(id)))
		if err != nil {
			return err
		}
		if len(rows) == 0 {
			return errorsx.ResourceNotFound("翻译记录不存在")
		}
		row = rows[0]
	} else {
		targetType = req.GetTargetType()
		targetID = req.GetTargetId()
		locale = req.GetLocale()
		if !isTranslationTargetType(targetType) {
			return errorsx.InvalidArgument("翻译目标类型无效")
		}
		if targetID <= 0 {
			return errorsx.InvalidArgument("目标资源ID不能为空")
		}
		var locales []string
		locales, err = c.enabledEditableLocales(ctx)
		if err != nil {
			return err
		}
		if locale, _ = editableLocale(locales, locale); locale == "" {
			return errorsx.InvalidArgument("翻译语言必须是已启用的非主语言")
		}
		row, err = c.translationRepo.Find(ctx,
			repository.Where(query.TargetType.Eq(int32(targetType))),
			repository.Where(query.TargetID.Eq(targetID)),
			repository.Where(query.Locale.Eq(locale)),
		)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			row = nil
			err = nil
		}
		if err != nil {
			return err
		}
	}
	if row != nil {
		targetType = systemadminv1.TranslationTargetType(row.TargetType)
		targetID = row.TargetID
		locale = row.Locale
		if !isTranslationTargetType(targetType) {
			return errorsx.InvalidArgument("翻译目标类型无效")
		}
	}
	if text == "" {
		var source *dto.TranslationDraftSource
		source, err = c.findTranslationDraftSource(ctx, targetType, targetID)
		if err != nil {
			return err
		}
		text, err = c.TranslateText(ctx, source.Text, "", locale)
		if err != nil {
			return err
		}
	}
	if row == nil {
		row = &models.BaseTranslation{TargetType: int32(targetType), TargetID: targetID, Locale: locale, Name: text}
		if err = c.translationRepo.Create(ctx, row); err != nil {
			return err
		}
		return nil
	}
	row.Name = text
	if err = c.translationRepo.UpdateByID(ctx, row); err != nil {
		return err
	}
	return nil
}

// DraftEnabled 返回当前部署是否启用机器翻译草稿能力。
func (c *BaseTranslationCase) DraftEnabled() bool {
	return c.draftConfig.Enabled
}

// CanTranslateConfigValue 判断系统配置值是否属于允许机器翻译的文本类型。
func (c *BaseTranslationCase) CanTranslateConfigValue(configType int32) bool {
	return isTranslatableConfigType(configType)
}

// EnqueueTranslation 将已保存资源投递到异步机器翻译队列。
func (c *BaseTranslationCase) EnqueueTranslation(targetType systemadminv1.TranslationTargetType, targetID int64) {
	if !c.DraftEnabled() || targetID <= 0 {
		return
	}
	ok := translationQueue.AddQueue(_const.TRANSLATION, &dto.TranslationQueueMessage{
		TargetType: targetType,
		TargetID:   targetID,
	})
	if !ok {
		log.Warn("投递机器翻译队列失败", "target_type", targetType.String(), "target_id", targetID)
	}
}

// PrimaryLocale 查询当前启用的主语言代码。
func (c *BaseTranslationCase) PrimaryLocale(ctx context.Context) (string, error) {
	query := c.languageRepo.Query(ctx).BaseLanguage
	opts := []repository.QueryOption{
		repository.Where(query.Status.Eq(int32(commonv1.Status_ENABLE))),
		repository.Where(query.IsPrimary.Is(true)),
		repository.Order(query.Sort.Asc()),
		repository.Order(query.ID.Asc()),
	}
	rows, err := c.languageRepo.List(ctx, opts...)
	if err != nil {
		return "", err
	}
	if len(rows) > 0 {
		return rows[0].LanguageCode, nil
	}

	// 存量环境未配置主语言时沿用第一种启用语言，保证升级期间仍可读写。
	rows, err = c.languageRepo.List(ctx,
		repository.Where(query.Status.Eq(int32(commonv1.Status_ENABLE))),
		repository.Order(query.Sort.Asc()),
		repository.Order(query.ID.Asc()),
	)
	if err != nil {
		return "", err
	}
	if len(rows) > 0 {
		return rows[0].LanguageCode, nil
	}
	return coreLocale.Default, nil
}

// IsPrimaryLocale 判断指定语言是否为当前主语言。
func (c *BaseTranslationCase) IsPrimaryLocale(ctx context.Context, localeValue string) (bool, error) {
	primaryLocale, err := c.PrimaryLocale(ctx)
	if err != nil {
		return false, err
	}
	return localeValue == primaryLocale, nil
}

// EditableLocales 查询当前启用且非主语言的语言代码，供其他业务触发翻译草稿使用。
func (c *BaseTranslationCase) EditableLocales(ctx context.Context) ([]string, error) {
	return c.enabledEditableLocales(ctx)
}

// TranslateResource 读取单个资源源文，翻译并保存机器译文。
func (c *BaseTranslationCase) TranslateResource(ctx context.Context, targetType systemadminv1.TranslationTargetType, targetID int64, targetLocale string) error {
	if !c.DraftEnabled() {
		return errorsx.PermissionDenied("机器翻译功能未启用")
	}
	if targetID <= 0 {
		return errorsx.InvalidArgument("目标ID不能为空")
	}
	locales, err := c.enabledEditableLocales(ctx)
	if err != nil {
		return err
	}
	targetLocale, ok := editableLocale(locales, targetLocale)
	if !ok {
		return errorsx.InvalidArgument("目标语言必须是已启用的非主语言")
	}
	var source *dto.TranslationDraftSource
	source, err = c.findTranslationDraftSource(ctx, targetType, targetID)
	if err != nil {
		return err
	}
	if source.Text == "" {
		return errorsx.InvalidArgument("待翻译源文不能为空")
	}
	if len([]byte(source.Text)) > translationDraftMaxBytes {
		return errorsx.InvalidArgument("待翻译源文不能超过2000字节")
	}
	var existing bool
	existing, err = c.hasExistingTranslation(ctx, source, targetLocale)
	if err != nil {
		return err
	}
	if existing {
		return errorsx.Conflict("已有非空译文，不允许被机器翻译覆盖")
	}

	var primaryLocale string
	primaryLocale, err = c.PrimaryLocale(ctx)
	if err != nil {
		return err
	}
	var translated string
	translated, err = c.translateText(ctx, source.Text, primaryLocale, targetLocale)
	if err != nil {
		log.Error("生成翻译失败", "target_type", source.TargetType.String(), "target_id", source.TargetID, "locale", targetLocale, "error", err)
		return errorsx.Internal("生成翻译失败").WithCause(err)
	}

	err = c.saveMachineDraft(ctx, source, targetLocale, translated)
	if err != nil {
		return err
	}
	log.Info("生成翻译成功", "target_type", source.TargetType.String(), "target_id", source.TargetID, "locale", targetLocale)
	return nil
}

// TranslateText 将指定源语言文本翻译为目标语言并返回结果。
func (c *BaseTranslationCase) TranslateText(ctx context.Context, source, sourceLocale, targetLocale string) (string, error) {
	if !c.DraftEnabled() {
		return "", errorsx.PermissionDenied("机器翻译草稿功能未启用")
	}
	locales, err := c.enabledEditableLocales(ctx)
	if err != nil {
		return "", err
	}
	var primaryLocale string
	primaryLocale, err = c.PrimaryLocale(ctx)
	if err != nil {
		return "", err
	}
	if sourceLocale == "" {
		sourceLocale = primaryLocale
	}
	normalizedSourceLocale, ok := translationLocale(locales, primaryLocale, sourceLocale)
	if !ok {
		return "", errorsx.InvalidArgument("源语言必须是主语言或已启用语言")
	}
	var normalizedTargetLocale string
	normalizedTargetLocale, ok = translationLocale(locales, primaryLocale, targetLocale)
	if !ok {
		return "", errorsx.InvalidArgument("目标语言必须是主语言或已启用语言")
	}
	if source == "" {
		return "", errorsx.InvalidArgument("待翻译源文不能为空")
	}
	if len([]byte(source)) > translationDraftMaxBytes {
		return "", errorsx.InvalidArgument("待翻译源文不能超过2000字节")
	}
	var translated string
	translated, err = c.translateText(ctx, source, normalizedSourceLocale, normalizedTargetLocale)
	if err != nil {
		return "", errorsx.Internal("生成翻译草稿失败").WithCause(err)
	}
	return translated, nil
}

// NormalizePrimaryText 将当前请求语言的输入文本转换为主语言文本。
func (c *BaseTranslationCase) NormalizePrimaryText(ctx context.Context, source string) (string, string, string, error) {
	sourceLocale := coreLocale.FromContext(ctx)
	primaryLocale, err := c.PrimaryLocale(ctx)
	if err != nil {
		return "", "", "", err
	}
	if sourceLocale == primaryLocale {
		return source, sourceLocale, primaryLocale, nil
	}
	var translated string
	translated, err = c.translateText(ctx, source, sourceLocale, primaryLocale)
	if err != nil {
		return "", "", "", errorsx.Internal("输入内容翻译为主语言失败").WithCause(err)
	}
	return translated, sourceLocale, primaryLocale, nil
}

// TranslatedConfigIDsByName 查询当前语言名称包含关键字的系统配置编号。
func (c *BaseTranslationCase) TranslatedConfigIDsByName(ctx context.Context, keyword string) ([]int64, error) {
	if keyword == "" {
		return nil, nil
	}
	localeValue := coreLocale.FromContext(ctx)
	isPrimary, err := c.IsPrimaryLocale(ctx, localeValue)
	if err != nil {
		return nil, err
	}
	if isPrimary {
		return nil, nil
	}
	query := c.translationRepo.Query(ctx).BaseTranslation
	var rows []*models.BaseTranslation
	rows, err = c.translationRepo.List(ctx,
		repository.Where(query.TargetType.Eq(int32(_const.TRANSLATION_TARGET_TYPE_BASE_CONFIG_NAME))),
		repository.Where(query.Locale.Eq(localeValue)),
		repository.Where(query.Name.Like("%"+keyword+"%")),
	)
	if err != nil {
		return nil, err
	}
	ids := make([]int64, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.TargetID)
	}
	return ids, nil
}

// ConfigTranslations 批量查询系统配置维护界面的翻译内容。
func (c *BaseTranslationCase) ConfigTranslations(ctx context.Context, sources map[int64]dto.ConfigTranslationSource) (map[int64][]*systemadminv1.BaseTranslation, error) {
	result := make(map[int64][]*systemadminv1.BaseTranslation, len(sources))
	if len(sources) == 0 {
		return result, nil
	}
	locales, err := c.enabledEditableLocales(ctx)
	if err != nil {
		return nil, err
	}
	ids := make([]int64, 0, len(sources))
	for configID := range sources {
		ids = append(ids, configID)
	}
	query := c.translationRepo.Query(ctx).BaseTranslation
	var rows []*models.BaseTranslation
	rows, err = c.translationRepo.List(ctx,
		repository.Where(query.TargetType.In(_const.TRANSLATION_TARGET_TYPE_BASE_CONFIG_NAME, _const.TRANSLATION_TARGET_TYPE_BASE_CONFIG_VALUE)),
		repository.Where(query.TargetID.In(ids...)),
	)
	if err != nil {
		return nil, err
	}
	rowMap := make(map[string]*models.BaseTranslation, len(rows))
	for _, row := range rows {
		rowMap[configTranslationKey(row.TargetID, row.Locale, row.TargetType)] = row
	}
	for configID, source := range sources {
		fields := []int32{int32(_const.TRANSLATION_TARGET_TYPE_BASE_CONFIG_NAME)}
		if isTranslatableConfigType(source.Type) {
			fields = append(fields, int32(_const.TRANSLATION_TARGET_TYPE_BASE_CONFIG_VALUE))
		}
		translations := make([]*systemadminv1.BaseTranslation, 0, len(locales)*len(fields))
		for _, targetType := range fields {
			for _, localeValue := range locales {
				translations = append(translations, translationDTO(rowMap[configTranslationKey(configID, localeValue, targetType)], systemadminv1.TranslationTargetType(targetType), configID, localeValue))
			}
		}
		result[configID] = translations
	}
	return result, nil
}

// SaveConfigTranslations 保存人工维护的系统配置译文。
func (c *BaseTranslationCase) SaveConfigTranslations(ctx context.Context, configID int64, source dto.ConfigTranslationSource, translations []*systemadminv1.BaseTranslation) error {
	if translations == nil {
		return nil
	}
	locales, err := c.enabledEditableLocales(ctx)
	if err != nil {
		return err
	}
	query := c.translationRepo.Query(ctx).BaseTranslation
	var rows []*models.BaseTranslation
	rows, err = c.translationRepo.List(ctx,
		repository.Where(query.TargetType.In(int32(_const.TRANSLATION_TARGET_TYPE_BASE_CONFIG_NAME), int32(_const.TRANSLATION_TARGET_TYPE_BASE_CONFIG_VALUE))),
		repository.Where(query.TargetID.Eq(configID)),
	)
	if err != nil {
		return err
	}
	existing := make(map[string]*models.BaseTranslation, len(rows))
	for _, row := range rows {
		existing[configTranslationKey(row.TargetID, row.Locale, row.TargetType)] = row
	}
	seen := make(map[string]struct{}, len(translations))
	for _, translation := range translations {
		localeValue, ok := editableLocale(locales, translation.GetLocale())
		if !ok {
			return errorsx.InvalidArgument("系统配置翻译语言必须是已启用的非主语言")
		}
		targetType := int32(translation.GetTargetType())
		if targetType != int32(_const.TRANSLATION_TARGET_TYPE_BASE_CONFIG_NAME) && (targetType != int32(_const.TRANSLATION_TARGET_TYPE_BASE_CONFIG_VALUE) || !isTranslatableConfigType(source.Type)) {
			return errorsx.InvalidArgument("当前系统配置类型不支持翻译")
		}
		key := configTranslationKey(configID, localeValue, targetType)
		if _, duplicated := seen[key]; duplicated {
			return errorsx.Conflict("同一系统配置语言和字段不能重复")
		}
		seen[key] = struct{}{}
		row := existing[key]
		if translation.GetName() == "" {
			if row != nil && row.Name != "" {
				row.Name = ""
				if err = c.translationRepo.UpdateByID(ctx, row); err != nil {
					return err
				}
			}
			continue
		}
		if row == nil {
			if err = c.translationRepo.Create(ctx, &models.BaseTranslation{TargetType: targetType, TargetID: configID, Locale: localeValue, Name: translation.GetName()}); err != nil {
				return err
			}
			continue
		}
		row.Name = translation.GetName()
		if err = c.translationRepo.UpdateByID(ctx, row); err != nil {
			return err
		}
	}
	return nil
}

// DeleteConfigTranslations 删除系统配置对应的统一翻译记录。
func (c *BaseTranslationCase) DeleteConfigTranslations(ctx context.Context, configIDs []int64) error {
	if len(configIDs) == 0 {
		return nil
	}
	query := c.translationRepo.Query(ctx).BaseTranslation
	return c.translationRepo.Delete(ctx,
		repository.Where(query.TargetType.In(int32(_const.TRANSLATION_TARGET_TYPE_BASE_CONFIG_NAME), int32(_const.TRANSLATION_TARGET_TYPE_BASE_CONFIG_VALUE))),
		repository.Where(query.TargetID.In(configIDs...)),
	)
}

// TranslatedDictNames 批量查询当前语言的非空字典名称翻译。
func (c *BaseTranslationCase) TranslatedDictNames(ctx context.Context, dictIDs []int64) (map[int64]string, error) {
	result := make(map[int64]string)
	localeValue := coreLocale.FromContext(ctx)
	if len(dictIDs) == 0 {
		return result, nil
	}
	isPrimary, err := c.IsPrimaryLocale(ctx, localeValue)
	if err != nil {
		return nil, err
	}
	if isPrimary {
		return result, nil
	}
	query := c.translationRepo.Query(ctx).BaseTranslation
	var rows []*models.BaseTranslation
	rows, err = c.translationRepo.List(ctx,
		repository.Where(query.TargetType.Eq(int32(_const.TRANSLATION_TARGET_TYPE_BASE_DICT))),
		repository.Where(query.TargetID.In(dictIDs...)),
		repository.Where(query.Locale.Eq(localeValue)),
	)
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		if row.Name != "" {
			result[row.TargetID] = row.Name
		}
	}
	return result, nil
}

// TranslatedDictIDsByName 查询当前语言名称包含关键字的字典编号。
func (c *BaseTranslationCase) TranslatedDictIDsByName(ctx context.Context, keyword string) ([]int64, error) {
	if keyword == "" {
		return nil, nil
	}
	localeValue := coreLocale.FromContext(ctx)
	isPrimary, err := c.IsPrimaryLocale(ctx, localeValue)
	if err != nil {
		return nil, err
	}
	if isPrimary {
		return nil, nil
	}
	query := c.translationRepo.Query(ctx).BaseTranslation
	var rows []*models.BaseTranslation
	rows, err = c.translationRepo.List(ctx,
		repository.Where(query.TargetType.Eq(int32(_const.TRANSLATION_TARGET_TYPE_BASE_DICT))),
		repository.Where(query.Locale.Eq(localeValue)),
		repository.Where(query.Name.Like("%"+keyword+"%")),
	)
	if err != nil {
		return nil, err
	}
	ids := make([]int64, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.TargetID)
	}
	return ids, nil
}

// DictTranslations 批量查询字典维护界面的翻译内容。
func (c *BaseTranslationCase) DictTranslations(ctx context.Context, sources map[int64]string) (map[int64][]*systemadminv1.BaseTranslation, error) {
	result := make(map[int64][]*systemadminv1.BaseTranslation, len(sources))
	if len(sources) == 0 {
		return result, nil
	}
	locales, err := c.enabledEditableLocales(ctx)
	if err != nil {
		return nil, err
	}
	ids := translationSourceIDs(sources)
	query := c.translationRepo.Query(ctx).BaseTranslation
	var rows []*models.BaseTranslation
	rows, err = c.translationRepo.List(ctx,
		repository.Where(query.TargetType.Eq(int32(_const.TRANSLATION_TARGET_TYPE_BASE_DICT))),
		repository.Where(query.TargetID.In(ids...)),
	)
	if err != nil {
		return nil, err
	}
	rowMap := make(map[dto.TranslationKey]*models.BaseTranslation, len(rows))
	for _, row := range rows {
		rowMap[dto.TranslationKey{TargetID: row.TargetID, Locale: row.Locale}] = row
	}
	for resourceID := range sources {
		translations := make([]*systemadminv1.BaseTranslation, 0, len(locales))
		for _, localeValue := range locales {
			translations = append(translations, translationDTO(rowMap[dto.TranslationKey{TargetID: resourceID, Locale: localeValue}], systemadminv1.TranslationTargetType_TRANSLATION_TARGET_TYPE_BASE_DICT, resourceID, localeValue))
		}
		result[resourceID] = translations
	}
	return result, nil
}

// SaveDictTranslations 保存人工维护的字典译文。
func (c *BaseTranslationCase) SaveDictTranslations(ctx context.Context, dictID int64, _ string, translations []*systemadminv1.BaseTranslation) error {
	if translations == nil {
		return nil
	}
	locales, err := c.enabledEditableLocales(ctx)
	if err != nil {
		return err
	}
	query := c.translationRepo.Query(ctx).BaseTranslation
	var rows []*models.BaseTranslation
	rows, err = c.translationRepo.List(ctx,
		repository.Where(query.TargetType.Eq(int32(_const.TRANSLATION_TARGET_TYPE_BASE_DICT))),
		repository.Where(query.TargetID.Eq(dictID)),
	)
	if err != nil {
		return err
	}
	existing := make(map[string]*models.BaseTranslation, len(rows))
	for _, row := range rows {
		existing[row.Locale] = row
	}
	seen := make(map[string]struct{}, len(translations))
	for _, translation := range translations {
		if translation.GetTargetType() != systemadminv1.TranslationTargetType_TRANSLATION_TARGET_TYPE_BASE_DICT {
			return errorsx.InvalidArgument("字典翻译目标类型无效")
		}
		localeValue, ok := editableLocale(locales, translation.GetLocale())
		if !ok {
			return errorsx.InvalidArgument("字典翻译语言必须是已启用的非主语言")
		}
		if _, duplicated := seen[localeValue]; duplicated {
			return errorsx.Conflict("同一字典语言不能重复")
		}
		seen[localeValue] = struct{}{}
		row := existing[localeValue]
		if translation.GetName() == "" {
			if row != nil && row.Name != "" {
				row.Name = ""
				if err = c.translationRepo.UpdateByID(ctx, row); err != nil {
					return err
				}
			}
			continue
		}
		if row == nil {
			if err = c.translationRepo.Create(ctx, &models.BaseTranslation{TargetType: int32(_const.TRANSLATION_TARGET_TYPE_BASE_DICT), TargetID: dictID, Locale: localeValue, Name: translation.GetName()}); err != nil {
				return err
			}
			continue
		}
		row.Name = translation.GetName()
		if err = c.translationRepo.UpdateByID(ctx, row); err != nil {
			return err
		}
	}
	return nil
}

// DeleteDictTranslations 删除字典对应的统一翻译记录。
func (c *BaseTranslationCase) DeleteDictTranslations(ctx context.Context, dictIDs []int64) error {
	if len(dictIDs) == 0 {
		return nil
	}
	query := c.translationRepo.Query(ctx).BaseTranslation
	return c.translationRepo.Delete(ctx,
		repository.Where(query.TargetType.Eq(int32(_const.TRANSLATION_TARGET_TYPE_BASE_DICT))),
		repository.Where(query.TargetID.In(dictIDs...)),
	)
}

// TranslatedDictItemLabels 批量查询当前语言的非空字典项标签翻译。
func (c *BaseTranslationCase) TranslatedDictItemLabels(ctx context.Context, dictItemIDs []int64) (map[int64]string, error) {
	result := make(map[int64]string)
	localeValue := coreLocale.FromContext(ctx)
	if len(dictItemIDs) == 0 {
		return result, nil
	}
	isPrimary, err := c.IsPrimaryLocale(ctx, localeValue)
	if err != nil {
		return nil, err
	}
	if isPrimary {
		return result, nil
	}
	query := c.translationRepo.Query(ctx).BaseTranslation
	var rows []*models.BaseTranslation
	rows, err = c.translationRepo.List(ctx,
		repository.Where(query.TargetType.Eq(int32(_const.TRANSLATION_TARGET_TYPE_BASE_DICT_ITEM))),
		repository.Where(query.TargetID.In(dictItemIDs...)),
		repository.Where(query.Locale.Eq(localeValue)),
	)
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		if row.Name != "" {
			result[row.TargetID] = row.Name
		}
	}
	return result, nil
}

// TranslatedDictItemIDsByLabel 查询当前语言标签包含关键字的字典项编号。
func (c *BaseTranslationCase) TranslatedDictItemIDsByLabel(ctx context.Context, keyword string) ([]int64, error) {
	if keyword == "" {
		return nil, nil
	}
	localeValue := coreLocale.FromContext(ctx)
	isPrimary, err := c.IsPrimaryLocale(ctx, localeValue)
	if err != nil {
		return nil, err
	}
	if isPrimary {
		return nil, nil
	}
	query := c.translationRepo.Query(ctx).BaseTranslation
	var rows []*models.BaseTranslation
	rows, err = c.translationRepo.List(ctx,
		repository.Where(query.TargetType.Eq(int32(_const.TRANSLATION_TARGET_TYPE_BASE_DICT_ITEM))),
		repository.Where(query.Locale.Eq(localeValue)),
		repository.Where(query.Name.Like("%"+keyword+"%")),
	)
	if err != nil {
		return nil, err
	}
	ids := make([]int64, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.TargetID)
	}
	return ids, nil
}

// DictItemTranslations 批量查询字典项维护界面的翻译内容。
func (c *BaseTranslationCase) DictItemTranslations(ctx context.Context, sources map[int64]string) (map[int64][]*systemadminv1.BaseTranslation, error) {
	result := make(map[int64][]*systemadminv1.BaseTranslation, len(sources))
	if len(sources) == 0 {
		return result, nil
	}
	locales, err := c.enabledEditableLocales(ctx)
	if err != nil {
		return nil, err
	}
	ids := translationSourceIDs(sources)
	query := c.translationRepo.Query(ctx).BaseTranslation
	var rows []*models.BaseTranslation
	rows, err = c.translationRepo.List(ctx,
		repository.Where(query.TargetType.Eq(int32(_const.TRANSLATION_TARGET_TYPE_BASE_DICT_ITEM))),
		repository.Where(query.TargetID.In(ids...)),
	)
	if err != nil {
		return nil, err
	}
	rowMap := make(map[dto.TranslationKey]*models.BaseTranslation, len(rows))
	for _, row := range rows {
		rowMap[dto.TranslationKey{TargetID: row.TargetID, Locale: row.Locale}] = row
	}
	for resourceID := range sources {
		translations := make([]*systemadminv1.BaseTranslation, 0, len(locales))
		for _, localeValue := range locales {
			translations = append(translations, translationDTO(rowMap[dto.TranslationKey{TargetID: resourceID, Locale: localeValue}], systemadminv1.TranslationTargetType_TRANSLATION_TARGET_TYPE_BASE_DICT_ITEM, resourceID, localeValue))
		}
		result[resourceID] = translations
	}
	return result, nil
}

// SaveDictItemTranslations 保存人工维护的字典项译文。
func (c *BaseTranslationCase) SaveDictItemTranslations(ctx context.Context, dictItemID int64, _ string, translations []*systemadminv1.BaseTranslation) error {
	if translations == nil {
		return nil
	}
	locales, err := c.enabledEditableLocales(ctx)
	if err != nil {
		return err
	}
	query := c.translationRepo.Query(ctx).BaseTranslation
	var rows []*models.BaseTranslation
	rows, err = c.translationRepo.List(ctx,
		repository.Where(query.TargetType.Eq(int32(_const.TRANSLATION_TARGET_TYPE_BASE_DICT_ITEM))),
		repository.Where(query.TargetID.Eq(dictItemID)),
	)
	if err != nil {
		return err
	}
	existing := make(map[string]*models.BaseTranslation, len(rows))
	for _, row := range rows {
		existing[row.Locale] = row
	}
	seen := make(map[string]struct{}, len(translations))
	for _, translation := range translations {
		if translation.GetTargetType() != systemadminv1.TranslationTargetType_TRANSLATION_TARGET_TYPE_BASE_DICT_ITEM {
			return errorsx.InvalidArgument("字典项翻译目标类型无效")
		}
		localeValue, ok := editableLocale(locales, translation.GetLocale())
		if !ok {
			return errorsx.InvalidArgument("字典项翻译语言必须是已启用的非主语言")
		}
		if _, duplicated := seen[localeValue]; duplicated {
			return errorsx.Conflict("同一字典项语言不能重复")
		}
		seen[localeValue] = struct{}{}
		row := existing[localeValue]
		if translation.GetName() == "" {
			if row != nil && row.Name != "" {
				row.Name = ""
				if err = c.translationRepo.UpdateByID(ctx, row); err != nil {
					return err
				}
			}
			continue
		}
		if row == nil {
			if err = c.translationRepo.Create(ctx, &models.BaseTranslation{TargetType: int32(_const.TRANSLATION_TARGET_TYPE_BASE_DICT_ITEM), TargetID: dictItemID, Locale: localeValue, Name: translation.GetName()}); err != nil {
				return err
			}
			continue
		}
		row.Name = translation.GetName()
		if err = c.translationRepo.UpdateByID(ctx, row); err != nil {
			return err
		}
	}
	return nil
}

// DeleteDictItemTranslations 删除字典项对应的统一翻译记录。
func (c *BaseTranslationCase) DeleteDictItemTranslations(ctx context.Context, dictItemIDs []int64) error {
	if len(dictItemIDs) == 0 {
		return nil
	}
	query := c.translationRepo.Query(ctx).BaseTranslation
	return c.translationRepo.Delete(ctx,
		repository.Where(query.TargetType.Eq(int32(_const.TRANSLATION_TARGET_TYPE_BASE_DICT_ITEM))),
		repository.Where(query.TargetID.In(dictItemIDs...)),
	)
}

// TranslatedMenuTitles 批量查询当前语言的非空菜单标题翻译。
func (c *BaseTranslationCase) TranslatedMenuTitles(ctx context.Context, menuIDs []int64) (map[int64]string, error) {
	result := make(map[int64]string)
	localeValue := coreLocale.FromContext(ctx)
	if len(menuIDs) == 0 {
		return result, nil
	}
	isPrimary, err := c.IsPrimaryLocale(ctx, localeValue)
	if err != nil {
		return nil, err
	}
	if isPrimary {
		return result, nil
	}
	query := c.translationRepo.Query(ctx).BaseTranslation
	var rows []*models.BaseTranslation
	rows, err = c.translationRepo.List(ctx,
		repository.Where(query.TargetType.Eq(int32(_const.TRANSLATION_TARGET_TYPE_BASE_MENU))),
		repository.Where(query.TargetID.In(menuIDs...)),
		repository.Where(query.Locale.Eq(localeValue)),
	)
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		if row.Name != "" {
			result[row.TargetID] = row.Name
		}
	}
	return result, nil
}

// MenuTranslations 批量查询菜单维护界面的翻译内容。
func (c *BaseTranslationCase) MenuTranslations(ctx context.Context, sources map[int64]string) (map[int64][]*systemadminv1.BaseTranslation, error) {
	result := make(map[int64][]*systemadminv1.BaseTranslation, len(sources))
	if len(sources) == 0 {
		return result, nil
	}
	locales, err := c.enabledEditableLocales(ctx)
	if err != nil {
		return nil, err
	}
	ids := translationSourceIDs(sources)
	query := c.translationRepo.Query(ctx).BaseTranslation
	var rows []*models.BaseTranslation
	rows, err = c.translationRepo.List(ctx,
		repository.Where(query.TargetType.Eq(int32(_const.TRANSLATION_TARGET_TYPE_BASE_MENU))),
		repository.Where(query.TargetID.In(ids...)),
	)
	if err != nil {
		return nil, err
	}
	rowMap := make(map[dto.TranslationKey]*models.BaseTranslation, len(rows))
	for _, row := range rows {
		rowMap[dto.TranslationKey{TargetID: row.TargetID, Locale: row.Locale}] = row
	}
	for resourceID := range sources {
		translations := make([]*systemadminv1.BaseTranslation, 0, len(locales))
		for _, localeValue := range locales {
			translations = append(translations, translationDTO(rowMap[dto.TranslationKey{TargetID: resourceID, Locale: localeValue}], systemadminv1.TranslationTargetType_TRANSLATION_TARGET_TYPE_BASE_MENU, resourceID, localeValue))
		}
		result[resourceID] = translations
	}
	return result, nil
}

// SaveMenuTranslations 保存人工维护的菜单译文。
func (c *BaseTranslationCase) SaveMenuTranslations(ctx context.Context, menuID int64, _ string, translations []*systemadminv1.BaseTranslation) error {
	if translations == nil {
		return nil
	}
	locales, err := c.enabledEditableLocales(ctx)
	if err != nil {
		return err
	}
	query := c.translationRepo.Query(ctx).BaseTranslation
	var rows []*models.BaseTranslation
	rows, err = c.translationRepo.List(ctx,
		repository.Where(query.TargetType.Eq(int32(_const.TRANSLATION_TARGET_TYPE_BASE_MENU))),
		repository.Where(query.TargetID.Eq(menuID)),
	)
	if err != nil {
		return err
	}
	existing := make(map[string]*models.BaseTranslation, len(rows))
	for _, row := range rows {
		existing[row.Locale] = row
	}
	seen := make(map[string]struct{}, len(translations))
	for _, translation := range translations {
		if translation.GetTargetType() != systemadminv1.TranslationTargetType_TRANSLATION_TARGET_TYPE_BASE_MENU {
			return errorsx.InvalidArgument("菜单翻译目标类型无效")
		}
		localeValue, ok := editableLocale(locales, translation.GetLocale())
		if !ok {
			return errorsx.InvalidArgument("菜单翻译语言必须是已启用的非主语言")
		}
		if _, duplicated := seen[localeValue]; duplicated {
			return errorsx.Conflict("同一菜单语言不能重复")
		}
		seen[localeValue] = struct{}{}
		row := existing[localeValue]
		if translation.GetName() == "" {
			if row != nil && row.Name != "" {
				row.Name = ""
				if err = c.translationRepo.UpdateByID(ctx, row); err != nil {
					return err
				}
			}
			continue
		}
		if row == nil {
			row = &models.BaseTranslation{TargetType: int32(_const.TRANSLATION_TARGET_TYPE_BASE_MENU), TargetID: menuID, Locale: localeValue}
			row.Name = translation.GetName()
			if err = c.translationRepo.Create(ctx, row); err != nil {
				return err
			}
			continue
		}
		row.Name = translation.GetName()
		if err = c.translationRepo.UpdateByID(ctx, row); err != nil {
			return err
		}
	}
	return nil
}

// SaveGeneratedMenuTranslations 保存代码生成器提供的译文，不覆盖已有内容。
func (c *BaseTranslationCase) SaveGeneratedMenuTranslations(ctx context.Context, menuID int64, _ string, translations map[string]string) error {
	if len(translations) == 0 {
		return nil
	}
	locales, err := c.enabledEditableLocales(ctx)
	if err != nil {
		return err
	}
	query := c.translationRepo.Query(ctx).BaseTranslation
	var rows []*models.BaseTranslation
	rows, err = c.translationRepo.List(ctx,
		repository.Where(query.TargetType.Eq(int32(_const.TRANSLATION_TARGET_TYPE_BASE_MENU))),
		repository.Where(query.TargetID.Eq(menuID)),
	)
	if err != nil {
		return err
	}
	existing := make(map[string]*models.BaseTranslation, len(rows))
	for _, row := range rows {
		existing[row.Locale] = row
	}
	for _, localeValue := range locales {
		text := translations[localeValue]
		if text == "" || existing[localeValue] != nil && existing[localeValue].Name != "" {
			continue
		}
		if existing[localeValue] == nil {
			if err = c.translationRepo.Create(ctx, &models.BaseTranslation{TargetType: int32(_const.TRANSLATION_TARGET_TYPE_BASE_MENU), TargetID: menuID, Locale: localeValue, Name: text}); err != nil {
				return err
			}
			continue
		}
		existing[localeValue].Name = text
		if err = c.translationRepo.UpdateByID(ctx, existing[localeValue]); err != nil {
			return err
		}
	}
	return nil
}

// DeleteMenuTranslations 删除菜单对应的统一翻译记录。
func (c *BaseTranslationCase) DeleteMenuTranslations(ctx context.Context, menuIDs []int64) error {
	if len(menuIDs) == 0 {
		return nil
	}
	query := c.translationRepo.Query(ctx).BaseTranslation
	return c.translationRepo.Delete(ctx,
		repository.Where(query.TargetType.Eq(int32(_const.TRANSLATION_TARGET_TYPE_BASE_MENU))),
		repository.Where(query.TargetID.In(menuIDs...)),
	)
}

// enabledEditableLocales 查询当前启用且非主语言的可维护语言代码。
func (c *BaseTranslationCase) enabledEditableLocales(ctx context.Context) ([]string, error) {
	query := c.languageRepo.Query(ctx).BaseLanguage
	opts := make([]repository.QueryOption, 0, 4)
	opts = append(opts, repository.Where(query.Status.Eq(int32(commonv1.Status_ENABLE))))
	opts = append(opts, repository.Where(query.IsPrimary.Is(false)))
	opts = append(opts, repository.Order(query.Sort.Asc()), repository.Order(query.ID.Asc()))
	rows, err := c.languageRepo.List(ctx, opts...)
	if err != nil {
		return nil, err
	}
	locales := make([]string, 0, len(rows))
	for _, row := range rows {
		locales = append(locales, row.LanguageCode)
	}
	return locales, nil
}

// translateText 串行调用翻译提供方，避免超出草稿接口的并发边界。
func (c *BaseTranslationCase) translateText(ctx context.Context, source, sourceLocale, targetLocale string) (string, error) {
	c.draftMu.Lock()
	defer c.draftMu.Unlock()
	deadlineCtx, cancel := context.WithTimeout(ctx, c.draftConfig.Timeout)
	defer cancel()
	return backendI18n.TranslateProtected(deadlineCtx, c.draftTranslator, source, sourceLocale, targetLocale)
}

// findTranslationDraftSource 按资源类型从服务端读取允许外发的展示源文。
func (c *BaseTranslationCase) findTranslationDraftSource(ctx context.Context, targetType systemadminv1.TranslationTargetType, targetID int64) (*dto.TranslationDraftSource, error) {
	switch targetType {
	case systemadminv1.TranslationTargetType_TRANSLATION_TARGET_TYPE_BASE_MENU:
		return c.findMenuTranslationDraftSource(ctx, targetType, targetID)
	case systemadminv1.TranslationTargetType_TRANSLATION_TARGET_TYPE_BASE_DICT:
		return c.findDictTranslationDraftSource(ctx, targetType, targetID)
	case systemadminv1.TranslationTargetType_TRANSLATION_TARGET_TYPE_BASE_DICT_ITEM:
		return c.findDictItemTranslationDraftSource(ctx, targetType, targetID)
	case systemadminv1.TranslationTargetType_TRANSLATION_TARGET_TYPE_BASE_CONFIG_NAME,
		systemadminv1.TranslationTargetType_TRANSLATION_TARGET_TYPE_BASE_CONFIG_VALUE:
		return c.findConfigTranslationDraftSource(ctx, targetType, targetID)
	default:
		return nil, errorsx.InvalidArgument("翻译目标类型无效")
	}
}

// hasExistingTranslation 判断目标资源是否已有不可覆盖的非空译文。
func (c *BaseTranslationCase) hasExistingTranslation(ctx context.Context, source *dto.TranslationDraftSource, localeValue string) (bool, error) {
	if !isTranslationTargetType(source.TargetType) {
		return false, errorsx.InvalidArgument("翻译目标类型无效")
	}
	query := c.translationRepo.Query(ctx).BaseTranslation
	row, err := c.translationRepo.Find(ctx,
		repository.Where(query.TargetType.Eq(int32(source.TargetType))),
		repository.Where(query.TargetID.Eq(source.TargetID)),
		repository.Where(query.Locale.Eq(localeValue)),
	)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return row.Name != "", nil
}

// saveMachineDraft 新增或更新可覆盖的机器翻译草稿。
func (c *BaseTranslationCase) saveMachineDraft(ctx context.Context, source *dto.TranslationDraftSource, localeValue, translated string) error {
	if translated == "" {
		return nil
	}
	if !isTranslationTargetType(source.TargetType) {
		return errorsx.InvalidArgument("翻译目标类型无效")
	}
	query := c.translationRepo.Query(ctx).BaseTranslation
	row, err := c.translationRepo.Find(ctx,
		repository.Where(query.TargetType.Eq(int32(source.TargetType))),
		repository.Where(query.TargetID.Eq(source.TargetID)),
		repository.Where(query.Locale.Eq(localeValue)),
	)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return c.translationRepo.Create(ctx, &models.BaseTranslation{TargetType: int32(source.TargetType), TargetID: source.TargetID, Locale: localeValue, Name: translated})
	}
	if err != nil {
		return err
	}
	if row.Name != "" {
		return nil
	}
	row.Name = translated
	return c.translationRepo.UpdateByID(ctx, row)
}

// findTranslationDraftSource 按系统配置编号读取指定字段的源文。
func (c *BaseTranslationCase) findConfigTranslationDraftSource(ctx context.Context, targetType systemadminv1.TranslationTargetType, resourceID int64) (*dto.TranslationDraftSource, error) {
	config, err := c.configRepo.FindByID(ctx, resourceID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errorsx.ResourceNotFound("待翻译资源不存在").WithCause(err)
		}
		return nil, errorsx.Internal("读取待翻译资源失败").WithCause(err)
	}
	var text string
	text, err = configTranslationSourceText(config, targetType)
	if err != nil {
		return nil, err
	}
	return &dto.TranslationDraftSource{TargetType: targetType, TargetID: resourceID, Text: text}, nil
}

// findTranslationDraftSource 按字典编号读取字典名称源文。
func (c *BaseTranslationCase) findDictTranslationDraftSource(ctx context.Context, targetType systemadminv1.TranslationTargetType, resourceID int64) (*dto.TranslationDraftSource, error) {
	dict, err := c.dictRepo.FindByID(ctx, resourceID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errorsx.ResourceNotFound("待翻译资源不存在").WithCause(err)
		}
		return nil, errorsx.Internal("读取待翻译资源失败").WithCause(err)
	}
	return &dto.TranslationDraftSource{TargetType: targetType, TargetID: resourceID, Text: dict.Name}, nil
}

// findTranslationDraftSource 按字典项编号读取标签源文。
func (c *BaseTranslationCase) findDictItemTranslationDraftSource(ctx context.Context, targetType systemadminv1.TranslationTargetType, resourceID int64) (*dto.TranslationDraftSource, error) {
	item, err := c.dictItemRepo.FindByID(ctx, resourceID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errorsx.ResourceNotFound("待翻译资源不存在").WithCause(err)
		}
		return nil, errorsx.Internal("读取待翻译资源失败").WithCause(err)
	}
	return &dto.TranslationDraftSource{TargetType: targetType, TargetID: resourceID, Text: item.Label}, nil
}

// findTranslationDraftSource 按菜单编号读取菜单标题源文。
func (c *BaseTranslationCase) findMenuTranslationDraftSource(ctx context.Context, targetType systemadminv1.TranslationTargetType, resourceID int64) (*dto.TranslationDraftSource, error) {
	menu, err := c.menuRepo.FindByID(ctx, resourceID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errorsx.ResourceNotFound("待翻译资源不存在").WithCause(err)
		}
		return nil, errorsx.Internal("读取待翻译资源失败").WithCause(err)
	}
	metadata := new(dto.MenuMetadata)
	if err = json.Unmarshal([]byte(menu.Meta), metadata); err != nil {
		return nil, errorsx.Internal("读取待翻译资源失败").WithCause(err)
	}
	return &dto.TranslationDraftSource{
		TargetType: targetType,
		TargetID:   resourceID,
		Text:       metadata.Title,
	}, nil
}

// isTranslationTargetType 判断统一翻译目标类型是否受支持。
func isTranslationTargetType(targetType systemadminv1.TranslationTargetType) bool {
	switch targetType {
	case systemadminv1.TranslationTargetType_TRANSLATION_TARGET_TYPE_BASE_DICT,
		systemadminv1.TranslationTargetType_TRANSLATION_TARGET_TYPE_BASE_MENU,
		systemadminv1.TranslationTargetType_TRANSLATION_TARGET_TYPE_BASE_DICT_ITEM,
		systemadminv1.TranslationTargetType_TRANSLATION_TARGET_TYPE_BASE_CONFIG_VALUE,
		systemadminv1.TranslationTargetType_TRANSLATION_TARGET_TYPE_BASE_CONFIG_NAME:
		return true
	default:
		return false
	}
}

// editableLocale 在启用语言列表中查找可维护的非主语言。
func editableLocale(locales []string, value string) (string, bool) {
	for _, localeValue := range locales {
		if localeValue == value {
			return localeValue, true
		}
	}
	return "", false
}

// translationLocale 校验主语言或启用的非主语言。
func translationLocale(locales []string, primaryLocale, value string) (string, bool) {
	if value == primaryLocale {
		return value, true
	}
	return editableLocale(locales, value)
}

// translationSourceIDs 提取批量翻译源文的资源ID。
func translationSourceIDs(sources map[int64]string) []int64 {
	ids := make([]int64, 0, len(sources))
	for resourceID := range sources {
		ids = append(ids, resourceID)
	}
	return ids
}

// translationDTO 转换统一翻译表记录，记录不存在时返回待保存的空翻译。
func translationDTO(row *models.BaseTranslation, targetType systemadminv1.TranslationTargetType, resourceID int64, localeValue string) *systemadminv1.BaseTranslation {
	if row == nil {
		return &systemadminv1.BaseTranslation{
			TargetType: targetType,
			TargetId:   resourceID,
			Locale:     localeValue,
		}
	}
	return &systemadminv1.BaseTranslation{Id: row.ID, TargetType: systemadminv1.TranslationTargetType(row.TargetType), TargetId: row.TargetID, Locale: row.Locale, Name: row.Name}
}

// configTranslationSourceText 返回配置指定翻译字段的主语言源文。
func configTranslationSourceText(config *models.BaseConfig, targetType systemadminv1.TranslationTargetType) (string, error) {
	switch targetType {
	case systemadminv1.TranslationTargetType_TRANSLATION_TARGET_TYPE_BASE_CONFIG_NAME:
		return config.Name, nil
	case systemadminv1.TranslationTargetType_TRANSLATION_TARGET_TYPE_BASE_CONFIG_VALUE:
		if !isTranslatableConfigType(config.Type) {
			return "", errorsx.InvalidArgument("图片、字典和布尔配置值不支持翻译")
		}
		return config.Value, nil
	default:
		return "", errorsx.InvalidArgument("系统配置翻译字段无效")
	}
}

// isTranslatableConfigType 判断配置值是否属于允许翻译的文本类型。
func isTranslatableConfigType(configType int32) bool {
	return configType == int32(systemadminv1.BaseConfigType_BASE_CONFIG_TYPE_TEXT) || configType == int32(systemadminv1.BaseConfigType_BASE_CONFIG_TYPE_RICH_TEXT)
}

// configTranslationKey 生成配置翻译记录的内存索引键。
func configTranslationKey(configID int64, localeValue string, targetType int32) string {
	return fmt.Sprintf("%d:%s:%d", configID, localeValue, targetType)
}
