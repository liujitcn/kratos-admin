package biz

import (
	"context"
	"errors"
	"time"

	systemadminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/core/pkg/errorsx"
	coreLocale "github.com/liujitcn/kratos-admin/backend/core/pkg/locale"
	"github.com/liujitcn/kratos-admin/backend/internal/biz"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin/v1/dto"
	_const "github.com/liujitcn/kratos-admin/backend/internal/const"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"

	"github.com/liujitcn/gorm-kit/repository"
	"gorm.io/gorm"
)

// BaseDictItemTranslationCase 负责字典项翻译业务。
type BaseDictItemTranslationCase struct {
	*biz.BaseCase
	translationCase         *BaseTranslationCase
	dictItemRepo            *data.BaseDictItemRepository
	dictItemTranslationRepo *data.BaseDictItemTranslationRepository
}

// NewBaseDictItemTranslationCase 创建字典项翻译业务实例。
func NewBaseDictItemTranslationCase(translationCase *BaseTranslationCase, dictItemRepo *data.BaseDictItemRepository, dictItemTranslationRepo *data.BaseDictItemTranslationRepository) *BaseDictItemTranslationCase {
	return &BaseDictItemTranslationCase{
		BaseCase:                translationCase.BaseCase,
		translationCase:         translationCase,
		dictItemRepo:            dictItemRepo,
		dictItemTranslationRepo: dictItemTranslationRepo,
	}
}

