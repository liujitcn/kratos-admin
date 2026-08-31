package biz

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"

	"github.com/liujitcn/gorm-kit/repository"
	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin/dto"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	"github.com/liujitcn/kratos-core/biz"
	"github.com/liujitcn/kratos-core/errorsx"

	"gorm.io/gorm"
)

var (
	protectedI18nTextPattern  = regexp.MustCompile("(?s)```.*?```|`[^`]+`|\\{\\{[^{}]+\\}\\}|\\$\\{[^{}]+\\}|\\{[A-Za-z_][A-Za-z0-9_.-]*\\}|%[sdv]|</?[^>]+>|https?://[^\\s<>()]+|[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\\.[A-Za-z]{2,}|/(?:api|events|mcp|v[0-9]+)/[A-Za-z0-9_./:{}-]+|(?i:kratos-admin)")
	protectedI18nTokenPattern = regexp.MustCompile(`__KRATOS_I18N_TOKEN_[0-9]{3}__`)
)

// BaseI18nCase 统一管理所有资源的翻译能力。
type BaseI18nCase struct {
	*biz.BaseCase
	tx data.Transaction
	*data.BaseI18NRepository
	languageCase *BaseLanguageCase
	draftMu      sync.Mutex
}

// NewBaseI18nCase 创建动态翻译业务实例。
func NewBaseI18nCase(
	baseCase *biz.BaseCase,
	tx data.Transaction,
	baseI18nRepository *data.BaseI18NRepository,
	languageCase *BaseLanguageCase,
) *BaseI18nCase {
	i18nCase := &BaseI18nCase{
		BaseCase:           baseCase,
		tx:                 tx,
		BaseI18NRepository: baseI18nRepository,
		languageCase:       languageCase,
	}
	return i18nCase
}

// LocaleState 查询动态翻译使用的运行时语言状态。
func (c *BaseI18nCase) LocaleState(ctx context.Context) (*dto.LocaleState, error) {
	return c.languageCase.LocaleState(ctx)
}

// DraftBaseI18n 翻译请求中的单个文本，不保存翻译结果。
func (c *BaseI18nCase) DraftBaseI18n(ctx context.Context, req *adminv1.DraftBaseI18nRequest) (*adminv1.DraftBaseI18nResponse, error) {
	translator := c.Translator
	if translator == nil {
		return nil, errorsx.PermissionDenied("机器翻译草稿功能未启用")
	}
	if req.GetSource() == "" {
		return nil, errorsx.InvalidArgument("待翻译源文不能为空")
	}
	state, err := c.LocaleState(ctx)
	if err != nil {
		return nil, err
	}
	locales := state.EditableLocales()
	if locale := req.GetLocale(); locale != "" {
		if !state.IsEditable(locale) {
			return nil, errorsx.InvalidArgument("翻译语言必须是已启用的非主语言")
		}
		locales = []string{locale}
	}

	c.draftMu.Lock()
	defer c.draftMu.Unlock()
	i18ns := make([]*adminv1.DraftBaseI18nItem, 0, len(locales))
	for _, locale := range locales {
		var translated string
		translated, err = c.TranslateText(ctx, req.GetSource(), state.Primary, locale)
		if err != nil {
			return nil, errorsx.Internal("生成翻译草稿失败").WithCause(err)
		}
		i18ns = append(i18ns, &adminv1.DraftBaseI18nItem{Locale: locale, I18n: translated})
	}
	return &adminv1.DraftBaseI18nResponse{I18ns: i18ns}, nil
}

