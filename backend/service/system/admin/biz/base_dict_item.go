package biz

import (
	"context"

	systemadminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/pkg/biz"
	"github.com/liujitcn/kratos-admin/backend/pkg/errorsx"
	"github.com/liujitcn/kratos-admin/backend/pkg/gen/data"
	"github.com/liujitcn/kratos-admin/backend/pkg/gen/models"

	"github.com/liujitcn/go-utils/mapper"
	_string "github.com/liujitcn/go-utils/string"
	"github.com/liujitcn/gorm-kit/repository"
)

// BaseDictItemCase 字典项业务实例
type BaseDictItemCase struct {
	*biz.BaseCase
	baseDictRepo *data.BaseDictRepository
	*data.BaseDictItemRepository
	formMapper *mapper.CopierMapper[systemadminv1.BaseDictItemForm, models.BaseDictItem]
	mapper     *mapper.CopierMapper[systemadminv1.BaseDictItem, models.BaseDictItem]
}

// NewBaseDictItemCase 创建字典项业务实例
func NewBaseDictItemCase(baseCase *biz.BaseCase, baseDictRepo *data.BaseDictRepository, baseDictItemRepo *data.BaseDictItemRepository) *BaseDictItemCase {
	return &BaseDictItemCase{
		BaseCase:               baseCase,
		baseDictRepo:           baseDictRepo,
		BaseDictItemRepository: baseDictItemRepo,
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
	if req.GetLabel() != "" {
		opts = append(opts, repository.Where(query.Label.Like("%"+req.GetLabel()+"%")))
	}

	list, total, err := c.Page(ctx, req.GetPageNum(), req.GetPageSize(), opts...)
	if err != nil {
		return nil, err
	}

	resList := make([]*systemadminv1.BaseDictItem, 0, len(list))
	for _, item := range list {
		baseDictItem := c.mapper.ToDTO(item)
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
	return res, nil
}

// CreateBaseDictItem 创建字典项
func (c *BaseDictItemCase) CreateBaseDictItem(ctx context.Context, req *systemadminv1.BaseDictItemForm) error {
	baseDictItem := c.formMapper.ToEntity(req)
	err := c.Create(ctx, baseDictItem)
	if err != nil {
		// 命中字典项属性值唯一索引冲突时，返回稳定的业务冲突错误。
		if errorsx.IsMySQLDuplicateKey(err) {
			return errorsx.UniqueConflict("同一字典的属性值重复", "base_dict_item", "", "unique_base_dict").WithCause(err)
		}
		return err
	}
	return nil
}

// UpdateBaseDictItem 更新字典项
func (c *BaseDictItemCase) UpdateBaseDictItem(ctx context.Context, req *systemadminv1.BaseDictItemForm) error {
	baseDictItem := c.formMapper.ToEntity(req)
	err := c.UpdateByID(ctx, baseDictItem)
	if err != nil {
		// 命中字典项属性值唯一索引冲突时，返回稳定的业务冲突错误。
		if errorsx.IsMySQLDuplicateKey(err) {
			return errorsx.UniqueConflict("同一字典的属性值重复", "base_dict_item", "", "unique_base_dict").WithCause(err)
		}
		return err
	}
	return nil
}

// DeleteBaseDictItem 删除字典项
func (c *BaseDictItemCase) DeleteBaseDictItem(ctx context.Context, id string) error {
	return c.DeleteByIDs(ctx, _string.ConvertStringToInt64Array(id))
}

// SetBaseDictItemStatus 设置字典项状态
func (c *BaseDictItemCase) SetBaseDictItemStatus(ctx context.Context, req *systemadminv1.SetBaseDictItemStatusRequest) error {
	return c.UpdateByID(ctx, &models.BaseDictItem{
		ID:     req.GetId(),
		Status: req.GetStatus(),
	})
}
