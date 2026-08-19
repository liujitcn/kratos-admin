package biz

import (
	"context"
	"sort"

	adminbiz "github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	"github.com/liujitcn/kratos-core/biz"
	"github.com/liujitcn/kratos-core/const"

	"github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/api/gen/go/system/app/v1"

	"github.com/liujitcn/go-utils/mapper"
	"github.com/liujitcn/gorm-kit/repository"
)

// BaseDictCase 字典业务处理对象
type BaseDictCase struct {
	*biz.BaseCase
	*data.BaseDictRepository
	baseDictItemCase *BaseDictItemCase
	translationCase  *adminbiz.BaseTranslationCase
	dictMapper       *mapper.CopierMapper[appv1.BaseDictForm, models.BaseDict]
	itemMapper       *mapper.CopierMapper[appv1.BaseDictForm_DictItem, models.BaseDictItem]
}

// NewBaseDictCase 创建字典业务处理对象
func NewBaseDictCase(baseCase *biz.BaseCase, baseDictRepo *data.BaseDictRepository, baseDictItemCase *BaseDictItemCase, translationCase *adminbiz.BaseTranslationCase) *BaseDictCase {
	return &BaseDictCase{
		BaseCase:           baseCase,
		BaseDictRepository: baseDictRepo,
		baseDictItemCase:   baseDictItemCase,
		translationCase:    translationCase,
		dictMapper:         mapper.NewCopierMapper[appv1.BaseDictForm, models.BaseDict](),
		itemMapper:         mapper.NewCopierMapper[appv1.BaseDictForm_DictItem, models.BaseDictItem](),
	}
}

// GetBaseDict 查询字典
func (c *BaseDictCase) GetBaseDict(ctx context.Context, code string) (*appv1.BaseDictForm, error) {
	query := c.Query(ctx).BaseDict
	opts := make([]repository.QueryOption, 0, 3)
	opts = append(opts, repository.Order(query.CreatedAt.Desc()))
	opts = append(opts, repository.Where(query.Code.Eq(code)))
	opts = append(opts, repository.Where(query.Status.Eq(_const.STATUS_STATUS_ENABLE)))
	baseDict, err := c.Find(ctx, opts...)
	if err != nil {
		return nil, err
	}

	var baseDictItemList []*models.BaseDictItem
	baseDictItemList, err = c.baseDictItemCase.findByDictIDs(ctx, []int64{baseDict.ID})
	if err != nil {
		return nil, err
	}

	dictItemMap := make(map[int64][]*models.BaseDictItem)
	for _, item := range baseDictItemList {
		dictItemMap[item.DictID] = append(dictItemMap[item.DictID], item)
	}
	var dictNames map[int64]string
	dictNames, err = c.translationCase.GetBaseTranslationNameMapByLocale(ctx, adminv1.TranslationTargetType_TRANSLATION_TARGET_TYPE_BASE_DICT, biz.LocaleFromContext(ctx), []int64{baseDict.ID})
	if err != nil {
		return nil, err
	}
	dictItemIDs := make([]int64, 0, len(baseDictItemList))
	for _, item := range baseDictItemList {
		dictItemIDs = append(dictItemIDs, item.ID)
	}
	var dictItemLabels map[int64]string
	dictItemLabels, err = c.translationCase.GetBaseTranslationNameMapByLocale(ctx, adminv1.TranslationTargetType_TRANSLATION_TARGET_TYPE_BASE_DICT_ITEM, biz.LocaleFromContext(ctx), dictItemIDs)
	if err != nil {
		return nil, err
	}

	items := make([]*appv1.BaseDictForm_DictItem, 0)
	// 命中字典项映射时，再按排序规则组装当前字典的子项。
	if dictItems, ok := dictItemMap[baseDict.ID]; ok {
		sort.SliceStable(dictItems, func(i, j int) bool {
			return dictItems[i].Sort < dictItems[j].Sort
		})
		for _, dictItem := range dictItems {
			item := c.itemMapper.ToDTO(dictItem)
			if translated := dictItemLabels[dictItem.ID]; translated != "" {
				item.Label = translated
			}
			items = append(items, item)
		}
	}

	res := c.dictMapper.ToDTO(baseDict)
	if translated := dictNames[baseDict.ID]; translated != "" {
		res.Name = translated
	}
	res.Items = items
	return res, nil
}