// TranslateText 使用 SDK 翻译器生成译文，并保护代码、占位符和 URL 等结构化片段。
func (c *BaseI18nCase) TranslateText(ctx context.Context, source, sourceLocale, targetLocale string) (string, error) {
	translator := c.Translator
	if translator == nil {
		return "", errorsx.PermissionDenied("机器翻译功能未启用")
	}
	protectedSource, values := protectI18nText(source)
	translated, err := translator.Translate(ctx, protectedSource, sourceLocale, targetLocale)
	if err != nil {
		return "", fmt.Errorf("生成翻译草稿: %w", err)
	}
	for index, value := range values {
		token := protectedI18nToken(index)
		if strings.Count(translated, token) != 1 {
			return "", fmt.Errorf("翻译草稿哨兵 %s 数量不一致", token)
		}
		translated = strings.Replace(translated, token, value, 1)
	}
	if protectedI18nTokenPattern.MatchString(translated) {
		return "", fmt.Errorf("翻译草稿包含未恢复哨兵")
	}
	return translated, nil
}

// UpdateBaseI18n 优先按 ID 更新，未找到时按目标信息更新或新增翻译记录。
func (c *BaseI18nCase) UpdateBaseI18n(ctx context.Context, req *adminv1.UpdateBaseI18nRequest) error {
	state, err := c.LocaleState(ctx)
	if err != nil {
		return err
	}
	var row *models.BaseI18N
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

	query := c.Query(ctx).BaseI18N
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
	return c.Create(ctx, &models.BaseI18N{
		TargetType: int32(req.GetTargetType()),
		TargetID:   req.GetTargetId(),
		Locale:     req.GetLocale(),
		Name:       req.GetName(),
	})
}