// ReviewedDictItemLabels 批量查询当前语言已审核的字典项标签。
func (c *BaseDictItemTranslationCase) ReviewedDictItemLabels(ctx context.Context, dictItemIDs []int64) (map[int64]string, error) {
	result := make(map[int64]string)
	localeValue := coreLocale.FromContext(ctx)
	if len(dictItemIDs) == 0 {
		return result, nil
	}
	isPrimary, err := c.translationCase.IsPrimaryLocale(ctx, localeValue)
	if err != nil {
		return nil, err
	}
	if isPrimary {
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
func (c *BaseDictItemTranslationCase) ReviewedDictItemIDsByLabel(ctx context.Context, keyword string) ([]int64, error) {
	localeValue := coreLocale.FromContext(ctx)
	if keyword == "" {
		return nil, nil
	}
	isPrimary, err := c.translationCase.IsPrimaryLocale(ctx, localeValue)
	if err != nil {
		return nil, err
	}
	if isPrimary {
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
func (c *BaseDictItemTranslationCase) DictItemTranslations(ctx context.Context, sources map[int64]string) (map[int64][]*systemadminv1.BaseDictItemTranslation, error) {
	result := make(map[int64][]*systemadminv1.BaseDictItemTranslation, len(sources))
	if len(sources) == 0 {
		return result, nil
	}
	locales, err := c.translationCase.enabledEditableLocales(ctx)
	if err != nil {
		return nil, err
	}
	ids := translationSourceIDs(sources)
	query := c.dictItemTranslationRepo.Query(ctx).BaseDictItemTranslation
	var rows []*models.BaseDictItemTranslation
	rows, err = c.dictItemTranslationRepo.List(ctx, repository.Where(query.DictItemID.In(ids...)))
	if err != nil {
		return nil, err
	}
	rowMap := make(map[dto.TranslationKey]*models.BaseDictItemTranslation, len(rows))
	for _, row := range rows {
		rowMap[dto.TranslationKey{ResourceID: row.DictItemID, Locale: row.Locale}] = row
	}
	for resourceID, source := range sources {
		translations := make([]*systemadminv1.BaseDictItemTranslation, 0, len(locales))
		for _, localeValue := range locales {
			row := rowMap[dto.TranslationKey{ResourceID: resourceID, Locale: localeValue}]
			translations = append(translations, dictItemTranslationDTO(row, resourceID, localeValue, source))
		}
		result[resourceID] = translations
	}
	return result, nil
}

// SaveDictItemTranslations 将人工维护的字典项译文保存为已审核状态。
func (c *BaseDictItemTranslationCase) SaveDictItemTranslations(ctx context.Context, dictItemID int64, source string, translations []*systemadminv1.BaseDictItemTranslation) error {
	if translations == nil {
		return nil
	}
	authInfo, err := c.translationCase.GetAuthInfo(ctx)
	if err != nil {
		return err
	}
	var locales []string
	locales, err = c.translationCase.enabledEditableLocales(ctx)
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
		localeValue, ok := editableLocale(locales, translation.GetLocale())
		if !ok {
			return errorsx.InvalidArgument("字典项翻译语言必须是已启用的非主语言")
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
func (c *BaseDictItemTranslationCase) MarkDictItemSourceChanged(ctx context.Context, dictItemID int64, previousSource, currentSource string) error {
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
func (c *BaseDictItemTranslationCase) DeleteDictItemTranslations(ctx context.Context, dictItemIDs []int64) error {
	if len(dictItemIDs) == 0 {
		return nil
	}
	query := c.dictItemTranslationRepo.Query(ctx).BaseDictItemTranslation
	return c.dictItemTranslationRepo.Delete(ctx, repository.Where(query.DictItemID.In(dictItemIDs...)))
}

// findTranslationDraftSource 按字典项编号读取字典项标签源文。
func (c *BaseDictItemTranslationCase) findTranslationDraftSource(ctx context.Context, resourceID int64, field systemadminv1.BaseConfigTranslationField) (*dto.TranslationDraftSource, error) {
	item, err := c.dictItemRepo.FindByID(ctx, resourceID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errorsx.ResourceNotFound("待翻译资源不存在").WithCause(err)
		}
		return nil, errorsx.Internal("读取待翻译资源失败").WithCause(err)
	}
	return &dto.TranslationDraftSource{
		ResourceType: systemadminv1.TranslationResourceType_TRANSLATION_RESOURCE_TYPE_DICT_ITEM,
		ResourceID:   resourceID,
		Field:        field,
		Text:         item.Label,
	}, nil
}

// hasReviewedTranslation 判断字典项目标语言是否已有人工译文。
func (c *BaseDictItemTranslationCase) hasReviewedTranslation(ctx context.Context, source *dto.TranslationDraftSource, localeValue string) (bool, error) {
	query := c.dictItemTranslationRepo.Query(ctx).BaseDictItemTranslation
	row, err := c.dictItemTranslationRepo.Find(ctx, repository.Where(query.DictItemID.Eq(source.ResourceID)), repository.Where(query.Locale.Eq(localeValue)))
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return row.TranslationStatus == _const.TRANSLATION_STATUS_REVIEWED, nil
}

// saveMachineDraft 保存字典项机器翻译草稿。
func (c *BaseDictItemTranslationCase) saveMachineDraft(ctx context.Context, dictItemID int64, localeValue, translated, sourceHash string, translatedAt time.Time) error {
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

// dictItemTranslationDTO 转换字典项翻译维护数据并计算源文变化状态。
func dictItemTranslationDTO(row *models.BaseDictItemTranslation, resourceID int64, localeValue, source string) *systemadminv1.BaseDictItemTranslation {
	if row == nil {
		return &systemadminv1.BaseDictItemTranslation{DictItemId: resourceID, Locale: localeValue, TranslationStatus: systemadminv1.TranslationStatus_TRANSLATION_STATUS_PENDING, SourceHash: translationSourceHash(source)}
	}
	return &systemadminv1.BaseDictItemTranslation{Id: row.ID, DictItemId: row.DictItemID, Locale: row.Locale, Label: row.Label, TranslationStatus: translationStatusDTO(row.TranslationStatus), SourceHash: row.SourceHash, TranslationProvider: row.TranslationProvider, TranslatedAt: formatTranslationTime(row.TranslatedAt), ReviewedBy: row.ReviewedBy, ReviewedAt: formatTranslationTime(row.ReviewedAt), CreatedBy: row.CreatedBy, UpdatedBy: row.UpdatedBy, CreatedAt: formatTranslationTime(row.CreatedAt), UpdatedAt: formatTranslationTime(row.UpdatedAt), DeletedAt: uint64(row.DeletedAt), SourceChanged: row.SourceHash != translationSourceHash(source)}
}
