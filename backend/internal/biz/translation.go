package biz

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	systemadminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	systemcommonv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/common/v1"
	"github.com/liujitcn/kratos-admin/backend/core/pkg/errorsx"
	coreLocale "github.com/liujitcn/kratos-admin/backend/core/pkg/locale"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/dto"
	systemConfig "github.com/liujitcn/kratos-admin/backend/internal/config"
	_const "github.com/liujitcn/kratos-admin/backend/internal/const"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	backendI18n "github.com/liujitcn/kratos-admin/backend/internal/i18n"

	"github.com/go-kratos/kratos/v3/log"
	"github.com/liujitcn/go-utils/translator"
	"github.com/liujitcn/gorm-kit/repository"
	"gorm.io/gorm"
)

const translationDraftMaxBytes = 2000

var editableLocales = []string{coreLocale.EnUS, coreLocale.JaJP}

// TranslationCase 统一管理动态菜单、字典和字典项翻译。
type TranslationCase struct {
	*BaseCase
	draftConfig             systemConfig.TranslationDraftConfig
	draftTranslator         translator.Translator
	menuRepo                *data.BaseMenuRepository
	dictRepo                *data.BaseDictRepository
	dictItemRepo            *data.BaseDictItemRepository
	configRepo              *data.BaseConfigRepository
	menuTranslationRepo     *data.BaseMenuTranslationRepository
	dictTranslationRepo     *data.BaseDictTranslationRepository
	dictItemTranslationRepo *data.BaseDictItemTranslationRepository
	configTranslationRepo   *data.BaseConfigTranslationRepository
	draftMu                 sync.Mutex
}

// NewTranslationCase 创建动态翻译业务实例。
func NewTranslationCase(
	baseCase *BaseCase,
	draftConfig systemConfig.TranslationDraftConfig,
	draftTranslator translator.Translator,
	menuRepo *data.BaseMenuRepository,
	dictRepo *data.BaseDictRepository,
	dictItemRepo *data.BaseDictItemRepository,
	configRepo *data.BaseConfigRepository,
	menuTranslationRepo *data.BaseMenuTranslationRepository,
	dictTranslationRepo *data.BaseDictTranslationRepository,
	dictItemTranslationRepo *data.BaseDictItemTranslationRepository,
	configTranslationRepo *data.BaseConfigTranslationRepository,
) *TranslationCase {
	return &TranslationCase{
		BaseCase:                baseCase,
		draftConfig:             draftConfig,
		draftTranslator:         draftTranslator,
		menuRepo:                menuRepo,
		dictRepo:                dictRepo,
		dictItemRepo:            dictItemRepo,
		configRepo:              configRepo,
		menuTranslationRepo:     menuTranslationRepo,
		dictTranslationRepo:     dictTranslationRepo,
		dictItemTranslationRepo: dictItemTranslationRepo,
		configTranslationRepo:   configTranslationRepo,
	}
}

// DraftEnabled 返回当前部署是否启用机器翻译草稿能力。
func (c *TranslationCase) DraftEnabled() bool {
	return c.draftConfig.Enabled
}

// GenerateTranslationDraft 为单个已保存资源生成并保存机器翻译草稿。
func (c *TranslationCase) GenerateTranslationDraft(ctx context.Context, req *systemadminv1.GenerateTranslationDraftRequest) (*systemadminv1.GenerateTranslationDraftResponse, error) {
	if !c.DraftEnabled() {
		return nil, errorsx.PermissionDenied("机器翻译草稿功能未启用")
	}
	targetLocale, ok := editableLocale(req.GetTargetLocale())
	if !ok {
		return nil, errorsx.InvalidArgument("目标语言仅支持英语或日语")
	}
	source, err := c.findTranslationDraftSource(ctx, req.GetResourceType(), req.GetResourceId(), req.GetField())
	if err != nil {
		return nil, err
	}
	if source.Text == "" {
		return nil, errorsx.InvalidArgument("待翻译源文不能为空")
	}
	if len([]byte(source.Text)) > translationDraftMaxBytes {
		return nil, errorsx.InvalidArgument("待翻译源文不能超过2000字节")
	}
	var reviewed bool
	reviewed, err = c.hasReviewedTranslation(ctx, source, targetLocale)
	if err != nil {
		return nil, err
	}
	if reviewed {
		return nil, errorsx.Conflict("人工审核译文不允许被机器草稿覆盖")
	}

	startedAt := time.Now()
	c.draftMu.Lock()
	deadlineCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
	var translated string
	translated, err = backendI18n.TranslateProtected(deadlineCtx, c.draftTranslator, source.Text, coreLocale.ZhCN, targetLocale)
	cancel()
	c.draftMu.Unlock()
	if err != nil {
		log.Error("生成翻译草稿失败", "resource_type", source.ResourceType.String(), "resource_id", source.ResourceID, "locale", targetLocale, "duration", time.Since(startedAt), "error", err)
		return nil, errorsx.Internal("生成翻译草稿失败").WithCause(err)
	}

	now := time.Now()
	sourceHash := translationSourceHash(source.Text)
	err = c.saveMachineDraft(ctx, source, targetLocale, translated, sourceHash, now)
	if err != nil {
		return nil, err
	}
	log.Info("生成翻译草稿成功", "resource_type", source.ResourceType.String(), "resource_id", source.ResourceID, "locale", targetLocale, "duration", time.Since(startedAt))
	return &systemadminv1.GenerateTranslationDraftResponse{
		ResourceType:        source.ResourceType,
		ResourceId:          source.ResourceID,
		Locale:              targetLocale,
		Translation:         translated,
		TranslationStatus:   systemadminv1.TranslationStatus_TRANSLATION_STATUS_MACHINE,
		SourceHash:          sourceHash,
		TranslationProvider: _const.TRANSLATION_PROVIDER_GOOGLE_V1,
		TranslatedAt:        formatTranslationTime(now),
		Field:               source.Field,
	}, nil
}

// findTranslationDraftSource 按资源类型从服务端读取允许外发的展示源文。
func (c *TranslationCase) findTranslationDraftSource(ctx context.Context, resourceType systemadminv1.TranslationResourceType, resourceID int64, field systemadminv1.BaseConfigTranslationField) (*dto.TranslationDraftSource, error) {
	source := &dto.TranslationDraftSource{ResourceType: resourceType, ResourceID: resourceID, Field: field}
	var err error
	switch resourceType {
	case systemadminv1.TranslationResourceType_TRANSLATION_RESOURCE_TYPE_MENU:
		var menu *models.BaseMenu
		menu, err = c.menuRepo.FindByID(ctx, resourceID)
		if err == nil {
			metadata := new(dto.MenuMetadata)
			err = json.Unmarshal([]byte(menu.Meta), metadata)
			if err == nil {
				source.Text = metadata.Title
			}
		}
	case systemadminv1.TranslationResourceType_TRANSLATION_RESOURCE_TYPE_DICT:
		var dict *models.BaseDict
		dict, err = c.dictRepo.FindByID(ctx, resourceID)
		if err == nil {
			source.Text = dict.Name
		}
	case systemadminv1.TranslationResourceType_TRANSLATION_RESOURCE_TYPE_DICT_ITEM:
		var item *models.BaseDictItem
		item, err = c.dictItemRepo.FindByID(ctx, resourceID)
		if err == nil {
			source.Text = item.Label
		}
	case systemadminv1.TranslationResourceType_TRANSLATION_RESOURCE_TYPE_CONFIG:
		var config *models.BaseConfig
		config, err = c.configRepo.FindByID(ctx, resourceID)
		if err == nil {
			source.Text, err = configTranslationSourceText(config, source.Field)
		}
	default:
		return nil, errorsx.InvalidArgument("翻译资源类型无效")
	}
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errorsx.ResourceNotFound("待翻译资源不存在").WithCause(err)
		}
		return nil, errorsx.Internal("读取待翻译资源失败").WithCause(err)
	}
	return source, nil
}

