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

	"github.com/go-kratos/kratos/v3/log"
	"github.com/liujitcn/gorm-kit/repository"
	"gorm.io/gorm"
)

// BaseDictTranslationCase 负责字典翻译业务。
type BaseDictTranslationCase struct {
	*biz.BaseCase
	translationCase     *BaseTranslationCase
	dictRepo            *data.BaseDictRepository
	dictTranslationRepo *data.BaseDictTranslationRepository
}

// NewBaseDictTranslationCase 创建字典翻译业务实例。
func NewBaseDictTranslationCase(translationCase *BaseTranslationCase, dictRepo *data.BaseDictRepository, dictTranslationRepo *data.BaseDictTranslationRepository) *BaseDictTranslationCase {
	return &BaseDictTranslationCase{
		BaseCase:            translationCase.BaseCase,
		translationCase:     translationCase,
		dictRepo:            dictRepo,
		dictTranslationRepo: dictTranslationRepo,
	}
}

// GenerateDictTranslationDraft 从指定源语言生成字典机器译文并保存为草稿。
func (c *BaseDictTranslationCase) GenerateDictTranslationDraft(ctx context.Context, dictID int64, source, sourceLocale, targetLocale, mainSource string) error {
	locales, err := c.translationCase.enabledEditableLocales(ctx)
	if err != nil {
		return err
	}
	var ok bool
	targetLocale, ok = editableLocale(locales, targetLocale)
	if !ok {
		return errorsx.InvalidArgument("字典机器译文目标语言必须是已启用的非主语言")
	}
	sourceRecord := &dto.TranslationDraftSource{
		ResourceType: systemadminv1.TranslationResourceType_TRANSLATION_RESOURCE_TYPE_DICT,
		ResourceID:   dictID,
	}
	var reviewed bool
	reviewed, err = c.hasReviewedTranslation(ctx, sourceRecord, targetLocale)
	if err != nil {
		return err
	}
	if reviewed {
		return errorsx.Conflict("人工审核译文不允许被机器草稿覆盖")
	}
	startedAt := time.Now()
	var translated string
	translated, err = c.translationCase.TranslateText(ctx, source, sourceLocale, targetLocale)
	if err != nil {
		log.Error("生成字典翻译草稿失败", "resource_id", dictID, "locale", targetLocale, "duration", time.Since(startedAt), "error", err)
		return err
	}
	err = c.saveMachineDraft(ctx, dictID, targetLocale, translated, translationSourceHash(mainSource), time.Now())
	if err != nil {
		return err
	}
	log.Info("生成字典翻译草稿成功", "resource_id", dictID, "locale", targetLocale, "duration", time.Since(startedAt))
	return nil
}

// DictTranslations 批量查询字典维护界面需要的完整翻译状态。
func (c *BaseDictTranslationCase) DictTranslations(ctx context.Context, sources map[int64]string) (map[int64][]*systemadminv1.BaseDictTranslation, error) {
	result := make(map[int64][]*systemadminv1.BaseDictTranslation, len(sources))
	if len(sources) == 0 {
		return result, nil
	}
	locales, err := c.translationCase.enabledEditableLocales(ctx)
	if err != nil {
		return nil, err
	}
	ids := translationSourceIDs(sources)
	query := c.dictTranslationRepo.Query(ctx).BaseDictTranslation
	var rows []*models.BaseDictTranslation
	rows, err = c.dictTranslationRepo.List(ctx, repository.Where(query.DictID.In(ids...)))
	if err != nil {
		return nil, err
	}
	rowMap := make(map[dto.TranslationKey]*models.BaseDictTranslation, len(rows))
	for _, row := range rows {
		rowMap[dto.TranslationKey{ResourceID: row.DictID, Locale: row.Locale}] = row
	}
	for resourceID, source := range sources {
		translations := make([]*systemadminv1.BaseDictTranslation, 0, len(locales))
		for _, localeValue := range locales {
			row := rowMap[dto.TranslationKey{ResourceID: resourceID, Locale: localeValue}]
			translations = append(translations, dictTranslationDTO(row, resourceID, localeValue, source))
		}
		result[resourceID] = translations
	}
	return result, nil
}

