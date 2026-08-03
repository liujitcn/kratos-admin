package biz

import (
	"context"
	"encoding/json"
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

// BaseMenuTranslationCase 负责菜单翻译业务。
type BaseMenuTranslationCase struct {
	*biz.BaseCase
	translationCase     *BaseTranslationCase
	menuRepo            *data.BaseMenuRepository
	menuTranslationRepo *data.BaseMenuTranslationRepository
}

// NewBaseMenuTranslationCase 创建菜单翻译业务实例。
func NewBaseMenuTranslationCase(translationCase *BaseTranslationCase, menuRepo *data.BaseMenuRepository, menuTranslationRepo *data.BaseMenuTranslationRepository) *BaseMenuTranslationCase {
	return &BaseMenuTranslationCase{
		BaseCase:            translationCase.BaseCase,
		translationCase:     translationCase,
		menuRepo:            menuRepo,
		menuTranslationRepo: menuTranslationRepo,
	}
}

// ReviewedMenuTitles 批量查询当前语言已审核的菜单标题。
func (c *BaseMenuTranslationCase) ReviewedMenuTitles(ctx context.Context, menuIDs []int64) (map[int64]string, error) {
	result := make(map[int64]string)
	localeValue := coreLocale.FromContext(ctx)
	if len(menuIDs) == 0 {
		return result, nil
	}
	isPrimary, err := c.translationCase.IsPrimaryLocale(ctx, localeValue)
	if err != nil {
		return nil, err
	}
	if isPrimary {
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
func (c *BaseMenuTranslationCase) MenuTranslations(ctx context.Context, sources map[int64]string) (map[int64][]*systemadminv1.BaseMenuTranslation, error) {
	result := make(map[int64][]*systemadminv1.BaseMenuTranslation, len(sources))
	if len(sources) == 0 {
		return result, nil
	}
	locales, err := c.translationCase.enabledEditableLocales(ctx)
	if err != nil {
		return nil, err
	}
	ids := translationSourceIDs(sources)
	query := c.menuTranslationRepo.Query(ctx).BaseMenuTranslation
	var rows []*models.BaseMenuTranslation
	rows, err = c.menuTranslationRepo.List(ctx, repository.Where(query.MenuID.In(ids...)))
	if err != nil {
		return nil, err
	}
	rowMap := make(map[dto.TranslationKey]*models.BaseMenuTranslation, len(rows))
	for _, row := range rows {
		rowMap[dto.TranslationKey{ResourceID: row.MenuID, Locale: row.Locale}] = row
	}
	for resourceID, source := range sources {
		translations := make([]*systemadminv1.BaseMenuTranslation, 0, len(locales))
		for _, localeValue := range locales {
			row := rowMap[dto.TranslationKey{ResourceID: resourceID, Locale: localeValue}]
			translations = append(translations, menuTranslationDTO(row, resourceID, localeValue, source))
		}
		result[resourceID] = translations
	}
	return result, nil
}

// SaveMenuTranslations 将人工维护的菜单译文保存为已审核状态。
func (c *BaseMenuTranslationCase) SaveMenuTranslations(ctx context.Context, menuID int64, source string, translations []*systemadminv1.BaseMenuTranslation) error {
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
		localeValue, ok := editableLocale(locales, translation.GetLocale())
		if !ok {
			return errorsx.InvalidArgument("菜单翻译语言必须是已启用的非主语言")
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
func (c *BaseMenuTranslationCase) SaveGeneratedMenuTranslations(ctx context.Context, menuID int64, source string, translations map[string]string) error {
	if len(translations) == 0 {
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
	for _, localeValue := range locales {
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
func (c *BaseMenuTranslationCase) MarkMenuSourceChanged(ctx context.Context, menuID int64, previousSource, currentSource string) error {
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
func (c *BaseMenuTranslationCase) DeleteMenuTranslations(ctx context.Context, menuIDs []int64) error {
	if len(menuIDs) == 0 {
		return nil
	}
	query := c.menuTranslationRepo.Query(ctx).BaseMenuTranslation
	return c.menuTranslationRepo.Delete(ctx, repository.Where(query.MenuID.In(menuIDs...)))
}

// findTranslationDraftSource 按菜单编号读取菜单标题源文。
func (c *BaseMenuTranslationCase) findTranslationDraftSource(ctx context.Context, resourceID int64, field systemadminv1.BaseConfigTranslationField) (*dto.TranslationDraftSource, error) {
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
		ResourceType: systemadminv1.TranslationResourceType_TRANSLATION_RESOURCE_TYPE_MENU,
		ResourceID:   resourceID,
		Field:        field,
		Text:         metadata.Title,
	}, nil
}

// hasReviewedTranslation 判断菜单目标语言是否已有人工译文。
func (c *BaseMenuTranslationCase) hasReviewedTranslation(ctx context.Context, source *dto.TranslationDraftSource, localeValue string) (bool, error) {
	query := c.menuTranslationRepo.Query(ctx).BaseMenuTranslation
	row, err := c.menuTranslationRepo.Find(ctx, repository.Where(query.MenuID.Eq(source.ResourceID)), repository.Where(query.Locale.Eq(localeValue)))
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return row.TranslationStatus == _const.TRANSLATION_STATUS_REVIEWED, nil
}

// saveMachineDraft 保存菜单机器翻译草稿。
func (c *BaseMenuTranslationCase) saveMachineDraft(ctx context.Context, menuID int64, localeValue, translated, sourceHash string, translatedAt time.Time) error {
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

// menuTranslationDTO 转换菜单翻译维护数据并计算源文变化状态。
func menuTranslationDTO(row *models.BaseMenuTranslation, resourceID int64, localeValue, source string) *systemadminv1.BaseMenuTranslation {
	if row == nil {
		return &systemadminv1.BaseMenuTranslation{MenuId: resourceID, Locale: localeValue, TranslationStatus: systemadminv1.TranslationStatus_TRANSLATION_STATUS_PENDING, SourceHash: translationSourceHash(source)}
	}
	return &systemadminv1.BaseMenuTranslation{Id: row.ID, MenuId: row.MenuID, Locale: row.Locale, Title: row.Title, TranslationStatus: translationStatusDTO(row.TranslationStatus), SourceHash: row.SourceHash, TranslationProvider: row.TranslationProvider, TranslatedAt: formatTranslationTime(row.TranslatedAt), ReviewedBy: row.ReviewedBy, ReviewedAt: formatTranslationTime(row.ReviewedAt), CreatedBy: row.CreatedBy, UpdatedBy: row.UpdatedBy, CreatedAt: formatTranslationTime(row.CreatedAt), UpdatedAt: formatTranslationTime(row.UpdatedAt), DeletedAt: uint64(row.DeletedAt), SourceChanged: row.SourceHash != translationSourceHash(source)}
}
