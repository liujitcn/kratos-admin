package biz

import (
	"context"

	systemadminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/core/pkg/errorsx"
	coreLocale "github.com/liujitcn/kratos-admin/backend/core/pkg/locale"
	"github.com/liujitcn/kratos-admin/backend/internal/biz"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"

	"github.com/liujitcn/go-utils/mapper"
	_string "github.com/liujitcn/go-utils/string"
	"github.com/liujitcn/gorm-kit/repository"
	"gorm.io/gen/field"
)

// BaseDictItemCase 字典项业务实例
type BaseDictItemCase struct {
	*biz.BaseCase
	tx           data.Transaction
	baseDictRepo *data.BaseDictRepository
	*data.BaseDictItemRepository
	translationCase *BaseTranslationCase
	formMapper      *mapper.CopierMapper[systemadminv1.BaseDictItemForm, models.BaseDictItem]
	mapper          *mapper.CopierMapper[systemadminv1.BaseDictItem, models.BaseDictItem]
}

// NewBaseDictItemCase 创建字典项业务实例
func NewBaseDictItemCase(baseCase *biz.BaseCase, tx data.Transaction, baseDictRepo *data.BaseDictRepository, baseDictItemRepo *data.BaseDictItemRepository, translationCase *BaseTranslationCase) *BaseDictItemCase {
	return &BaseDictItemCase{
		BaseCase:               baseCase,
		tx:                     tx,
		baseDictRepo:           baseDictRepo,
		BaseDictItemRepository: baseDictItemRepo,
		translationCase:        translationCase,
		formMapper:             mapper.NewCopierMapper[systemadminv1.BaseDictItemForm, models.BaseDictItem](),
		mapper:                 mapper.NewCopierMapper[systemadminv1.BaseDictItem, models.BaseDictItem](),
	}
}

// PageBaseDictItem 分页查询字典项
func (c *BaseDictItemCase) PageBaseDictItem(ctx context.Context, req *systemadminv1.PageBaseDictItemRequest) (*systemadminv1.PageBaseDictItemResponse, error) {
	query := c.Query(ctx).BaseDictItem
	opts := make([]repository.QueryOption, 0, 5)
	opts = append(opts, repository.Order(query.Sort.Asc()))
	opts = append(opts, repository.Order(query.CreatedAt.Desc()))
	// 传入字典编号时，按所属字典过滤字典项。
	if req.GetDictId() > 0 {
		opts = append(opts, repository.Where(query.DictID.Eq(req.GetDictId())))
	}
	if req.Status != nil {
		opts = append(opts, repository.Where(query.Status.Eq(int32(req.GetStatus()))))
	}
	// 传入标签关键字时，按标签模糊匹配字典项。
	var err error
	if req.GetLabel() != "" {
		var translatedIDs []int64
		translatedIDs, err = c.translationCase.ReviewedDictItemIDsByLabel(ctx, req.GetLabel())
		if err != nil {
			return nil, err
		}
		labelCondition := query.Label.Like("%" + req.GetLabel() + "%")
		if len(translatedIDs) > 0 {
			opts = append(opts, repository.Where(field.Or(labelCondition, query.ID.In(translatedIDs...))))
		} else {
			opts = append(opts, repository.Where(labelCondition))
		}
	}

	var list []*models.BaseDictItem
	var total int64
	list, total, err = c.Page(ctx, req.GetPageNum(), req.GetPageSize(), opts...)
	if err != nil {
		return nil, err
	}
	sources := make(map[int64]string, len(list))
	for _, item := range list {
		sources[item.ID] = item.Label
	}
	var translations map[int64][]*systemadminv1.BaseDictItemTranslation
	translations, err = c.translationCase.DictItemTranslations(ctx, sources)
	if err != nil {
		return nil, err
	}
	resList := make([]*systemadminv1.BaseDictItem, 0, len(list))
	for _, item := range list {
		baseDictItem := c.mapper.ToDTO(item)
		baseDictItem.Translations = translations[item.ID]
		resList = append(resList, baseDictItem)
	}
	return &systemadminv1.PageBaseDictItemResponse{BaseDictItems: resList, Total: int32(total)}, nil
}

// GetBaseDictItem 获取字典项
func (c *BaseDictItemCase) GetBaseDictItem(ctx context.Context, id int64) (*systemadminv1.BaseDictItemForm, error) {
	baseDictItem, err := c.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	res := c.formMapper.ToDTO(baseDictItem)
	var translations map[int64][]*systemadminv1.BaseDictItemTranslation
	translations, err = c.translationCase.DictItemTranslations(ctx, map[int64]string{id: baseDictItem.Label})
	if err != nil {
		return nil, err
	}
	res.Translations = translations[id]
	return res, nil
}