// SaveDictTranslations 将人工维护的字典译文保存为已审核状态。
func (c *BaseDictTranslationCase) SaveDictTranslations(ctx context.Context, dictID int64, source string, translations []*systemadminv1.BaseDictTranslation) error {
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
func (c *BaseDictTranslationCase) MarkDictSourceChanged(ctx context.Context, dictID int64, previousSource, currentSource string) error {
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
func (c *BaseDictTranslationCase) DeleteDictTranslations(ctx context.Context, dictIDs []int64) error {
	if len(dictIDs) == 0 {
		return nil
	}
	query := c.dictTranslationRepo.Query(ctx).BaseDictTranslation
	return c.dictTranslationRepo.Delete(ctx, repository.Where(query.DictID.In(dictIDs...)))
}

// ReviewedDictNames 批量查询当前语言已审核的字典名称。
func (c *BaseDictTranslationCase) ReviewedDictNames(ctx context.Context, dictIDs []int64) (map[int64]string, error) {
	result := make(map[int64]string)
	localeValue := coreLocale.FromContext(ctx)
	if len(dictIDs) == 0 {
		return result, nil
	}
	isPrimary, err := c.translationCase.IsPrimaryLocale(ctx, localeValue)
	if err != nil {
		return nil, err
	}
	if isPrimary {
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
func (c *BaseDictTranslationCase) ReviewedDictIDsByName(ctx context.Context, keyword string) ([]int64, error) {
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

// findTranslationDraftSource 按字典编号读取字典名称源文。
func (c *BaseDictTranslationCase) findTranslationDraftSource(ctx context.Context, resourceID int64, field systemadminv1.BaseConfigTranslationField) (*dto.TranslationDraftSource, error) {
	dict, err := c.dictRepo.FindByID(ctx, resourceID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errorsx.ResourceNotFound("待翻译资源不存在").WithCause(err)
		}
		return nil, errorsx.Internal("读取待翻译资源失败").WithCause(err)
	}
	return &dto.TranslationDraftSource{
		ResourceType: systemadminv1.TranslationResourceType_TRANSLATION_RESOURCE_TYPE_DICT,
		ResourceID:   resourceID,
		Field:        field,
		Text:         dict.Name,
	}, nil
}

// hasReviewedTranslation 判断字典目标语言是否已有人工译文。
func (c *BaseDictTranslationCase) hasReviewedTranslation(ctx context.Context, source *dto.TranslationDraftSource, localeValue string) (bool, error) {
	query := c.dictTranslationRepo.Query(ctx).BaseDictTranslation
	row, err := c.dictTranslationRepo.Find(ctx, repository.Where(query.DictID.Eq(source.ResourceID)), repository.Where(query.Locale.Eq(localeValue)))
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return row.TranslationStatus == _const.TRANSLATION_STATUS_REVIEWED, nil
}

// saveMachineDraft 保存字典机器翻译草稿。
func (c *BaseDictTranslationCase) saveMachineDraft(ctx context.Context, dictID int64, localeValue, translated, sourceHash string, translatedAt time.Time) error {
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

// dictTranslationDTO 转换字典翻译维护数据并计算源文变化状态。
func dictTranslationDTO(row *models.BaseDictTranslation, resourceID int64, localeValue, source string) *systemadminv1.BaseDictTranslation {
	if row == nil {
		return &systemadminv1.BaseDictTranslation{DictId: resourceID, Locale: localeValue, TranslationStatus: systemadminv1.TranslationStatus_TRANSLATION_STATUS_PENDING, SourceHash: translationSourceHash(source)}
	}
	return &systemadminv1.BaseDictTranslation{Id: row.ID, DictId: row.DictID, Locale: row.Locale, Name: row.Name, TranslationStatus: translationStatusDTO(row.TranslationStatus), SourceHash: row.SourceHash, TranslationProvider: row.TranslationProvider, TranslatedAt: formatTranslationTime(row.TranslatedAt), ReviewedBy: row.ReviewedBy, ReviewedAt: formatTranslationTime(row.ReviewedAt), CreatedBy: row.CreatedBy, UpdatedBy: row.UpdatedBy, CreatedAt: formatTranslationTime(row.CreatedAt), UpdatedAt: formatTranslationTime(row.UpdatedAt), DeletedAt: uint64(row.DeletedAt), SourceChanged: row.SourceHash != translationSourceHash(source)}
}
