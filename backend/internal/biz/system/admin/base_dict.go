package biz

import (
	"context"
	"sort"

	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	"github.com/liujitcn/kratos-core/biz"
	"github.com/liujitcn/kratos-core/errorsx"

	"github.com/liujitcn/go-utils/mapper"
	_string "github.com/liujitcn/go-utils/string"
	"github.com/liujitcn/gorm-kit/repository"
	"gorm.io/gen/field"
)

// BaseDictCase 字典业务实例
type BaseDictCase struct {
	*biz.BaseCase
	tx data.Transaction
	*data.BaseDictRepository
	baseDictItemCase *BaseDictItemCase
	baseI18nCase     *BaseI18nCase
	formMapper       *mapper.CopierMapper[adminv1.BaseDictForm, models.BaseDict]
	mapper           *mapper.CopierMapper[adminv1.BaseDict, models.BaseDict]
}

// NewBaseDictCase 创建字典业务实例
func NewBaseDictCase(baseCase *biz.BaseCase, tx data.Transaction, baseDictRepo *data.BaseDictRepository, baseDictItemCase *BaseDictItemCase, baseI18nCase *BaseI18nCase) *BaseDictCase {
	return &BaseDictCase{
		BaseCase:           baseCase,
		tx:                 tx,
		BaseDictRepository: baseDictRepo,
		baseDictItemCase:   baseDictItemCase,
		baseI18nCase:       baseI18nCase,
		formMapper:         mapper.NewCopierMapper[adminv1.BaseDictForm, models.BaseDict](),
		mapper:             mapper.NewCopierMapper[adminv1.BaseDict, models.BaseDict](),
	}
}

// OptionBaseDict 查询字典下拉选择
func (c *BaseDictCase) OptionBaseDict(ctx context.Context) (*adminv1.OptionBaseDictResponse, error) {
	query := c.Query(ctx).BaseDict
	opts := make([]repository.QueryOption, 0, 1)
	opts = append(opts, repository.Order(query.CreatedAt.Desc()))
	baseDictList, err := c.List(ctx, opts...)
	if err != nil {
		return nil, err
	}

	baseDictItemQuery := c.baseDictItemCase.Query(ctx).BaseDictItem
	itemOpts := make([]repository.QueryOption, 0, 2)
	itemOpts = append(itemOpts, repository.Order(baseDictItemQuery.Sort.Asc()))
	itemOpts = append(itemOpts, repository.Order(baseDictItemQuery.CreatedAt.Desc()))
	var baseDictItemList []*models.BaseDictItem
	baseDictItemList, err = c.baseDictItemCase.List(ctx, itemOpts...)
	if err != nil {
		return nil, err
	}

	dictItemMap := make(map[int64][]*models.BaseDictItem)
	for _, item := range baseDictItemList {
		dictItemMap[item.DictID] = append(dictItemMap[item.DictID], item)
	}
	dictIDs := make([]int64, 0, len(baseDictList))
	for _, item := range baseDictList {
		dictIDs = append(dictIDs, item.ID)
	}
	var dictNames map[int64]string
	dictNames, err = c.baseI18nCase.GetBaseI18nNameMapByLocale(ctx, adminv1.I18nTargetType_I18N_TARGET_TYPE_BASE_DICT_NAME, biz.LocaleFromContext(ctx), dictIDs)
	if err != nil {
		return nil, err
	}
	dictItemIDs := make([]int64, 0, len(baseDictItemList))
	for _, item := range baseDictItemList {
		dictItemIDs = append(dictItemIDs, item.ID)
	}
	var dictItemLabels map[int64]string
	dictItemLabels, err = c.baseI18nCase.GetBaseI18nNameMapByLocale(ctx, adminv1.I18nTargetType_I18N_TARGET_TYPE_BASE_DICT_ITEM_LABEL, biz.LocaleFromContext(ctx), dictItemIDs)
	if err != nil {
		return nil, err
	}

	resList := make([]*adminv1.OptionBaseDictResponse_BaseDict, 0, len(baseDictList))
	for _, dict := range baseDictList {
		items := make([]*adminv1.OptionBaseDictResponse_BaseDictItem, 0)
		dictItems, ok := dictItemMap[dict.ID]
		// 当前字典存在子项时，按排序字段稳定输出字典项。
		if ok {
			sort.SliceStable(dictItems, func(i, j int) bool {
				return dictItems[i].Sort < dictItems[j].Sort
			})
			for _, dictItem := range dictItems {
				label := dictItem.Label
				if translated := dictItemLabels[dictItem.ID]; translated != "" {
					label = translated
				}
				items = append(items, &adminv1.OptionBaseDictResponse_BaseDictItem{
					Value:   dictItem.Value,
					Label:   label,
					TagType: dictItem.TagType,
				})
			}
		}
		name := dict.Name
		if translated := dictNames[dict.ID]; translated != "" {
			name = translated
		}
		resList = append(resList, &adminv1.OptionBaseDictResponse_BaseDict{
			Code:  dict.Code,
			Name:  name,
			Items: items,
		})
	}
	return &adminv1.OptionBaseDictResponse{BaseDicts: resList}, nil
}

