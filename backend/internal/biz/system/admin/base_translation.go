package biz

import (
	"context"
	"errors"
	"sync"

	"github.com/liujitcn/go-utils/translator"
	"github.com/liujitcn/gorm-kit/repository"
	systemadminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/core/pkg/errorsx"
	coreI18n "github.com/liujitcn/kratos-admin/backend/core/pkg/i18n"
	"github.com/liujitcn/kratos-admin/backend/internal/biz"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin/dto"
	_const "github.com/liujitcn/kratos-admin/backend/internal/const"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	translationQueue "github.com/liujitcn/kratos-admin/backend/internal/data/queue"

	"github.com/go-kratos/kratos/v3/log"
	"gorm.io/gorm"
)

// BaseTranslationCase 统一管理所有资源的翻译能力。
type BaseTranslationCase struct {
	*biz.BaseCase
	tx data.Transaction
	*data.BaseTranslationRepository
	languageCase    *BaseLanguageCase
	draftTranslator translator.Translator
	draftMu         sync.Mutex
}

// NewBaseTranslationCase 创建动态翻译业务实例。
func NewBaseTranslationCase(
	baseCase *biz.BaseCase,
	tx data.Transaction,
	baseTranslationRepository *data.BaseTranslationRepository,
	languageCase *BaseLanguageCase,
	draftTranslator translator.Translator,
) *BaseTranslationCase {
	translationCase := &BaseTranslationCase{
		BaseCase:                  baseCase,
		tx:                        tx,
		BaseTranslationRepository: baseTranslationRepository,
		languageCase:              languageCase,
		draftTranslator:           draftTranslator,
	}
	return translationCase
}

// LocaleState 查询动态翻译使用的运行时语言状态。
func (c *BaseTranslationCase) LocaleState(ctx context.Context) (*dto.LocaleState, error) {
	return c.languageCase.LocaleState(ctx)
}

// DraftBaseTranslation 翻译请求中的单个文本，不保存翻译结果。
func (c *BaseTranslationCase) DraftBaseTranslation(ctx context.Context, req *systemadminv1.DraftBaseTranslationRequest) (*systemadminv1.DraftBaseTranslationResponse, error) {
	if c.draftTranslator == nil {
		return nil, errorsx.PermissionDenied("机器翻译草稿功能未启用")
	}
	if req.GetSource() == "" {
		return nil, errorsx.InvalidArgument("待翻译源文不能为空")
	}
	state, err := c.LocaleState(ctx)
	if err != nil {
		return nil, err
	}

	c.draftMu.Lock()
	defer c.draftMu.Unlock()
	locales := state.EditableLocales()
	translations := make([]*systemadminv1.DraftBaseTranslationItem, 0, len(locales))
	for _, locale := range locales {
		translated, translateErr := coreI18n.TranslateProtected(ctx, c.draftTranslator, req.GetSource(), state.Primary, locale)
		if translateErr != nil {
			return nil, errorsx.Internal("生成翻译草稿失败").WithCause(translateErr)
		}
		translations = append(translations, &systemadminv1.DraftBaseTranslationItem{Locale: locale, Translation: translated})
	}
	return &systemadminv1.DraftBaseTranslationResponse{Translations: translations}, nil
}

