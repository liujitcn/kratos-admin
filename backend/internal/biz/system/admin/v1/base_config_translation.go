package biz

import (
	"context"
	"errors"
	"fmt"
	"time"

	systemadminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	systemcommonv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/common/v1"
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

// BaseConfigTranslationCase 负责系统配置翻译业务。
type BaseConfigTranslationCase struct {
	*biz.BaseCase
	translationCase       *BaseTranslationCase
	configRepo            *data.BaseConfigRepository
	configTranslationRepo *data.BaseConfigTranslationRepository
}

// NewBaseConfigTranslationCase 创建系统配置翻译业务实例。
func NewBaseConfigTranslationCase(translationCase *BaseTranslationCase, configRepo *data.BaseConfigRepository, configTranslationRepo *data.BaseConfigTranslationRepository) *BaseConfigTranslationCase {
	return &BaseConfigTranslationCase{
		BaseCase:              translationCase.BaseCase,
		translationCase:       translationCase,
		configRepo:            configRepo,
		configTranslationRepo: configTranslationRepo,
	}
}

// ReviewedConfigValues 批量查询当前语言已审核的文本配置值。
func (c *BaseConfigTranslationCase) ReviewedConfigValues(ctx context.Context, configIDs []int64) (map[int64]string, error) {
	result := make(map[int64]string)
	localeValue := coreLocale.FromContext(ctx)
	if len(configIDs) == 0 {
		return result, nil
	}
	isPrimary, err := c.translationCase.IsPrimaryLocale(ctx, localeValue)
	if err != nil {
		return nil, err
	}
	if isPrimary {
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
func (c *BaseConfigTranslationCase) ReviewedConfigNames(ctx context.Context, configIDs []int64) (map[int64]string, error) {
	result := make(map[int64]string)
	localeValue := coreLocale.FromContext(ctx)
	if len(configIDs) == 0 {
		return result, nil
	}
	isPrimary, err := c.translationCase.IsPrimaryLocale(ctx, localeValue)
	if err != nil {
		return nil, err
	}
	if isPrimary {
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
func (c *BaseConfigTranslationCase) ReviewedConfigIDsByName(ctx context.Context, keyword string) ([]int64, error) {
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
func (c *BaseConfigTranslationCase) ConfigTranslations(ctx context.Context, sources map[int64]dto.ConfigTranslationSource) (map[int64][]*systemadminv1.BaseConfigTranslation, error) {
	result := make(map[int64][]*systemadminv1.BaseConfigTranslation, len(sources))
	if len(sources) == 0 {
		return result, nil
	}
	locales, err := c.translationCase.enabledEditableLocales(ctx)
	if err != nil {
		return nil, err
	}
	ids := make([]int64, 0, len(sources))
	for configID := range sources {
		ids = append(ids, configID)
	}
	query := c.configTranslationRepo.Query(ctx).BaseConfigTranslation
	var rows []*models.BaseConfigTranslation
	rows, err = c.configTranslationRepo.List(ctx, repository.Where(query.ConfigID.In(ids...)))
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
		translations := make([]*systemadminv1.BaseConfigTranslation, 0, len(locales)*len(fields))
		for _, field := range fields {
			fieldValue, _ := configTranslationFieldValue(field)
			sourceText := source.Name
			if field == systemadminv1.BaseConfigTranslationField_BASE_CONFIG_TRANSLATION_FIELD_VALUE {
				sourceText = source.Value
			}
			for _, localeValue := range locales {
				row := rowMap[configTranslationKey(configID, localeValue, fieldValue)]
				translations = append(translations, configTranslationDTO(row, configID, localeValue, field, sourceText))
			}
		}
		result[configID] = translations
	}
	return result, nil
}

// SaveConfigTranslations 将人工维护的系统配置译文保存为已审核状态。
func (c *BaseConfigTranslationCase) SaveConfigTranslations(ctx context.Context, configID int64, source dto.ConfigTranslationSource, translations []*systemadminv1.BaseConfigTranslation) error {
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
		localeValue, ok := editableLocale(locales, translation.GetLocale())
		if !ok {
			return errorsx.InvalidArgument("系统配置翻译语言必须是已启用的非主语言")
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
func (c *BaseConfigTranslationCase) MarkConfigSourceChanged(ctx context.Context, configID int64, previousSource, currentSource dto.ConfigTranslationSource) error {
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
func (c *BaseConfigTranslationCase) DeleteConfigTranslations(ctx context.Context, configIDs []int64) error {
	if len(configIDs) == 0 {
		return nil
	}
	query := c.configTranslationRepo.Query(ctx).BaseConfigTranslation
	return c.configTranslationRepo.Delete(ctx, repository.Where(query.ConfigID.In(configIDs...)))
}

// findTranslationDraftSource 按系统配置编号读取指定字段的源文。
func (c *BaseConfigTranslationCase) findTranslationDraftSource(ctx context.Context, resourceID int64, field systemadminv1.BaseConfigTranslationField) (*dto.TranslationDraftSource, error) {
	config, err := c.configRepo.FindByID(ctx, resourceID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errorsx.ResourceNotFound("待翻译资源不存在").WithCause(err)
		}
		return nil, errorsx.Internal("读取待翻译资源失败").WithCause(err)
	}
	text, err := configTranslationSourceText(config, field)
	if err != nil {
		return nil, errorsx.Internal("读取待翻译资源失败").WithCause(err)
	}
	return &dto.TranslationDraftSource{
		ResourceType: systemadminv1.TranslationResourceType_TRANSLATION_RESOURCE_TYPE_CONFIG,
		ResourceID:   resourceID,
		Field:        field,
		Text:         text,
	}, nil
}

// hasReviewedTranslation 判断系统配置目标语言是否已有人工译文。
func (c *BaseConfigTranslationCase) hasReviewedTranslation(ctx context.Context, source *dto.TranslationDraftSource, localeValue string) (bool, error) {
	fieldValue, ok := configTranslationFieldValue(source.Field)
	if !ok {
		return false, errorsx.InvalidArgument("系统配置翻译字段无效")
	}
	query := c.configTranslationRepo.Query(ctx).BaseConfigTranslation
	row, err := c.configTranslationRepo.Find(ctx, repository.Where(query.ConfigID.Eq(source.ResourceID)), repository.Where(query.Locale.Eq(localeValue)), repository.Where(query.Field.Eq(fieldValue)))
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return row.TranslationStatus == _const.TRANSLATION_STATUS_REVIEWED, nil
}

// saveMachineDraft 保存系统配置机器翻译草稿。
func (c *BaseConfigTranslationCase) saveMachineDraft(ctx context.Context, configID int64, field systemadminv1.BaseConfigTranslationField, localeValue, translated, sourceHash string, translatedAt time.Time) error {
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

// configTranslationDTO 转换系统配置翻译维护数据并计算源文变化状态。
func configTranslationDTO(row *models.BaseConfigTranslation, resourceID int64, localeValue string, field systemadminv1.BaseConfigTranslationField, source string) *systemadminv1.BaseConfigTranslation {
	if row == nil {
		return &systemadminv1.BaseConfigTranslation{ConfigId: resourceID, Locale: localeValue, Field: field, TranslationStatus: systemadminv1.TranslationStatus_TRANSLATION_STATUS_PENDING, SourceHash: translationSourceHash(source)}
	}
	return &systemadminv1.BaseConfigTranslation{Id: row.ID, ConfigId: row.ConfigID, Locale: row.Locale, Field: configTranslationFieldEnum(row.Field), Text: row.Text, TranslationStatus: translationStatusDTO(row.TranslationStatus), SourceHash: row.SourceHash, SourceChanged: row.SourceHash != translationSourceHash(source), TranslationProvider: row.TranslationProvider, TranslatedAt: formatTranslationTime(row.TranslatedAt), ReviewedBy: row.ReviewedBy, ReviewedAt: formatTranslationTime(row.ReviewedAt), CreatedBy: row.CreatedBy, UpdatedBy: row.UpdatedBy, CreatedAt: formatTranslationTime(row.CreatedAt), UpdatedAt: formatTranslationTime(row.UpdatedAt), DeletedAt: int64(row.DeletedAt)}
}

// configTranslationSourceText 返回配置指定翻译字段的主语言源文。
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