// hasReviewedTranslation 判断目标资源是否已有不可覆盖的人工译文。
func (c *TranslationCase) hasReviewedTranslation(ctx context.Context, source *dto.TranslationDraftSource, localeValue string) (bool, error) {
	var status string
	var err error
	switch source.ResourceType {
	case systemadminv1.TranslationResourceType_TRANSLATION_RESOURCE_TYPE_MENU:
		query := c.menuTranslationRepo.Query(ctx).BaseMenuTranslation
		var row *models.BaseMenuTranslation
		row, err = c.menuTranslationRepo.Find(ctx, repository.Where(query.MenuID.Eq(source.ResourceID)), repository.Where(query.Locale.Eq(localeValue)))
		if err == nil {
			status = row.TranslationStatus
		}
	case systemadminv1.TranslationResourceType_TRANSLATION_RESOURCE_TYPE_DICT:
		query := c.dictTranslationRepo.Query(ctx).BaseDictTranslation
		var row *models.BaseDictTranslation
		row, err = c.dictTranslationRepo.Find(ctx, repository.Where(query.DictID.Eq(source.ResourceID)), repository.Where(query.Locale.Eq(localeValue)))
		if err == nil {
			status = row.TranslationStatus
		}
	case systemadminv1.TranslationResourceType_TRANSLATION_RESOURCE_TYPE_DICT_ITEM:
		query := c.dictItemTranslationRepo.Query(ctx).BaseDictItemTranslation
		var row *models.BaseDictItemTranslation
		row, err = c.dictItemTranslationRepo.Find(ctx, repository.Where(query.DictItemID.Eq(source.ResourceID)), repository.Where(query.Locale.Eq(localeValue)))
		if err == nil {
			status = row.TranslationStatus
		}
	case systemadminv1.TranslationResourceType_TRANSLATION_RESOURCE_TYPE_CONFIG:
		fieldValue, ok := configTranslationFieldValue(source.Field)
		if !ok {
			return false, errorsx.InvalidArgument("系统配置翻译字段无效")
		}
		query := c.configTranslationRepo.Query(ctx).BaseConfigTranslation
		var row *models.BaseConfigTranslation
		row, err = c.configTranslationRepo.Find(ctx, repository.Where(query.ConfigID.Eq(source.ResourceID)), repository.Where(query.Locale.Eq(localeValue)), repository.Where(query.Field.Eq(fieldValue)))
		if err == nil {
			status = row.TranslationStatus
		}
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	return status == _const.TRANSLATION_STATUS_REVIEWED, err
}

// saveMachineDraft 新增或更新可覆盖的机器翻译草稿。
func (c *TranslationCase) saveMachineDraft(ctx context.Context, source *dto.TranslationDraftSource, localeValue, translated, sourceHash string, translatedAt time.Time) error {
	switch source.ResourceType {
	case systemadminv1.TranslationResourceType_TRANSLATION_RESOURCE_TYPE_MENU:
		return c.saveMenuMachineDraft(ctx, source.ResourceID, localeValue, translated, sourceHash, translatedAt)
	case systemadminv1.TranslationResourceType_TRANSLATION_RESOURCE_TYPE_DICT:
		return c.saveDictMachineDraft(ctx, source.ResourceID, localeValue, translated, sourceHash, translatedAt)
	case systemadminv1.TranslationResourceType_TRANSLATION_RESOURCE_TYPE_DICT_ITEM:
		return c.saveDictItemMachineDraft(ctx, source.ResourceID, localeValue, translated, sourceHash, translatedAt)
	case systemadminv1.TranslationResourceType_TRANSLATION_RESOURCE_TYPE_CONFIG:
		return c.saveConfigMachineDraft(ctx, source.ResourceID, source.Field, localeValue, translated, sourceHash, translatedAt)
	default:
		return errorsx.InvalidArgument("翻译资源类型无效")
	}
}

// saveConfigMachineDraft 保存系统配置机器翻译草稿。
func (c *TranslationCase) saveConfigMachineDraft(ctx context.Context, configID int64, field systemadminv1.BaseConfigTranslationField, localeValue, translated, sourceHash string, translatedAt time.Time) error {
	fieldValue, ok := configTranslationFieldValue(field)
	if !ok {
		return errorsx.InvalidArgument("系统配置翻译字段无效")
	}
	query := c.configTranslationRepo.Query(ctx).BaseConfigTranslation
	row, err := c.configTranslationRepo.Find(ctx, repository.Where(query.ConfigID.Eq(configID)), repository.Where(query.Locale.Eq(localeValue)), repository.Where(query.Field.Eq(fieldValue)))
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return c.configTranslationRepo.Create(ctx, &models.BaseConfigTranslation{ConfigID: configID, Locale: localeValue, Field: fieldValue, Text: translated, TranslationStatus: _const.TRANSLATION_STATUS_MACHINE, SourceHash: sourceHash, TranslationProvider: _const.TRANSLATION_PROVIDER_GOOGLE_V1, TranslatedAt: translatedAt})
	}
	if err != nil {
		return err
	}
	row.Text = translated
	row.TranslationStatus = _const.TRANSLATION_STATUS_MACHINE
	row.SourceHash = sourceHash
	row.TranslationProvider = _const.TRANSLATION_PROVIDER_GOOGLE_V1
	row.TranslatedAt = translatedAt
	return c.configTranslationRepo.UpdateByID(ctx, row)
}

// saveMenuMachineDraft 保存菜单机器翻译草稿。
func (c *TranslationCase) saveMenuMachineDraft(ctx context.Context, menuID int64, localeValue, translated, sourceHash string, translatedAt time.Time) error {
	query := c.menuTranslationRepo.Query(ctx).BaseMenuTranslation
	row, err := c.menuTranslationRepo.Find(ctx, repository.Where(query.MenuID.Eq(menuID)), repository.Where(query.Locale.Eq(localeValue)))
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return c.menuTranslationRepo.Create(ctx, &models.BaseMenuTranslation{MenuID: menuID, Locale: localeValue, Title: translated, TranslationStatus: _const.TRANSLATION_STATUS_MACHINE, SourceHash: sourceHash, TranslationProvider: _const.TRANSLATION_PROVIDER_GOOGLE_V1, TranslatedAt: translatedAt})
	}
	if err != nil {
		return err
	}
	row.Title = translated
	row.TranslationStatus = _const.TRANSLATION_STATUS_MACHINE
	row.SourceHash = sourceHash
	row.TranslationProvider = _const.TRANSLATION_PROVIDER_GOOGLE_V1
	row.TranslatedAt = translatedAt
	return c.menuTranslationRepo.UpdateByID(ctx, row)
}

