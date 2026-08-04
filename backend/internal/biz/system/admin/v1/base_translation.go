package biz

import (
	"context"
	"sync"

	"github.com/liujitcn/go-utils/translator"
	"github.com/liujitcn/gorm-kit/repository"
	systemadminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/core/pkg/errorsx"
	coreLocale "github.com/liujitcn/kratos-admin/backend/core/pkg/locale"
	"github.com/liujitcn/kratos-admin/backend/internal/biz"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin/v1/dto"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
)

const translationDraftMaxBytes = 2000

// BaseTranslationCase 统一管理所有资源的翻译能力。
type BaseTranslationCase struct {
	*biz.BaseCase
	*data.BaseTranslationRepository
	languageCase    *BaseLanguageCase
	draftTranslator translator.Translator
	draftMu         sync.Mutex
}

// NewBaseTranslationCase 创建动态翻译业务实例。
func NewBaseTranslationCase(
	baseCase *biz.BaseCase,
	baseTranslationRepository *data.BaseTranslationRepository,
	languageCase *BaseLanguageCase,
	draftTranslator translator.Translator,
) *BaseTranslationCase {
	translationCase := &BaseTranslationCase{
		BaseCase:                  baseCase,
		BaseTranslationRepository: baseTranslationRepository,
		languageCase:              languageCase,
		draftTranslator:           draftTranslator,
	}
	return translationCase
}

// DraftBaseTranslation 翻译请求中的单个文本，不保存翻译结果。
func (c *BaseTranslationCase) DraftBaseTranslation(ctx context.Context, req *systemadminv1.DraftBaseTranslationRequest) (*systemadminv1.DraftBaseTranslationResponse, error) {
	return nil, nil
}

// UpdateBaseTranslation 修改或新增单个翻译信息；文本为空时补充机器译文。
func (c *BaseTranslationCase) UpdateBaseTranslation(ctx context.Context, req *systemadminv1.UpdateBaseTranslationRequest) error {
	if req.GetId() > 0 {
		return c.UpdateByID(ctx, &models.BaseTranslation{ID: req.GetId(), Name: req.GetName()})
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

	localeValue := coreLocale.FromContext(ctx)
	locales, err := c.languageCase.Locales(ctx)
	if err != nil {
		return nil, err
	}
	currentLocaleFound := false
	for _, locale := range locales {
		if !locale.IsCurrent {
			continue
		}
		currentLocaleFound = true
		if locale.IsPrimary {
			return nil, nil
		}
		break
	}
	if !currentLocaleFound && len(locales) == 0 && localeValue == coreLocale.Default {
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

// SaveBaseTranslation 保存翻译信息，当前语言为非主语言时通过回调更新主表信息。
func (c *BaseTranslationCase) SaveBaseTranslation(ctx context.Context, targetType systemadminv1.TranslationTargetType, targetId int64, currentText string, translations []*systemadminv1.BaseTranslation, updateMain func(context.Context, string) error) error {
	var err error
	var locales []*dto.LocaleInfo
	locales, err = c.languageCase.Locales(ctx)
	if err != nil {
		return err
	}
	currentLocale := coreLocale.FromContext(ctx)
	currentLocaleFound := false
	isPrimary := len(locales) == 0 && currentLocale == coreLocale.Default
	editableLocales := make(map[string]struct{}, len(locales))
	for _, locale := range locales {
		if locale.IsCurrent {
			currentLocale = locale.Locale
			currentLocaleFound = true
			isPrimary = locale.IsPrimary
		}
		if !locale.IsPrimary {
			editableLocales[locale.Locale] = struct{}{}
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
		return save(ctx)
	}
	return c.BaseTranslationRepository.Transaction(ctx, save)
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