// CreateBaseDictItem 创建字典项
func (c *BaseDictItemCase) CreateBaseDictItem(ctx context.Context, req *systemadminv1.BaseDictItemForm) error {
	var err error
	sourceText := req.GetLabel()
	var primaryText, sourceLocale, primaryLocale string
	primaryText, sourceLocale, primaryLocale, err = c.translationCase.NormalizePrimaryText(ctx, sourceText)
	if err != nil {
		return err
	}
	req.Label = primaryText
	translations := appendDictItemSourceTranslation(req.GetTranslations(), sourceLocale, primaryLocale, sourceText)
	baseDictItem := c.formMapper.ToEntity(req)
	return c.tx.Transaction(ctx, func(ctx context.Context) error {
		err = c.Create(ctx, baseDictItem)
		if err != nil {
			// 命中字典项属性值唯一索引冲突时，返回稳定的业务冲突错误。
			if errorsx.IsMySQLDuplicateKey(err) {
				return errorsx.UniqueConflict("同一字典的属性值重复", "base_dict_item", "", "unique_base_dict").WithCause(err)
			}
			return err
		}
		return c.translationCase.SaveDictItemTranslations(ctx, baseDictItem.ID, baseDictItem.Label, translations)
	})
}

// UpdateBaseDictItem 更新字典项
func (c *BaseDictItemCase) UpdateBaseDictItem(ctx context.Context, req *systemadminv1.BaseDictItemForm) error {
	var err error
	sourceText := req.GetLabel()
	var primaryText, sourceLocale, primaryLocale string
	primaryText, sourceLocale, primaryLocale, err = c.translationCase.NormalizePrimaryText(ctx, sourceText)
	if err != nil {
		return err
	}
	req.Label = primaryText
	translations := appendDictItemSourceTranslation(req.GetTranslations(), sourceLocale, primaryLocale, sourceText)
	baseDictItem := c.formMapper.ToEntity(req)
	return c.tx.Transaction(ctx, func(ctx context.Context) error {
		var current *models.BaseDictItem
		current, err = c.FindByID(ctx, req.GetId())
		if err != nil {
			return err
		}
		err = c.UpdateByID(ctx, baseDictItem)
		if err != nil {
			// 命中字典项属性值唯一索引冲突时，返回稳定的业务冲突错误。
			if errorsx.IsMySQLDuplicateKey(err) {
				return errorsx.UniqueConflict("同一字典的属性值重复", "base_dict_item", "", "unique_base_dict").WithCause(err)
			}
			return err
		}
		err = c.translationCase.MarkDictItemSourceChanged(ctx, current.ID, current.Label, baseDictItem.Label)
		if err != nil {
			return err
		}
		return c.translationCase.SaveDictItemTranslations(ctx, baseDictItem.ID, baseDictItem.Label, translations)
	})
}

// DeleteBaseDictItem 删除字典项
func (c *BaseDictItemCase) DeleteBaseDictItem(ctx context.Context, id string) error {
	ids := _string.ConvertStringToInt64Array(id)
	return c.tx.Transaction(ctx, func(ctx context.Context) error {
		var err error
		err = c.DeleteByIDs(ctx, ids)
		if err != nil {
			return err
		}
		return c.translationCase.DeleteDictItemTranslations(ctx, ids)
	})
}

// SetBaseDictItemStatus 设置字典项状态
func (c *BaseDictItemCase) SetBaseDictItemStatus(ctx context.Context, req *systemadminv1.SetBaseDictItemStatusRequest) error {
	return c.UpdateByID(ctx, &models.BaseDictItem{
		ID:     req.GetId(),
		Status: req.GetStatus(),
	})
}

// appendDictItemSourceTranslation 保留当前请求语言的原始字典项标签。
func appendDictItemSourceTranslation(translations []*systemadminv1.BaseDictItemTranslation, sourceLocale, primaryLocale, source string) []*systemadminv1.BaseDictItemTranslation {
	if sourceLocale == primaryLocale {
		return translations
	}
	result := make([]*systemadminv1.BaseDictItemTranslation, 0, len(translations)+1)
	added := false
	for _, translation := range translations {
		if coreLocale.IsSupported(translation.GetLocale()) && coreLocale.Normalize(translation.GetLocale()) == sourceLocale {
			if !added {
				result = append(result, &systemadminv1.BaseDictItemTranslation{Locale: sourceLocale, Label: source})
				added = true
			}
			continue
		}
		result = append(result, translation)
	}
	if !added {
		result = append(result, &systemadminv1.BaseDictItemTranslation{Locale: sourceLocale, Label: source})
	}
	return result
}