// saveDictMachineDraft 保存字典机器翻译草稿。
func (c *TranslationCase) saveDictMachineDraft(ctx context.Context, dictID int64, localeValue, translated, sourceHash string, translatedAt time.Time) error {
	query := c.dictTranslationRepo.Query(ctx).BaseDictTranslation
	row, err := c.dictTranslationRepo.Find(ctx, repository.Where(query.DictID.Eq(dictID)), repository.Where(query.Locale.Eq(localeValue)))
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return c.dictTranslationRepo.Create(ctx, &models.BaseDictTranslation{DictID: dictID, Locale: localeValue, Name: translated, TranslationStatus: _const.TRANSLATION_STATUS_MACHINE, SourceHash: sourceHash, TranslationProvider: _const.TRANSLATION_PROVIDER_GOOGLE_V1, TranslatedAt: translatedAt})
	}
	if err != nil {
		return err
	}
	row.Name = translated
	row.TranslationStatus = _const.TRANSLATION_STATUS_MACHINE
	row.SourceHash = sourceHash
	row.TranslationProvider = _const.TRANSLATION_PROVIDER_GOOGLE_V1
	row.TranslatedAt = translatedAt
	return c.dictTranslationRepo.UpdateByID(ctx, row)
}

// saveDictItemMachineDraft 保存字典项机器翻译草稿。
func (c *TranslationCase) saveDictItemMachineDraft(ctx context.Context, dictItemID int64, localeValue, translated, sourceHash string, translatedAt time.Time) error {
	query := c.dictItemTranslationRepo.Query(ctx).BaseDictItemTranslation
	row, err := c.dictItemTranslationRepo.Find(ctx, repository.Where(query.DictItemID.Eq(dictItemID)), repository.Where(query.Locale.Eq(localeValue)))
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return c.dictItemTranslationRepo.Create(ctx, &models.BaseDictItemTranslation{DictItemID: dictItemID, Locale: localeValue, Label: translated, TranslationStatus: _const.TRANSLATION_STATUS_MACHINE, SourceHash: sourceHash, TranslationProvider: _const.TRANSLATION_PROVIDER_GOOGLE_V1, TranslatedAt: translatedAt})
	}
	if err != nil {
		return err
	}
	row.Label = translated
	row.TranslationStatus = _const.TRANSLATION_STATUS_MACHINE
	row.SourceHash = sourceHash
	row.TranslationProvider = _const.TRANSLATION_PROVIDER_GOOGLE_V1
	row.TranslatedAt = translatedAt
	return c.dictItemTranslationRepo.UpdateByID(ctx, row)
}

// ReviewedMenuTitles 批量查询当前语言已审核的菜单标题。
func (c *TranslationCase) ReviewedMenuTitles(ctx context.Context, menuIDs []int64) (map[int64]string, error) {
	result := make(map[int64]string)
	localeValue := coreLocale.FromContext(ctx)
	if localeValue == coreLocale.Default || len(menuIDs) == 0 {
		return result, nil
	}
	query := c.menuTranslationRepo.Query(ctx).BaseMenuTranslation
	opts := make([]repository.QueryOption, 0, 3)
	opts = append(opts, repository.Where(query.MenuID.In(menuIDs...)))
	opts = append(opts, repository.Where(query.Locale.Eq(localeValue)))
	opts = append(opts, repository.Where(query.TranslationStatus.Eq(_const.TRANSLATION_STATUS_REVIEWED)))
	rows, err := c.menuTranslationRepo.List(ctx, opts...)
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		result[row.MenuID] = row.Title
	}
	return result, nil
}

// MenuTranslations 批量查询菜单维护界面需要的完整翻译状态。
func (c *TranslationCase) MenuTranslations(ctx context.Context, sources map[int64]string) (map[int64][]*systemadminv1.BaseMenuTranslation, error) {
	result := make(map[int64][]*systemadminv1.BaseMenuTranslation, len(sources))
	if len(sources) == 0 {
		return result, nil
	}
	ids := translationSourceIDs(sources)
	query := c.menuTranslationRepo.Query(ctx).BaseMenuTranslation
	rows, err := c.menuTranslationRepo.List(ctx, repository.Where(query.MenuID.In(ids...)))
	if err != nil {
		return nil, err
	}
	rowMap := make(map[dto.TranslationKey]*models.BaseMenuTranslation, len(rows))
	for _, row := range rows {
		rowMap[dto.TranslationKey{ResourceID: row.MenuID, Locale: row.Locale}] = row
	}
	for resourceID, source := range sources {
		translations := make([]*systemadminv1.BaseMenuTranslation, 0, len(editableLocales))
		for _, localeValue := range editableLocales {
			row := rowMap[dto.TranslationKey{ResourceID: resourceID, Locale: localeValue}]
			translations = append(translations, menuTranslationDTO(row, resourceID, localeValue, source))
		}
		result[resourceID] = translations
	}
	return result, nil
}