// UpdateBaseTranslation 优先按 ID 更新，未找到时按目标信息更新或新增翻译记录。
func (c *BaseTranslationCase) UpdateBaseTranslation(ctx context.Context, req *systemadminv1.UpdateBaseTranslationRequest) error {
	state, err := c.LocaleState(ctx)
	if err != nil {
		return err
	}
	var row *models.BaseTranslation
	if req.GetId() > 0 {
		row, err = c.FindByID(ctx, req.GetId())
		if err == nil {
			if !state.IsEditable(row.Locale) {
				return errorsx.InvalidArgument("翻译语言必须是已启用的非主语言")
			}
			row.Name = req.GetName()
			return c.UpdateByID(ctx, row)
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
	}
	if !state.IsEditable(req.GetLocale()) {
		return errorsx.InvalidArgument("翻译语言必须是已启用的非主语言")
	}

	query := c.Query(ctx).BaseTranslation
	opts := make([]repository.QueryOption, 0, 3)
	opts = append(opts, repository.Where(query.TargetType.Eq(int32(req.GetTargetType()))))
	opts = append(opts, repository.Where(query.TargetID.Eq(req.GetTargetId())))
	opts = append(opts, repository.Where(query.Locale.Eq(req.GetLocale())))
	row, err = c.Find(ctx, opts...)
	if err == nil {
		row.Name = req.GetName()
		return c.UpdateByID(ctx, row)
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	return c.Create(ctx, &models.BaseTranslation{
		TargetType: int32(req.GetTargetType()),
		TargetID:   req.GetTargetId(),
		Locale:     req.GetLocale(),
		Name:       req.GetName(),
	})
}

// GetTargetIdsByName 根据当前语言和名称关键字获取资源 ID。
func (c *BaseTranslationCase) GetTargetIdsByName(ctx context.Context, targetType systemadminv1.TranslationTargetType, name string) ([]int64, error) {
	if name == "" {
		return nil, nil
	}

	state, err := c.LocaleState(ctx)
	if err != nil {
		return nil, err
	}
	if state.IsCurrentPrimary() || !state.IsEnabled(state.Current) {
		return nil, nil
	}
	localeValue := state.Current

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
	state, err := c.LocaleState(ctx)
	if err != nil {
		return nil, err
	}
	if locale == state.Primary || !state.IsEnabled(locale) {
		return result, nil
	}
	query := c.Query(ctx).BaseTranslation
	opts := make([]repository.QueryOption, 0, 3)
	opts = append(opts, repository.Where(query.TargetType.Eq(int32(targetType))))
	opts = append(opts, repository.Where(query.Locale.Eq(locale)))
	opts = append(opts, repository.Where(query.TargetID.In(targetIds...)))

	var list []*models.BaseTranslation
	list, err = c.List(ctx, opts...)
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

// SaveBaseTranslation 保存主语言源文对应的翻译信息，并为缺失译文投递机器翻译任务。
func (c *BaseTranslationCase) SaveBaseTranslation(ctx context.Context, targetType systemadminv1.TranslationTargetType, targetId int64, primaryText string, translations []*systemadminv1.BaseTranslation, updateMain func(context.Context, string) error) error {
	var err error
	var state *dto.LocaleState
	state, err = c.LocaleState(ctx)
	if err != nil {
		return err
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

		values := make(map[string]string, len(translations))
		seen := make(map[string]struct{}, len(translations))
		for _, translation := range translations {
			if translation.GetTargetType() != targetType {
				return errorsx.InvalidArgument("翻译目标类型无效")
			}
			localeValue := translation.GetLocale()
			if !state.IsEditable(localeValue) {
				return errorsx.InvalidArgument("翻译语言必须是已启用的非主语言")
			}
			if _, duplicated := seen[localeValue]; duplicated {
				return errorsx.Conflict("同一资源语言不能重复")
			}
			seen[localeValue] = struct{}{}
			values[localeValue] = translation.GetName()
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
		if updateMain != nil {
			if err = updateMain(txCtx, primaryText); err != nil {
				return err
			}
		}
		return nil
	}
	if updateMain == nil {
		err = save(ctx)
	} else {
		err = c.tx.Transaction(ctx, save)
	}
	if err != nil || c.draftTranslator == nil || targetId <= 0 {
		return err
	}
	translationNames := make(map[string]string, len(translations))
	for _, translation := range translations {
		translationNames[translation.GetLocale()] = translation.GetName()
	}
	for _, locale := range state.EditableLocales() {
		if translationNames[locale] != "" {
			continue
		}
		if ok := translationQueue.AddQueue(_const.TRANSLATION, &dto.TranslationQueueMessage{
			TargetType:   targetType,
			TargetID:     targetId,
			SourceText:   primaryText,
			SourceLocale: state.Primary,
			TargetLocale: locale,
		}); !ok {
			log.Warn("投递机器翻译队列失败", "target_type", targetType.String(), "target_id", targetId, "source_locale", state.Primary, "target_locale", locale)
		}
	}
	return nil
}

// SaveGeneratedTranslations 保存代码生成器提供的非主语言译文，不覆盖已有非空内容。
func (c *BaseTranslationCase) SaveGeneratedTranslations(ctx context.Context, targetType systemadminv1.TranslationTargetType, targetID int64, translations map[string]string) error {
	if targetID <= 0 || len(translations) == 0 {
		return nil
	}
	state, err := c.LocaleState(ctx)
	if err != nil {
		return err
	}
	query := c.Query(ctx).BaseTranslation
	var rows []*models.BaseTranslation
	rows, err = c.List(ctx,
		repository.Where(query.TargetType.Eq(int32(targetType))),
		repository.Where(query.TargetID.Eq(targetID)),
	)
	if err != nil {
		return err
	}
	existing := make(map[string]*models.BaseTranslation, len(rows))
	for _, row := range rows {
		existing[row.Locale] = row
	}
	for locale, text := range translations {
		if text == "" || !state.IsEditable(locale) {
			continue
		}
		row := existing[locale]
		if row != nil && row.Name != "" {
			continue
		}
		if row == nil {
			if err = c.Create(ctx, &models.BaseTranslation{TargetType: int32(targetType), TargetID: targetID, Locale: locale, Name: text}); err != nil {
				return err
			}
			continue
		}
		row.Name = text
		if err = c.UpdateByID(ctx, row); err != nil {
			return err
		}
	}
	return nil
}

// DeleteBaseTranslation 删除翻译信息。
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