// GetTargetIdsByName 根据当前语言和名称关键字获取资源 ID。
func (c *BaseI18nCase) GetTargetIdsByName(ctx context.Context, targetType adminv1.I18nTargetType, name string) ([]int64, error) {
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

	query := c.Query(ctx).BaseI18N
	var rows []*models.BaseI18N
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

// GetBaseI18nMapByTargetType 根据类型查询翻译信息。
func (c *BaseI18nCase) GetBaseI18nMapByTargetType(ctx context.Context, targetType adminv1.I18nTargetType, targetIds []int64) (map[int64][]*adminv1.BaseI18n, error) {
	result := make(map[int64][]*adminv1.BaseI18n, len(targetIds))
	if len(targetIds) == 0 {
		return result, nil
	}
	query := c.Query(ctx).BaseI18N
	opts := make([]repository.QueryOption, 0, 2)
	opts = append(opts, repository.Where(query.TargetType.Eq(int32(targetType))))
	opts = append(opts, repository.Where(query.TargetID.In(targetIds...)))

	list, err := c.List(ctx, opts...)
	if err != nil {
		return nil, err
	}
	for _, item := range list {
		result[item.TargetID] = append(result[item.TargetID], &adminv1.BaseI18n{
			Id:         item.ID,
			TargetType: adminv1.I18nTargetType(item.TargetType),
			TargetId:   item.TargetID,
			Locale:     item.Locale,
			Name:       item.Name,
		})
	}
	return result, nil
}

// GetBaseI18nNameMapByLocale 根据语言返回资源名称译文。
func (c *BaseI18nCase) GetBaseI18nNameMapByLocale(ctx context.Context, targetType adminv1.I18nTargetType, locale string, targetIds []int64) (map[int64]string, error) {
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
	query := c.Query(ctx).BaseI18N
	opts := make([]repository.QueryOption, 0, 3)
	opts = append(opts, repository.Where(query.TargetType.Eq(int32(targetType))))
	opts = append(opts, repository.Where(query.Locale.Eq(locale)))
	opts = append(opts, repository.Where(query.TargetID.In(targetIds...)))

	var list []*models.BaseI18N
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

// SaveBaseI18n 保存主语言源文对应的翻译信息，缺失译文由定时任务统一补齐。
func (c *BaseI18nCase) SaveBaseI18n(ctx context.Context, targetType adminv1.I18nTargetType, targetId int64, primaryText string, i18ns []*adminv1.BaseI18n, updateMain func(context.Context, string) error) error {
	var err error
	var state *dto.LocaleState
	state, err = c.LocaleState(ctx)
	if err != nil {
		return err
	}

	save := func(txCtx context.Context) error {
		query := c.Query(txCtx).BaseI18N
		opts := make([]repository.QueryOption, 0, 2)
		opts = append(opts, repository.Where(query.TargetType.Eq(int32(targetType))))
		opts = append(opts, repository.Where(query.TargetID.Eq(targetId)))
		var list []*models.BaseI18N
		list, err = c.List(txCtx, opts...)
		if err != nil {
			return err
		}
		existing := make(map[string]*models.BaseI18N, len(list))
		for _, item := range list {
			existing[item.Locale] = item
		}

		values := make(map[string]string, len(i18ns))
		seen := make(map[string]struct{}, len(i18ns))
		for _, i18n := range i18ns {
			if i18n.GetTargetType() != targetType {
				return errorsx.InvalidArgument("翻译目标类型无效")
			}
			localeValue := i18n.GetLocale()
			if !state.IsEditable(localeValue) {
				return errorsx.InvalidArgument("翻译语言必须是已启用的非主语言")
			}
			if _, duplicated := seen[localeValue]; duplicated {
				return errorsx.Conflict("同一资源语言不能重复")
			}
			seen[localeValue] = struct{}{}
			values[localeValue] = i18n.GetName()
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
				if err = c.Create(txCtx, &models.BaseI18N{TargetType: int32(targetType), TargetID: targetId, Locale: localeValue, Name: text}); err != nil {
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
	return err
}

// SaveGeneratedI18ns 保存代码生成器提供的非主语言译文，不覆盖已有非空内容。
func (c *BaseI18nCase) SaveGeneratedI18ns(ctx context.Context, targetType adminv1.I18nTargetType, targetID int64, i18ns map[string]string) error {
	if targetID <= 0 || len(i18ns) == 0 {
		return nil
	}
	state, err := c.LocaleState(ctx)
	if err != nil {
		return err
	}
	query := c.Query(ctx).BaseI18N
	var rows []*models.BaseI18N
	rows, err = c.List(ctx,
		repository.Where(query.TargetType.Eq(int32(targetType))),
		repository.Where(query.TargetID.Eq(targetID)),
	)
	if err != nil {
		return err
	}
	existing := make(map[string]*models.BaseI18N, len(rows))
	for _, row := range rows {
		existing[row.Locale] = row
	}
	for locale, text := range i18ns {
		if text == "" || !state.IsEditable(locale) {
			continue
		}
		row := existing[locale]
		if row != nil && row.Name != "" {
			continue
		}
		if row == nil {
			if err = c.Create(ctx, &models.BaseI18N{TargetType: int32(targetType), TargetID: targetID, Locale: locale, Name: text}); err != nil {
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

// DeleteBaseI18n 删除翻译信息。
func (c *BaseI18nCase) DeleteBaseI18n(ctx context.Context, targetType adminv1.I18nTargetType, targetId []int64) error {
	if len(targetId) == 0 {
		return nil
	}
	query := c.Query(ctx).BaseI18N
	opts := make([]repository.QueryOption, 0, 2)
	opts = append(opts, repository.Where(query.TargetType.Eq(int32(targetType))))
	opts = append(opts, repository.Where(query.TargetID.In(targetId...)))
	return c.Delete(ctx, opts...)
}

// protectI18nText 使用稳定哨兵替换不应发送给翻译器改写的结构化片段。
func protectI18nText(source string) (string, []string) {
	values := make([]string, 0)
	protected := protectedI18nTextPattern.ReplaceAllStringFunc(source, func(value string) string {
		index := len(values)
		values = append(values, value)
		return protectedI18nToken(index)
	})
	return protected, values
}

// protectedI18nToken 返回指定位置的稳定翻译哨兵。
func protectedI18nToken(index int) string {
	return fmt.Sprintf("__KRATOS_I18N_TOKEN_%03d__", index)
}