// PageBaseDict 分页查询字典
func (c *BaseDictCase) PageBaseDict(ctx context.Context, req *adminv1.PageBaseDictRequest) (*adminv1.PageBaseDictResponse, error) {
	query := c.Query(ctx).BaseDict
	opts := make([]repository.QueryOption, 0, 4)
	opts = append(opts, repository.Order(query.CreatedAt.Desc()))
	if req.Status != nil {
		opts = append(opts, repository.Where(query.Status.Eq(int32(req.GetStatus()))))
	}
	// 传入名称关键字时，按名称模糊匹配字典。
	var err error
	if req.GetName() != "" {
		var translatedIDs []int64
		translatedIDs, err = c.baseI18nCase.GetTargetIdsByName(ctx, adminv1.I18nTargetType_I18N_TARGET_TYPE_BASE_DICT_NAME, req.GetName())
		if err != nil {
			return nil, err
		}
		nameCondition := query.Name.Like("%" + req.GetName() + "%")
		if len(translatedIDs) > 0 {
			opts = append(opts, repository.Where(field.Or(nameCondition, query.ID.In(translatedIDs...))))
		} else {
			opts = append(opts, repository.Where(nameCondition))
		}
	}
	// 传入编码关键字时，按编码模糊匹配字典。
	if req.GetCode() != "" {
		opts = append(opts, repository.Where(query.Code.Like("%"+req.GetCode()+"%")))
	}

	var list []*models.BaseDict
	var total int64
	list, total, err = c.Page(ctx, req.GetPageNum(), req.GetPageSize(), opts...)
	if err != nil {
		return nil, err
	}
	resList := make([]*adminv1.BaseDict, 0, len(list))
	targetIds := make([]int64, 0, len(list))
	for _, item := range list {
		targetIds = append(targetIds, item.ID)
	}
	var i18ns map[int64][]*adminv1.BaseI18n
	i18ns, err = c.baseI18nCase.GetBaseI18nMapByTargetType(ctx, adminv1.I18nTargetType_I18N_TARGET_TYPE_BASE_DICT_NAME, targetIds)
	if err != nil {
		return nil, err
	}
	for _, item := range list {
		baseDict := c.mapper.ToDTO(item)
		baseDict.I18ns = i18ns[item.ID]
		resList = append(resList, baseDict)
	}
	return &adminv1.PageBaseDictResponse{BaseDicts: resList, Total: int32(total)}, nil
}

// GetBaseDict 获取字典
func (c *BaseDictCase) GetBaseDict(ctx context.Context, id int64) (*adminv1.BaseDictForm, error) {
	baseDict, err := c.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	res := c.formMapper.ToDTO(baseDict)
	var i18ns map[int64][]*adminv1.BaseI18n
	i18ns, err = c.baseI18nCase.GetBaseI18nMapByTargetType(ctx, adminv1.I18nTargetType_I18N_TARGET_TYPE_BASE_DICT_NAME, []int64{id})
	if err != nil {
		return nil, err
	}
	res.I18ns = i18ns[id]
	return res, nil
}

// CreateBaseDict 创建字典
func (c *BaseDictCase) CreateBaseDict(ctx context.Context, req *adminv1.BaseDictForm) error {
	baseDict := c.formMapper.ToEntity(req)
	err := c.Create(ctx, baseDict)
	if err != nil {
		// 命中字典编码唯一索引冲突时，返回稳定的业务冲突错误。
		if errorsx.IsDuplicateKey(err) {
			return errorsx.UniqueConflict("字典编码重复", "base_dict", "code", "unique_base_dict").WithCause(err)
		}
		return err
	}
	return c.saveBaseI18n(ctx, req, baseDict)
}

// UpdateBaseDict 更新字典
func (c *BaseDictCase) UpdateBaseDict(ctx context.Context, req *adminv1.BaseDictForm) error {
	baseDict := c.formMapper.ToEntity(req)
	err := c.UpdateByID(ctx, baseDict)
	if err != nil {
		// 命中字典编码唯一索引冲突时，返回稳定的业务冲突错误。
		if errorsx.IsDuplicateKey(err) {
			return errorsx.UniqueConflict("字典编码重复", "base_dict", "code", "unique_base_dict").WithCause(err)
		}
		return err
	}
	return c.saveBaseI18n(ctx, req, baseDict)
}

// DeleteBaseDict 删除字典
func (c *BaseDictCase) DeleteBaseDict(ctx context.Context, id string) error {
	ids := _string.ConvertStringToInt64Array(id)
	query := c.baseDictItemCase.Query(ctx).BaseDictItem
	for _, dictID := range ids {
		opts := make([]repository.QueryOption, 0, 1)
		opts = append(opts, repository.Where(query.DictID.Eq(dictID)))
		count, err := c.baseDictItemCase.Count(ctx, opts...)
		if err != nil {
			return errorsx.Internal("删除字典失败").WithCause(err)
		}
		// 字典下仍有子项时，不允许直接删除字典。
		if count > 0 {
			return errorsx.HasChildrenConflict("删除字典失败，下面有属性", "base_dict", "base_dict_item")
		}
	}
	return c.tx.Transaction(ctx, func(ctx context.Context) error {
		err := c.DeleteByIDs(ctx, ids)
		if err != nil {
			return err
		}
		return c.baseI18nCase.DeleteBaseI18n(ctx, adminv1.I18nTargetType_I18N_TARGET_TYPE_BASE_DICT_NAME, ids)
	})
}

// SetBaseDictStatus 设置字典状态
func (c *BaseDictCase) SetBaseDictStatus(ctx context.Context, req *adminv1.SetBaseDictStatusRequest) error {
	return c.UpdateByID(ctx, &models.BaseDict{
		ID:     req.GetId(),
		Status: req.GetStatus(),
	})
}

// saveBaseI18n 保存字典名称翻译并同步主表名称。
func (c *BaseDictCase) saveBaseI18n(ctx context.Context, req *adminv1.BaseDictForm, entity *models.BaseDict) error {
	return c.baseI18nCase.SaveBaseI18n(ctx, adminv1.I18nTargetType_I18N_TARGET_TYPE_BASE_DICT_NAME, entity.ID, entity.Name, req.GetI18ns(), func(ctx context.Context, name string) error {
		return c.UpdateByID(ctx, &models.BaseDict{ID: entity.ID, Name: name})
	})
}
