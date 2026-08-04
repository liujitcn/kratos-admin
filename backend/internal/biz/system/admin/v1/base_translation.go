package biz

import (
	"context"
	"errors"
	"sync"

	"github.com/liujitcn/go-utils/translator"
	"github.com/liujitcn/gorm-kit/repository"
	systemadminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/core/pkg/errorsx"
	coreLocale "github.com/liujitcn/kratos-admin/backend/core/pkg/locale"
	"github.com/liujitcn/kratos-admin/backend/internal/biz"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin/v1/dto"
	_const "github.com/liujitcn/kratos-admin/backend/internal/const"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	translationQueue "github.com/liujitcn/kratos-admin/backend/internal/data/queue"
	backendI18n "github.com/liujitcn/kratos-admin/backend/internal/i18n"

	"github.com/go-kratos/kratos/v3/log"
	"gorm.io/gorm"
)

const translationDraftMaxBytes = 2000

// BaseTranslationCase 统一管理所有资源的翻译能力。
type BaseTranslationCase struct {
	*biz.BaseCase
	*data.BaseTranslationRepository
	languageCase     *BaseLanguageCase
	draftTranslator  translator.Translator
	baseMenuRepo     *data.BaseMenuRepository
	baseDictRepo     *data.BaseDictRepository
	baseDictItemRepo *data.BaseDictItemRepository
	baseConfigRepo   *data.BaseConfigRepository
	draftMu          sync.Mutex
}

// NewBaseTranslationCase 创建动态翻译业务实例。
func NewBaseTranslationCase(
	baseCase *biz.BaseCase,
	baseTranslationRepository *data.BaseTranslationRepository,
	languageCase *BaseLanguageCase,
	draftTranslator translator.Translator,
	baseMenuRepo *data.BaseMenuRepository,
	baseDictRepo *data.BaseDictRepository,
	baseDictItemRepo *data.BaseDictItemRepository,
	baseConfigRepo *data.BaseConfigRepository,
) *BaseTranslationCase {
	translationCase := &BaseTranslationCase{
		BaseCase:                  baseCase,
		BaseTranslationRepository: baseTranslationRepository,
		languageCase:              languageCase,
		draftTranslator:           draftTranslator,
		baseMenuRepo:              baseMenuRepo,
		baseDictRepo:              baseDictRepo,
		baseDictItemRepo:          baseDictItemRepo,
		baseConfigRepo:            baseConfigRepo,
	}
	return translationCase
}

// DraftBaseTranslation 翻译请求中的单个文本，不保存翻译结果。
func (c *BaseTranslationCase) DraftBaseTranslation(ctx context.Context, req *systemadminv1.DraftBaseTranslationRequest) (*systemadminv1.DraftBaseTranslationResponse, error) {
	if c.draftTranslator == nil {
		return nil, errorsx.PermissionDenied("机器翻译草稿功能未启用")
	}
	if req.GetSource() == "" {
		return nil, errorsx.InvalidArgument("待翻译源文不能为空")
	}
	if len([]byte(req.GetSource())) > translationDraftMaxBytes {
		return nil, errorsx.InvalidArgument("待翻译源文不能超过2000字节")
	}
	locales, primaryLocale, _, err := c.languageCase.Locales(ctx)
	if err != nil {
		return nil, err
	}
	if primaryLocale == "" {
		primaryLocale = coreLocale.Default
	}
	sourceLocale := req.GetSourceLocale()
	if sourceLocale == "" {
		sourceLocale = primaryLocale
	}
	if !isTranslationLocale(locales, primaryLocale, sourceLocale) {
		return nil, errorsx.InvalidArgument("源语言必须是主语言或已启用语言")
	}

	c.draftMu.Lock()
	defer c.draftMu.Unlock()
	translations := make([]*systemadminv1.DraftBaseTranslationItem, 0, len(locales))
	for _, locale := range locales {
		if locale == primaryLocale || locale == sourceLocale {
			continue
		}
		translated, translateErr := backendI18n.TranslateProtected(ctx, c.draftTranslator, req.GetSource(), sourceLocale, locale)
		if translateErr != nil {
			return nil, errorsx.Internal("生成翻译草稿失败").WithCause(translateErr)
		}
		translations = append(translations, &systemadminv1.DraftBaseTranslationItem{Locale: locale, Translation: translated})
	}
	return &systemadminv1.DraftBaseTranslationResponse{Translations: translations}, nil
}