// SaveMenuTranslations 将人工维护的菜单译文保存为已审核状态。
func (c *TranslationCase) SaveMenuTranslations(ctx context.Context, menuID int64, source string, translations []*systemadminv1.BaseMenuTranslation) error {
	if translations == nil {
		return nil
	}
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}
	query := c.menuTranslationRepo.Query(ctx).BaseMenuTranslation
	var existingRows []*models.BaseMenuTranslation
	existingRows, err = c.menuTranslationRepo.List(ctx, repository.Where(query.MenuID.Eq(menuID)))
	if err != nil {
		return err
	}
	existing := make(map[string]*models.BaseMenuTranslation, len(existingRows))
	for _, row := range existingRows {
		existing[row.Locale] = row
	}
	seen := make(map[string]struct{}, len(translations))
	now := time.Now()
	for _, translation := range translations {
		localeValue, ok := editableLocale(translation.GetLocale())
		if !ok {
			return errorsx.InvalidArgument("菜单翻译语言仅支持英语或日语")
		}
		if _, duplicated := seen[localeValue]; duplicated {
			return errorsx.Conflict("同一菜单语言不能重复")
		}
		seen[localeValue] = struct{}{}
		row := existing[localeValue]
		if translation.GetTitle() == "" {
			if row != nil {
				err = c.menuTranslationRepo.DeleteByID(ctx, row.ID)
				if err != nil {
					return err
				}
			}
			continue
		}
		if row == nil {
			row = &models.BaseMenuTranslation{MenuID: menuID, Locale: localeValue}
		}
		row.Title = translation.GetTitle()
		row.TranslationStatus = _const.TRANSLATION_STATUS_REVIEWED
		row.SourceHash = translationSourceHash(source)
		row.ReviewedBy = authInfo.UserId
		row.ReviewedAt = now
		if row.ID == 0 {
			err = query.WithContext(ctx).Omit(query.TranslationProvider, query.TranslatedAt).Create(row)
		} else {
			_, err = query.WithContext(ctx).Where(query.ID.Eq(row.ID)).UpdateSimple(
				query.Title.Value(row.Title),
				query.TranslationStatus.Value(row.TranslationStatus),
				query.SourceHash.Value(row.SourceHash),
				query.ReviewedBy.Value(row.ReviewedBy),
				query.ReviewedAt.Value(row.ReviewedAt),
			)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// SaveGeneratedMenuTranslations 保存代码生成器提供的已审核译文，并保留已有人工审核内容。
func (c *TranslationCase) SaveGeneratedMenuTranslations(ctx context.Context, menuID int64, source string, translations map[string]string) error {
	if len(translations) == 0 {
		return nil
	}
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}
	query := c.menuTranslationRepo.Query(ctx).BaseMenuTranslation
	var rows []*models.BaseMenuTranslation
	rows, err = c.menuTranslationRepo.List(ctx, repository.Where(query.MenuID.Eq(menuID)))
	if err != nil {
		return err
	}
	existing := make(map[string]*models.BaseMenuTranslation, len(rows))
	for _, row := range rows {
		existing[row.Locale] = row
	}
	now := time.Now()
	for _, localeValue := range editableLocales {
		title := translations[localeValue]
		if title == "" {
			continue
		}
		row := existing[localeValue]
		if row != nil && row.TranslationStatus == _const.TRANSLATION_STATUS_REVIEWED {
			continue
		}
		if row == nil {
			row = &models.BaseMenuTranslation{MenuID: menuID, Locale: localeValue}
		}
		row.Title = title
		row.TranslationStatus = _const.TRANSLATION_STATUS_REVIEWED
		row.SourceHash = translationSourceHash(source)
		row.TranslationProvider = ""
		row.ReviewedBy = authInfo.UserId
		row.ReviewedAt = now
		if row.ID == 0 {
			err = query.WithContext(ctx).Omit(query.TranslationProvider, query.TranslatedAt).Create(row)
		} else {
			_, err = query.WithContext(ctx).Where(query.ID.Eq(row.ID)).UpdateSimple(
				query.Title.Value(row.Title),
				query.TranslationStatus.Value(row.TranslationStatus),
				query.SourceHash.Value(row.SourceHash),
				query.TranslationProvider.Null(),
				query.TranslatedAt.Null(),
				query.ReviewedBy.Value(row.ReviewedBy),
				query.ReviewedAt.Value(row.ReviewedAt),
			)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// MarkMenuSourceChanged 将过期机器菜单译文标记为待处理。
func (c *TranslationCase) MarkMenuSourceChanged(ctx context.Context, menuID int64, previousSource, currentSource string) error {
	if previousSource == currentSource {
		return nil
	}
	query := c.menuTranslationRepo.Query(ctx).BaseMenuTranslation
	rows, err := c.menuTranslationRepo.List(ctx, repository.Where(query.MenuID.Eq(menuID)))
	if err != nil {
		return err
	}
	for _, row := range rows {
		_, err = query.WithContext(ctx).Where(query.ID.Eq(row.ID)).UpdateSimple(query.TranslationStatus.Value(_const.TRANSLATION_STATUS_PENDING))
		if err != nil {
			return err
		}
	}
	return nil
}

// DeleteMenuTranslations 删除菜单时同步软删除其翻译记录。
func (c *TranslationCase) DeleteMenuTranslations(ctx context.Context, menuIDs []int64) error {
	if len(menuIDs) == 0 {
		return nil
	}
	query := c.menuTranslationRepo.Query(ctx).BaseMenuTranslation
	return c.menuTranslationRepo.Delete(ctx, repository.Where(query.MenuID.In(menuIDs...)))
}

// ReviewedDictNames 批量查询当前语言已审核的字典名称。
func (c *TranslationCase) ReviewedDictNames(ctx context.Context, dictIDs []int64) (map[int64]string, error) {
	result := make(map[int64]string)
	localeValue := coreLocale.FromContext(ctx)
	if localeValue == coreLocale.Default || len(dictIDs) == 0 {
		return result, nil
	}
	query := c.dictTranslationRepo.Query(ctx).BaseDictTranslation
	rows, err := c.dictTranslationRepo.List(ctx, repository.Where(query.DictID.In(dictIDs...)), repository.Where(query.Locale.Eq(localeValue)), repository.Where(query.TranslationStatus.Eq(_const.TRANSLATION_STATUS_REVIEWED)))
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		result[row.DictID] = row.Name
	}
	return result, nil
}

// ReviewedDictIDsByName 查询当前语言名称包含关键字的已审核字典编号。
func (c *TranslationCase) ReviewedDictIDsByName(ctx context.Context, keyword string) ([]int64, error) {
	localeValue := coreLocale.FromContext(ctx)
	if localeValue == coreLocale.Default || keyword == "" {
		return nil, nil
	}
	query := c.dictTranslationRepo.Query(ctx).BaseDictTranslation
	rows, err := c.dictTranslationRepo.List(ctx, repository.Where(query.Locale.Eq(localeValue)), repository.Where(query.TranslationStatus.Eq(_const.TRANSLATION_STATUS_REVIEWED)), repository.Where(query.Name.Like("%"+keyword+"%")))
	if err != nil {
		return nil, err
	}
	ids := make([]int64, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.DictID)
	}
	return ids, nil
}

// DictTranslations 批量查询字典维护界面需要的完整翻译状态。
func (c *TranslationCase) DictTranslations(ctx context.Context, sources map[int64]string) (map[int64][]*systemadminv1.BaseDictTranslation, error) {
	result := make(map[int64][]*systemadminv1.BaseDictTranslation, len(sources))
	if len(sources) == 0 {
		return result, nil
	}
	ids := translationSourceIDs(sources)
	query := c.dictTranslationRepo.Query(ctx).BaseDictTranslation
	rows, err := c.dictTranslationRepo.List(ctx, repository.Where(query.DictID.In(ids...)))
	if err != nil {
		return nil, err
	}
	rowMap := make(map[dto.TranslationKey]*models.BaseDictTranslation, len(rows))
	for _, row := range rows {
		rowMap[dto.TranslationKey{ResourceID: row.DictID, Locale: row.Locale}] = row
	}
	for resourceID, source := range sources {
		translations := make([]*systemadminv1.BaseDictTranslation, 0, len(editableLocales))
		for _, localeValue := range editableLocales {
			row := rowMap[dto.TranslationKey{ResourceID: resourceID, Locale: localeValue}]
			translations = append(translations, dictTranslationDTO(row, resourceID, localeValue, source))
		}
		result[resourceID] = translations
	}
	return result, nil
}

// SaveDictTranslations 将人工维护的字典译文保存为已审核状态。
func (c *TranslationCase) SaveDictTranslations(ctx context.Context, dictID int64, source string, translations []*systemadminv1.BaseDictTranslation) error {
	if translations == nil {
		return nil
	}
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}
	query := c.dictTranslationRepo.Query(ctx).BaseDictTranslation
	var existingRows []*models.BaseDictTranslation
	existingRows, err = c.dictTranslationRepo.List(ctx, repository.Where(query.DictID.Eq(dictID)))
	if err != nil {
		return err
	}
	existing := make(map[string]*models.BaseDictTranslation, len(existingRows))
	for _, row := range existingRows {
		existing[row.Locale] = row
	}
	seen := make(map[string]struct{}, len(translations))
	now := time.Now()
	for _, translation := range translations {
		localeValue, ok := editableLocale(translation.GetLocale())
		if !ok {
			return errorsx.InvalidArgument("字典翻译语言仅支持英语或日语")
		}
		if _, duplicated := seen[localeValue]; duplicated {
			return errorsx.Conflict("同一字典语言不能重复")
		}
		seen[localeValue] = struct{}{}
		row := existing[localeValue]
		if translation.GetName() == "" {
			if row != nil {
				err = c.dictTranslationRepo.DeleteByID(ctx, row.ID)
				if err != nil {
					return err
				}
			}
			continue
		}
		if row == nil {
			row = &models.BaseDictTranslation{DictID: dictID, Locale: localeValue}
		}
		row.Name = translation.GetName()
		row.TranslationStatus = _const.TRANSLATION_STATUS_REVIEWED
		row.SourceHash = translationSourceHash(source)
		row.ReviewedBy = authInfo.UserId
		row.ReviewedAt = now
		if row.ID == 0 {
			err = query.WithContext(ctx).Omit(query.TranslationProvider, query.TranslatedAt).Create(row)
		} else {
			_, err = query.WithContext(ctx).Where(query.ID.Eq(row.ID)).UpdateSimple(
				query.Name.Value(row.Name),
				query.TranslationStatus.Value(row.TranslationStatus),
				query.SourceHash.Value(row.SourceHash),
				query.ReviewedBy.Value(row.ReviewedBy),
				query.ReviewedAt.Value(row.ReviewedAt),
			)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// MarkDictSourceChanged 将过期机器字典译文标记为待处理。
func (c *TranslationCase) MarkDictSourceChanged(ctx context.Context, dictID int64, previousSource, currentSource string) error {
	if previousSource == currentSource {
		return nil
	}
	query := c.dictTranslationRepo.Query(ctx).BaseDictTranslation
	rows, err := c.dictTranslationRepo.List(ctx, repository.Where(query.DictID.Eq(dictID)))
	if err != nil {
		return err
	}
	for _, row := range rows {
		_, err = query.WithContext(ctx).Where(query.ID.Eq(row.ID)).UpdateSimple(query.TranslationStatus.Value(_const.TRANSLATION_STATUS_PENDING))
		if err != nil {
			return err
		}
	}
	return nil
}

// DeleteDictTranslations 删除字典时同步软删除其翻译记录。
func (c *TranslationCase) DeleteDictTranslations(ctx context.Context, dictIDs []int64) error {
	if len(dictIDs) == 0 {
		return nil
	}
	query := c.dictTranslationRepo.Query(ctx).BaseDictTranslation
	return c.dictTranslationRepo.Delete(ctx, repository.Where(query.DictID.In(dictIDs...)))
}

// ReviewedDictItemLabels 批量查询当前语言已审核的字典项标签。
func (c *TranslationCase) ReviewedDictItemLabels(ctx context.Context, dictItemIDs []int64) (map[int64]string, error) {
	result := make(map[int64]string)
	localeValue := coreLocale.FromContext(ctx)
	if localeValue == coreLocale.Default || len(dictItemIDs) == 0 {
		return result, nil
	}
	query := c.dictItemTranslationRepo.Query(ctx).BaseDictItemTranslation
	rows, err := c.dictItemTranslationRepo.List(ctx, repository.Where(query.DictItemID.In(dictItemIDs...)), repository.Where(query.Locale.Eq(localeValue)), repository.Where(query.TranslationStatus.Eq(_const.TRANSLATION_STATUS_REVIEWED)))
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		result[row.DictItemID] = row.Label
	}
	return result, nil
}

// ReviewedDictItemIDsByLabel 查询当前语言标签包含关键字的已审核字典项编号。
func (c *TranslationCase) ReviewedDictItemIDsByLabel(ctx context.Context, keyword string) ([]int64, error) {
	localeValue := coreLocale.FromContext(ctx)
	if localeValue == coreLocale.Default || keyword == "" {
		return nil, nil
	}
	query := c.dictItemTranslationRepo.Query(ctx).BaseDictItemTranslation
	rows, err := c.dictItemTranslationRepo.List(ctx, repository.Where(query.Locale.Eq(localeValue)), repository.Where(query.TranslationStatus.Eq(_const.TRANSLATION_STATUS_REVIEWED)), repository.Where(query.Label.Like("%"+keyword+"%")))
	if err != nil {
		return nil, err
	}
	ids := make([]int64, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.DictItemID)
	}
	return ids, nil
}

// DictItemTranslations 批量查询字典项维护界面需要的完整翻译状态。
func (c *TranslationCase) DictItemTranslations(ctx context.Context, sources map[int64]string) (map[int64][]*systemadminv1.BaseDictItemTranslation, error) {
	result := make(map[int64][]*systemadminv1.BaseDictItemTranslation, len(sources))
	if len(sources) == 0 {
		return result, nil
	}
	ids := translationSourceIDs(sources)
	query := c.dictItemTranslationRepo.Query(ctx).BaseDictItemTranslation
	rows, err := c.dictItemTranslationRepo.List(ctx, repository.Where(query.DictItemID.In(ids...)))
	if err != nil {
		return nil, err
	}
	rowMap := make(map[dto.TranslationKey]*models.BaseDictItemTranslation, len(rows))
	for _, row := range rows {
		rowMap[dto.TranslationKey{ResourceID: row.DictItemID, Locale: row.Locale}] = row
	}
	for resourceID, source := range sources {
		translations := make([]*systemadminv1.BaseDictItemTranslation, 0, len(editableLocales))
		for _, localeValue := range editableLocales {
			row := rowMap[dto.TranslationKey{ResourceID: resourceID, Locale: localeValue}]
			translations = append(translations, dictItemTranslationDTO(row, resourceID, localeValue, source))
		}
		result[resourceID] = translations
	}
	return result, nil
}

// SaveDictItemTranslations 将人工维护的字典项译文保存为已审核状态。
func (c *TranslationCase) SaveDictItemTranslations(ctx context.Context, dictItemID int64, source string, translations []*systemadminv1.BaseDictItemTranslation) error {
	if translations == nil {
		return nil
	}
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}
	query := c.dictItemTranslationRepo.Query(ctx).BaseDictItemTranslation
	var existingRows []*models.BaseDictItemTranslation
	existingRows, err = c.dictItemTranslationRepo.List(ctx, repository.Where(query.DictItemID.Eq(dictItemID)))
	if err != nil {
		return err
	}
	existing := make(map[string]*models.BaseDictItemTranslation, len(existingRows))
	for _, row := range existingRows {
		existing[row.Locale] = row
	}
	seen := make(map[string]struct{}, len(translations))
	now := time.Now()
	for _, translation := range translations {
		localeValue, ok := editableLocale(translation.GetLocale())
		if !ok {
			return errorsx.InvalidArgument("字典项翻译语言仅支持英语或日语")
		}
		if _, duplicated := seen[localeValue]; duplicated {
			return errorsx.Conflict("同一字典项语言不能重复")
		}
		seen[localeValue] = struct{}{}
		row := existing[localeValue]
		if translation.GetLabel() == "" {
			if row != nil {
				err = c.dictItemTranslationRepo.DeleteByID(ctx, row.ID)
				if err != nil {
					return err
				}
			}
			continue
		}
		if row == nil {
			row = &models.BaseDictItemTranslation{DictItemID: dictItemID, Locale: localeValue}
		}
		row.Label = translation.GetLabel()
		row.TranslationStatus = _const.TRANSLATION_STATUS_REVIEWED
		row.SourceHash = translationSourceHash(source)
		row.ReviewedBy = authInfo.UserId
		row.ReviewedAt = now
		if row.ID == 0 {
			err = query.WithContext(ctx).Omit(query.TranslationProvider, query.TranslatedAt).Create(row)
		} else {
			_, err = query.WithContext(ctx).Where(query.ID.Eq(row.ID)).UpdateSimple(
				query.Label.Value(row.Label),
				query.TranslationStatus.Value(row.TranslationStatus),
				query.SourceHash.Value(row.SourceHash),
				query.ReviewedBy.Value(row.ReviewedBy),
				query.ReviewedAt.Value(row.ReviewedAt),
			)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// MarkDictItemSourceChanged 将过期机器字典项译文标记为待处理。
func (c *TranslationCase) MarkDictItemSourceChanged(ctx context.Context, dictItemID int64, previousSource, currentSource string) error {
	if previousSource == currentSource {
		return nil
	}
	query := c.dictItemTranslationRepo.Query(ctx).BaseDictItemTranslation
	rows, err := c.dictItemTranslationRepo.List(ctx, repository.Where(query.DictItemID.Eq(dictItemID)))
	if err != nil {
		return err
	}
	for _, row := range rows {
		_, err = query.WithContext(ctx).Where(query.ID.Eq(row.ID)).UpdateSimple(query.TranslationStatus.Value(_const.TRANSLATION_STATUS_PENDING))
		if err != nil {
			return err
		}
	}
	return nil
}

// DeleteDictItemTranslations 删除字典项时同步软删除其翻译记录。
func (c *TranslationCase) DeleteDictItemTranslations(ctx context.Context, dictItemIDs []int64) error {
	if len(dictItemIDs) == 0 {
		return nil
	}
	query := c.dictItemTranslationRepo.Query(ctx).BaseDictItemTranslation
	return c.dictItemTranslationRepo.Delete(ctx, repository.Where(query.DictItemID.In(dictItemIDs...)))
}

// ReviewedConfigValues 批量查询当前语言已审核的文本配置值。
func (c *TranslationCase) ReviewedConfigValues(ctx context.Context, configIDs []int64) (map[int64]string, error) {
	result := make(map[int64]string)
	localeValue := coreLocale.FromContext(ctx)
	if localeValue == coreLocale.Default || len(configIDs) == 0 {
		return result, nil
	}
	query := c.configTranslationRepo.Query(ctx).BaseConfigTranslation
	rows, err := c.configTranslationRepo.List(ctx, repository.Where(query.ConfigID.In(configIDs...)), repository.Where(query.Locale.Eq(localeValue)), repository.Where(query.Field.Eq("value")), repository.Where(query.TranslationStatus.Eq(_const.TRANSLATION_STATUS_REVIEWED)))
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		result[row.ConfigID] = row.Text
	}
	return result, nil
}

// ReviewedConfigNames 批量查询当前语言已审核的系统配置名称。
func (c *TranslationCase) ReviewedConfigNames(ctx context.Context, configIDs []int64) (map[int64]string, error) {
	result := make(map[int64]string)
	localeValue := coreLocale.FromContext(ctx)
	if localeValue == coreLocale.Default || len(configIDs) == 0 {
		return result, nil
	}
	query := c.configTranslationRepo.Query(ctx).BaseConfigTranslation
	rows, err := c.configTranslationRepo.List(ctx, repository.Where(query.ConfigID.In(configIDs...)), repository.Where(query.Locale.Eq(localeValue)), repository.Where(query.Field.Eq("name")), repository.Where(query.TranslationStatus.Eq(_const.TRANSLATION_STATUS_REVIEWED)))
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		result[row.ConfigID] = row.Text
	}
	return result, nil
}

// ReviewedConfigIDsByName 查询当前语言名称包含关键字的已审核系统配置编号。
func (c *TranslationCase) ReviewedConfigIDsByName(ctx context.Context, keyword string) ([]int64, error) {
	localeValue := coreLocale.FromContext(ctx)
	if localeValue == coreLocale.Default || keyword == "" {
		return nil, nil
	}
	query := c.configTranslationRepo.Query(ctx).BaseConfigTranslation
	rows, err := c.configTranslationRepo.List(ctx, repository.Where(query.Locale.Eq(localeValue)), repository.Where(query.Field.Eq("name")), repository.Where(query.TranslationStatus.Eq(_const.TRANSLATION_STATUS_REVIEWED)), repository.Where(query.Text.Like("%"+keyword+"%")))
	if err != nil {
		return nil, err
	}
	ids := make([]int64, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ConfigID)
	}
	return ids, nil
}

// ConfigTranslations 批量查询系统配置维护界面的翻译状态。
func (c *TranslationCase) ConfigTranslations(ctx context.Context, sources map[int64]dto.ConfigTranslationSource) (map[int64][]*systemadminv1.BaseConfigTranslation, error) {
	result := make(map[int64][]*systemadminv1.BaseConfigTranslation, len(sources))
	if len(sources) == 0 {
		return result, nil
	}
	ids := make([]int64, 0, len(sources))
	for configID := range sources {
		ids = append(ids, configID)
	}
	query := c.configTranslationRepo.Query(ctx).BaseConfigTranslation
	rows, err := c.configTranslationRepo.List(ctx, repository.Where(query.ConfigID.In(ids...)))
	if err != nil {
		return nil, err
	}
	rowMap := make(map[string]*models.BaseConfigTranslation, len(rows))
	for _, row := range rows {
		rowMap[configTranslationKey(row.ConfigID, row.Locale, row.Field)] = row
	}
	for configID, source := range sources {
		fields := []systemadminv1.BaseConfigTranslationField{systemadminv1.BaseConfigTranslationField_BASE_CONFIG_TRANSLATION_FIELD_NAME}
		if isTranslatableConfigType(source.Type) {
			fields = append(fields, systemadminv1.BaseConfigTranslationField_BASE_CONFIG_TRANSLATION_FIELD_VALUE)
		}
		translations := make([]*systemadminv1.BaseConfigTranslation, 0, len(editableLocales)*len(fields))
		for _, field := range fields {
			fieldValue, _ := configTranslationFieldValue(field)
			sourceText := source.Name
			if field == systemadminv1.BaseConfigTranslationField_BASE_CONFIG_TRANSLATION_FIELD_VALUE {
				sourceText = source.Value
			}
			for _, localeValue := range editableLocales {
				row := rowMap[configTranslationKey(configID, localeValue, fieldValue)]
				translations = append(translations, configTranslationDTO(row, configID, localeValue, field, sourceText))
			}
		}
		result[configID] = translations
	}
	return result, nil
}

// SaveConfigTranslations 将人工维护的系统配置译文保存为已审核状态。
func (c *TranslationCase) SaveConfigTranslations(ctx context.Context, configID int64, source dto.ConfigTranslationSource, translations []*systemadminv1.BaseConfigTranslation) error {
	if translations == nil {
		return nil
	}
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}
	query := c.configTranslationRepo.Query(ctx).BaseConfigTranslation
	var existingRows []*models.BaseConfigTranslation
	existingRows, err = c.configTranslationRepo.List(ctx, repository.Where(query.ConfigID.Eq(configID)))
	if err != nil {
		return err
	}
	existing := make(map[string]*models.BaseConfigTranslation, len(existingRows))
	for _, row := range existingRows {
		existing[configTranslationKey(row.ConfigID, row.Locale, row.Field)] = row
	}
	seen := make(map[string]struct{}, len(translations))
	now := time.Now()
	for _, translation := range translations {
		localeValue, ok := editableLocale(translation.GetLocale())
		if !ok {
			return errorsx.InvalidArgument("系统配置翻译语言仅支持英语或日语")
		}
		fieldValue, ok := configTranslationFieldValue(translation.GetField())
		if !ok || (fieldValue == "value" && !isTranslatableConfigType(source.Type)) {
			return errorsx.InvalidArgument("当前系统配置类型不支持该翻译字段")
		}
		key := configTranslationKey(configID, localeValue, fieldValue)
		if _, duplicated := seen[key]; duplicated {
			return errorsx.Conflict("同一系统配置语言和字段不能重复")
		}
		seen[key] = struct{}{}
		row := existing[key]
		if translation.GetText() == "" {
			if row != nil {
				err = c.configTranslationRepo.DeleteByID(ctx, row.ID)
				if err != nil {
					return err
				}
			}
			continue
		}
		if row == nil {
			row = &models.BaseConfigTranslation{ConfigID: configID, Locale: localeValue, Field: fieldValue}
		}
		sourceText := source.Name
		if fieldValue == "value" {
			sourceText = source.Value
		}
		row.Text = translation.GetText()
		row.TranslationStatus = _const.TRANSLATION_STATUS_REVIEWED
		row.SourceHash = translationSourceHash(sourceText)
		row.ReviewedBy = authInfo.UserId
		row.ReviewedAt = now
		if row.ID == 0 {
			err = query.WithContext(ctx).Omit(query.TranslationProvider, query.TranslatedAt).Create(row)
		} else {
			_, err = query.WithContext(ctx).Where(query.ID.Eq(row.ID)).UpdateSimple(
				query.Text.Value(row.Text),
				query.TranslationStatus.Value(row.TranslationStatus),
				query.SourceHash.Value(row.SourceHash),
				query.ReviewedBy.Value(row.ReviewedBy),
				query.ReviewedAt.Value(row.ReviewedAt),
			)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// MarkConfigSourceChanged 将源文变化的系统配置译文标记为待处理。
func (c *TranslationCase) MarkConfigSourceChanged(ctx context.Context, configID int64, previousSource, currentSource dto.ConfigTranslationSource) error {
	query := c.configTranslationRepo.Query(ctx).BaseConfigTranslation
	rows, err := c.configTranslationRepo.List(ctx, repository.Where(query.ConfigID.Eq(configID)))
	if err != nil {
		return err
	}
	for _, row := range rows {
		changed := previousSource.Name != currentSource.Name && row.Field == "name"
		changed = changed || (previousSource.Value != currentSource.Value && row.Field == "value")
		if !changed {
			continue
		}
		_, err = query.WithContext(ctx).Where(query.ID.Eq(row.ID)).UpdateSimple(query.TranslationStatus.Value(_const.TRANSLATION_STATUS_PENDING))
		if err != nil {
			return err
		}
	}
	return nil
}

// DeleteConfigTranslations 删除系统配置时同步软删除其翻译记录。
func (c *TranslationCase) DeleteConfigTranslations(ctx context.Context, configIDs []int64) error {
	if len(configIDs) == 0 {
		return nil
	}
	query := c.configTranslationRepo.Query(ctx).BaseConfigTranslation
	return c.configTranslationRepo.Delete(ctx, repository.Where(query.ConfigID.In(configIDs...)))
}

// editableLocale 规范化并限制可维护的非默认语言。
func editableLocale(value string) (string, bool) {
	if !coreLocale.IsSupported(value) {
		return "", false
	}
	normalized := coreLocale.Normalize(value)
	return normalized, normalized != coreLocale.Default
}

// translationSourceHash 计算原始中文源文的SHA-256。
func translationSourceHash(source string) string {
	hash := sha256.Sum256([]byte(source))
	return hex.EncodeToString(hash[:])
}

// translationSourceIDs 提取批量翻译源文的资源ID。
func translationSourceIDs(sources map[int64]string) []int64 {
	ids := make([]int64, 0, len(sources))
	for resourceID := range sources {
		ids = append(ids, resourceID)
	}
	return ids
}

// translationStatusDTO 将数据库状态转换为接口枚举。
func translationStatusDTO(value string) systemadminv1.TranslationStatus {
	switch value {
	case _const.TRANSLATION_STATUS_PENDING:
		return systemadminv1.TranslationStatus_TRANSLATION_STATUS_PENDING
	case _const.TRANSLATION_STATUS_MACHINE:
		return systemadminv1.TranslationStatus_TRANSLATION_STATUS_MACHINE
	case _const.TRANSLATION_STATUS_REVIEWED:
		return systemadminv1.TranslationStatus_TRANSLATION_STATUS_REVIEWED
	default:
		return systemadminv1.TranslationStatus_TRANSLATION_STATUS_UNSPECIFIED
	}
}

// menuTranslationDTO 转换菜单翻译维护数据并计算源文变化状态。
func menuTranslationDTO(row *models.BaseMenuTranslation, resourceID int64, localeValue, source string) *systemadminv1.BaseMenuTranslation {
	if row == nil {
		return &systemadminv1.BaseMenuTranslation{MenuId: resourceID, Locale: localeValue, TranslationStatus: systemadminv1.TranslationStatus_TRANSLATION_STATUS_PENDING, SourceHash: translationSourceHash(source)}
	}
	return &systemadminv1.BaseMenuTranslation{Id: row.ID, MenuId: row.MenuID, Locale: row.Locale, Title: row.Title, TranslationStatus: translationStatusDTO(row.TranslationStatus), SourceHash: row.SourceHash, TranslationProvider: row.TranslationProvider, TranslatedAt: formatTranslationTime(row.TranslatedAt), ReviewedBy: row.ReviewedBy, ReviewedAt: formatTranslationTime(row.ReviewedAt), CreatedBy: row.CreatedBy, UpdatedBy: row.UpdatedBy, CreatedAt: formatTranslationTime(row.CreatedAt), UpdatedAt: formatTranslationTime(row.UpdatedAt), DeletedAt: uint64(row.DeletedAt), SourceChanged: row.SourceHash != translationSourceHash(source)}
}

// dictTranslationDTO 转换字典翻译维护数据并计算源文变化状态。
func dictTranslationDTO(row *models.BaseDictTranslation, resourceID int64, localeValue, source string) *systemadminv1.BaseDictTranslation {
	if row == nil {
		return &systemadminv1.BaseDictTranslation{DictId: resourceID, Locale: localeValue, TranslationStatus: systemadminv1.TranslationStatus_TRANSLATION_STATUS_PENDING, SourceHash: translationSourceHash(source)}
	}
	return &systemadminv1.BaseDictTranslation{Id: row.ID, DictId: row.DictID, Locale: row.Locale, Name: row.Name, TranslationStatus: translationStatusDTO(row.TranslationStatus), SourceHash: row.SourceHash, TranslationProvider: row.TranslationProvider, TranslatedAt: formatTranslationTime(row.TranslatedAt), ReviewedBy: row.ReviewedBy, ReviewedAt: formatTranslationTime(row.ReviewedAt), CreatedBy: row.CreatedBy, UpdatedBy: row.UpdatedBy, CreatedAt: formatTranslationTime(row.CreatedAt), UpdatedAt: formatTranslationTime(row.UpdatedAt), DeletedAt: uint64(row.DeletedAt), SourceChanged: row.SourceHash != translationSourceHash(source)}
}

// dictItemTranslationDTO 转换字典项翻译维护数据并计算源文变化状态。
func dictItemTranslationDTO(row *models.BaseDictItemTranslation, resourceID int64, localeValue, source string) *systemadminv1.BaseDictItemTranslation {
	if row == nil {
		return &systemadminv1.BaseDictItemTranslation{DictItemId: resourceID, Locale: localeValue, TranslationStatus: systemadminv1.TranslationStatus_TRANSLATION_STATUS_PENDING, SourceHash: translationSourceHash(source)}
	}
	return &systemadminv1.BaseDictItemTranslation{Id: row.ID, DictItemId: row.DictItemID, Locale: row.Locale, Label: row.Label, TranslationStatus: translationStatusDTO(row.TranslationStatus), SourceHash: row.SourceHash, TranslationProvider: row.TranslationProvider, TranslatedAt: formatTranslationTime(row.TranslatedAt), ReviewedBy: row.ReviewedBy, ReviewedAt: formatTranslationTime(row.ReviewedAt), CreatedBy: row.CreatedBy, UpdatedBy: row.UpdatedBy, CreatedAt: formatTranslationTime(row.CreatedAt), UpdatedAt: formatTranslationTime(row.UpdatedAt), DeletedAt: uint64(row.DeletedAt), SourceChanged: row.SourceHash != translationSourceHash(source)}
}

// configTranslationDTO 转换系统配置翻译维护数据并计算源文变化状态。
func configTranslationDTO(row *models.BaseConfigTranslation, resourceID int64, localeValue string, field systemadminv1.BaseConfigTranslationField, source string) *systemadminv1.BaseConfigTranslation {
	if row == nil {
		return &systemadminv1.BaseConfigTranslation{ConfigId: resourceID, Locale: localeValue, Field: field, TranslationStatus: systemadminv1.TranslationStatus_TRANSLATION_STATUS_PENDING, SourceHash: translationSourceHash(source)}
	}
	return &systemadminv1.BaseConfigTranslation{Id: row.ID, ConfigId: row.ConfigID, Locale: row.Locale, Field: configTranslationFieldEnum(row.Field), Text: row.Text, TranslationStatus: translationStatusDTO(row.TranslationStatus), SourceHash: row.SourceHash, SourceChanged: row.SourceHash != translationSourceHash(source), TranslationProvider: row.TranslationProvider, TranslatedAt: formatTranslationTime(row.TranslatedAt), ReviewedBy: row.ReviewedBy, ReviewedAt: formatTranslationTime(row.ReviewedAt), CreatedBy: row.CreatedBy, UpdatedBy: row.UpdatedBy, CreatedAt: formatTranslationTime(row.CreatedAt), UpdatedAt: formatTranslationTime(row.UpdatedAt), DeletedAt: int64(row.DeletedAt)}
}

// configTranslationSourceText 返回配置指定翻译字段的中文源文。
func configTranslationSourceText(config *models.BaseConfig, field systemadminv1.BaseConfigTranslationField) (string, error) {
	switch field {
	case systemadminv1.BaseConfigTranslationField_BASE_CONFIG_TRANSLATION_FIELD_NAME:
		return config.Name, nil
	case systemadminv1.BaseConfigTranslationField_BASE_CONFIG_TRANSLATION_FIELD_VALUE:
		if !isTranslatableConfigType(config.Type) {
			return "", errorsx.InvalidArgument("图片、字典和布尔配置值不支持翻译")
		}
		return config.Value, nil
	default:
		return "", errorsx.InvalidArgument("系统配置翻译字段无效")
	}
}

// configTranslationFieldValue 返回数据库中的系统配置翻译字段值。
func configTranslationFieldValue(field systemadminv1.BaseConfigTranslationField) (string, bool) {
	switch field {
	case systemadminv1.BaseConfigTranslationField_BASE_CONFIG_TRANSLATION_FIELD_NAME:
		return "name", true
	case systemadminv1.BaseConfigTranslationField_BASE_CONFIG_TRANSLATION_FIELD_VALUE:
		return "value", true
	default:
		return "", false
	}
}

// configTranslationFieldEnum 返回接口中的系统配置翻译字段枚举。
func configTranslationFieldEnum(value string) systemadminv1.BaseConfigTranslationField {
	if value == "name" {
		return systemadminv1.BaseConfigTranslationField_BASE_CONFIG_TRANSLATION_FIELD_NAME
	}
	if value == "value" {
		return systemadminv1.BaseConfigTranslationField_BASE_CONFIG_TRANSLATION_FIELD_VALUE
	}
	return systemadminv1.BaseConfigTranslationField_BASE_CONFIG_TRANSLATION_FIELD_UNSPECIFIED
}

// isTranslatableConfigType 判断配置值是否属于允许翻译的文本类型。
func isTranslatableConfigType(configType int32) bool {
	return configType == int32(systemcommonv1.BaseConfigType_TEXT) || configType == int32(systemcommonv1.BaseConfigType_RICH_TEXT)
}

// configTranslationKey 生成配置翻译记录的内存索引键。
func configTranslationKey(configID int64, localeValue, fieldValue string) string {
	return fmt.Sprintf("%d:%s:%s", configID, localeValue, fieldValue)
}

// formatTranslationTime 将可空数据库时间转换为接口时间字符串。
func formatTranslationTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.Format(time.DateTime)
}