// UpdateBaseTranslation 修改或新增单个翻译信息；文本为空时清理已有译文。
func (c *BaseTranslationCase) UpdateBaseTranslation(ctx context.Context, req *systemadminv1.UpdateBaseTranslationRequest) error {
	if len([]byte(req.GetName())) > translationDraftMaxBytes {
		return errorsx.InvalidArgument("翻译文本不能超过2000字节")
	}
	query := c.Query(ctx).BaseTranslation
	var err error
	var row *models.BaseTranslation
	targetType := req.GetTargetType()
	targetID := req.GetTargetId()
	locale := req.GetLocale()
	if req.GetId() > 0 {
		row, err = c.Find(ctx, repository.Where(query.ID.Eq(req.GetId())))
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errorsx.ResourceNotFound("翻译记录不存在").WithCause(err)
		}
		if err != nil {
			return err
		}
		targetType = systemadminv1.TranslationTargetType(row.TargetType)
		targetID = row.TargetID
		locale = row.Locale
		if req.GetTargetType() != systemadminv1.TranslationTargetType_TRANSLATION_TARGET_TYPE_UNSPECIFIED && req.GetTargetType() != targetType {
			return errorsx.InvalidArgument("翻译目标类型与记录不匹配")
		}
		if req.GetTargetId() > 0 && req.GetTargetId() != targetID {
			return errorsx.InvalidArgument("翻译目标资源与记录不匹配")
		}
		if req.GetLocale() != "" && req.GetLocale() != locale {
			return errorsx.InvalidArgument("翻译语言与记录不匹配")
		}
	}
	if !isTranslationTargetType(targetType) {
		return errorsx.InvalidArgument("翻译目标类型无效")
	}
	if targetID <= 0 {
		return errorsx.InvalidArgument("目标资源ID不能为空")
	}
	locales, primaryLocale, _, err := c.languageCase.Locales(ctx)
	if err != nil {
		return err
	}
	if !containsEditableLocale(locales, primaryLocale, locale) {
		return errorsx.InvalidArgument("翻译语言必须是已启用的非主语言")
	}
	if err = c.validateTranslationTarget(ctx, targetType, targetID); err != nil {
		return err
	}
	if row == nil {
		row, err = c.Find(ctx,
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
	if row == nil {
		if req.GetName() == "" {
			return errorsx.InvalidArgument("新增翻译文本不能为空")
		}
		return c.Create(ctx, &models.BaseTranslation{TargetType: int32(targetType), TargetID: targetID, Locale: locale, Name: req.GetName()})
	}
	row.Name = req.GetName()
	return c.UpdateByID(ctx, row)
}

// GetTargetIdsByName 根据当前语言和名称关键字获取资源 ID。
func (c *BaseTranslationCase) GetTargetIdsByName(ctx context.Context, targetType systemadminv1.TranslationTargetType, name string) ([]int64, error) {
	if name == "" {
		return nil, nil
	}

	localeValue := coreLocale.FromContext(ctx)
	locales, _, currentLocaleIsPrimary, err := c.languageCase.Locales(ctx)
	if err != nil {
		return nil, err
	}
	if currentLocaleIsPrimary || (len(locales) == 0 && localeValue == coreLocale.Default) {
		return nil, nil
	}

	query := c.Query(ctx).BaseTranslation
	var rows []*models.BaseTranslation
	rows, err = c.List(ctx,
		repository.Where(query.TargetType.Eq(int32(targetType))),
		repository.Where(query.Locale.Eq(localeValue)),
		repository.Where(query.Name.Like("%"+name+"%")),
	)
	if err != nil {
		return nil, err
	}
	result := make([]int64, 0, len(rows))
	for _, row := range rows {
		result = append(result, row.TargetID)
	}
	return result, nil
}

// GetBaseTranslationMapByTargetType 根据类型查询翻译信息。
func (c *BaseTranslationCase) GetBaseTranslationMapByTargetType(ctx context.Context, targetType systemadminv1.TranslationTargetType, targetIds []int64) (map[int64][]*systemadminv1.BaseTranslation, error) {
	result := make(map[int64][]*systemadminv1.BaseTranslation, len(targetIds))
	if len(targetIds) == 0 {
		return result, nil
	}
	query := c.Query(ctx).BaseTranslation
	opts := make([]repository.QueryOption, 0, 2)
	opts = append(opts, repository.Where(query.TargetType.Eq(int32(targetType))))
	opts = append(opts, repository.Where(query.TargetID.In(targetIds...)))

	list, err := c.List(ctx, opts...)
	if err != nil {
		return nil, err
	}
	for _, item := range list {
		result[item.TargetID] = append(result[item.TargetID], &systemadminv1.BaseTranslation{
			Id:         item.ID,
			TargetType: systemadminv1.TranslationTargetType(item.TargetType),
			TargetId:   item.TargetID,
			Locale:     item.Locale,
			Name:       item.Name,
		})
	}
	return result, nil
}

// GetBaseTranslationNameMapByLocale 根据语言返回替换信息。
func (c *BaseTranslationCase) GetBaseTranslationNameMapByLocale(ctx context.Context, targetType systemadminv1.TranslationTargetType, locale string, targetIds []int64) (map[int64]string, error) {
	result := make(map[int64]string, len(targetIds))
	if len(targetIds) == 0 {
		return result, nil
	}
	query := c.Query(ctx).BaseTranslation
	opts := make([]repository.QueryOption, 0, 3)
	opts = append(opts, repository.Where(query.TargetType.Eq(int32(targetType))))
	opts = append(opts, repository.Where(query.Locale.Eq(locale)))
	opts = append(opts, repository.Where(query.TargetID.In(targetIds...)))

	list, err := c.List(ctx, opts...)
	if err != nil {
		return nil, err
	}
	for _, item := range list {
		if item.Name != "" {
			result[item.TargetID] = item.Name
		}
	}
	return result, nil
}

// SaveBaseTranslation 保存翻译信息，并在没有有效译文时投递机器翻译任务。
func (c *BaseTranslationCase) SaveBaseTranslation(ctx context.Context, targetType systemadminv1.TranslationTargetType, targetId int64, currentText string, translations []*systemadminv1.BaseTranslation, updateMain func(context.Context, string) error) error {
	var err error
	var locales []string
	var primaryLocale string
	locales, primaryLocale, _, err = c.languageCase.Locales(ctx)
	if err != nil {
		return err
	}
	currentLocale := coreLocale.FromContext(ctx)
	currentLocaleFound := false
	isPrimary := len(locales) == 0 && currentLocale == coreLocale.Default
	for _, locale := range locales {
		if locale == currentLocale {
			currentLocaleFound = true
			isPrimary = locale == primaryLocale
		}
	}
	editableLocales := make(map[string]struct{}, len(locales))
	for _, locale := range locales {
		if locale != primaryLocale {
			editableLocales[locale] = struct{}{}
		}
	}
	if !currentLocaleFound && (len(locales) > 0 || currentLocale != coreLocale.Default) {
		return errorsx.InvalidArgument("当前语言必须是已启用的非主语言")
	}

	save := func(txCtx context.Context) error {
		query := c.Query(txCtx).BaseTranslation
		opts := make([]repository.QueryOption, 0, 2)
		opts = append(opts, repository.Where(query.TargetType.Eq(int32(targetType))))
		opts = append(opts, repository.Where(query.TargetID.Eq(targetId)))
		var list []*models.BaseTranslation
		list, err = c.List(txCtx, opts...)
		if err != nil {
			return err
		}
		existing := make(map[string]*models.BaseTranslation, len(list))
		for _, item := range list {
			existing[item.Locale] = item
		}

		values := make(map[string]string, len(translations)+1)
		seen := make(map[string]struct{}, len(translations))
		for _, translation := range translations {
			if translation.GetTargetType() != targetType {
				return errorsx.InvalidArgument("翻译目标类型无效")
			}
			localeValue := translation.GetLocale()
			if _, ok := editableLocales[localeValue]; !ok {
				return errorsx.InvalidArgument("翻译语言必须是已启用的非主语言")
			}
			if !isPrimary && localeValue == currentLocale {
				values[localeValue] = currentText
				continue
			}
			if _, duplicated := seen[localeValue]; duplicated {
				return errorsx.Conflict("同一资源语言不能重复")
			}
			if len([]byte(translation.GetName())) > translationDraftMaxBytes {
				return errorsx.InvalidArgument("翻译文本不能超过2000字节")
			}
			seen[localeValue] = struct{}{}
			values[localeValue] = translation.GetName()
		}
		if !isPrimary {
			values[currentLocale] = currentText
		}
		for localeValue, text := range values {
			row := existing[localeValue]
			if text == "" {
				if row != nil && row.Name != "" {
					row.Name = ""
					if err = c.UpdateByID(txCtx, row); err != nil {
						return err
					}
				}
				continue
			}
			if row == nil {
				if err = c.Create(txCtx, &models.BaseTranslation{TargetType: int32(targetType), TargetID: targetId, Locale: localeValue, Name: text}); err != nil {
					return err
				}
				continue
			}
			if row.Name == text {
				continue
			}
			row.Name = text
			if err = c.UpdateByID(txCtx, row); err != nil {
				return err
			}
		}
		if !isPrimary && updateMain != nil {
			if err = updateMain(txCtx, currentText); err != nil {
				return err
			}
		}
		return nil
	}
	if updateMain == nil {
		err = save(ctx)
	} else {
		err = c.BaseTranslationRepository.Transaction(ctx, save)
	}
	if err != nil || c.draftTranslator == nil || targetId <= 0 {
		return err
	}
	hasUsefulTranslation := false
	for _, translation := range translations {
		if translation.GetName() != "" {
			hasUsefulTranslation = true
			break
		}
	}
	if hasUsefulTranslation {
		return nil
	}
	for _, locale := range locales {
		if locale == currentLocale || locale == primaryLocale {
			continue
		}
		if ok := translationQueue.AddQueue(_const.TRANSLATION, &dto.TranslationQueueMessage{
			TargetType:   targetType,
			TargetID:     targetId,
			SourceLocale: currentLocale,
			TargetLocale: locale,
		}); !ok {
			log.Warn("投递机器翻译队列失败", "target_type", targetType.String(), "target_id", targetId, "source_locale", currentLocale, "target_locale", locale)
		}
	}
	return nil
}

// validateTranslationTarget 校验翻译目标类型、资源存在性和可翻译配置值类型。
func (c *BaseTranslationCase) validateTranslationTarget(ctx context.Context, targetType systemadminv1.TranslationTargetType, targetID int64) error {
	var err error
	switch targetType {
	case systemadminv1.TranslationTargetType_TRANSLATION_TARGET_TYPE_BASE_MENU:
		_, err = c.baseMenuRepo.FindByID(ctx, targetID)
	case systemadminv1.TranslationTargetType_TRANSLATION_TARGET_TYPE_BASE_DICT:
		_, err = c.baseDictRepo.FindByID(ctx, targetID)
	case systemadminv1.TranslationTargetType_TRANSLATION_TARGET_TYPE_BASE_DICT_ITEM:
		_, err = c.baseDictItemRepo.FindByID(ctx, targetID)
	case systemadminv1.TranslationTargetType_TRANSLATION_TARGET_TYPE_BASE_CONFIG_NAME,
		systemadminv1.TranslationTargetType_TRANSLATION_TARGET_TYPE_BASE_CONFIG_VALUE:
		config, findErr := c.baseConfigRepo.FindByID(ctx, targetID)
		err = findErr
		if err == nil && targetType == systemadminv1.TranslationTargetType_TRANSLATION_TARGET_TYPE_BASE_CONFIG_VALUE && !isTranslatableConfigType(config.Type) {
			return errorsx.InvalidArgument("图片、字典和布尔配置值不支持翻译")
		}
	default:
		return errorsx.InvalidArgument("翻译目标类型无效")
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return errorsx.ResourceNotFound("翻译目标资源不存在").WithCause(err)
	}
	if err != nil {
		return errorsx.Internal("查询翻译目标资源失败").WithCause(err)
	}
	return nil
}

// isTranslationTargetType 判断统一翻译表目标类型是否有效。
func isTranslationTargetType(targetType systemadminv1.TranslationTargetType) bool {
	switch targetType {
	case systemadminv1.TranslationTargetType_TRANSLATION_TARGET_TYPE_BASE_CONFIG_VALUE,
		systemadminv1.TranslationTargetType_TRANSLATION_TARGET_TYPE_BASE_CONFIG_NAME,
		systemadminv1.TranslationTargetType_TRANSLATION_TARGET_TYPE_BASE_DICT,
		systemadminv1.TranslationTargetType_TRANSLATION_TARGET_TYPE_BASE_DICT_ITEM,
		systemadminv1.TranslationTargetType_TRANSLATION_TARGET_TYPE_BASE_MENU:
		return true
	default:
		return false
	}
}

// containsEditableLocale 判断语言是否为启用的非主语言。
func containsEditableLocale(locales []string, primaryLocale, locale string) bool {
	if locale == "" || locale == primaryLocale {
		return false
	}
	for _, item := range locales {
		if item == locale {
			return true
		}
	}
	return false
}

// isTranslationLocale 判断语言是否为主语言或启用语言。
func isTranslationLocale(locales []string, primaryLocale, locale string) bool {
	return locale == primaryLocale || containsEditableLocale(locales, primaryLocale, locale)
}

func (c *BaseTranslationCase) DeleteBaseTranslation(ctx context.Context, targetType systemadminv1.TranslationTargetType, targetId []int64) error {
	if len(targetId) == 0 {
		return nil
	}
	query := c.Query(ctx).BaseTranslation
	opts := make([]repository.QueryOption, 0, 2)
	opts = append(opts, repository.Where(query.TargetType.Eq(int32(targetType))))
	opts = append(opts, repository.Where(query.TargetID.In(targetId...)))
	return c.Delete(ctx, opts...)
}
