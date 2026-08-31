package biz

import (
	"context"

	basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"
	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	commonv1 "github.com/liujitcn/kratos-core/api/gen/go/common/v1"
	"github.com/liujitcn/kratos-core/biz"
	coreconst "github.com/liujitcn/kratos-core/const"
	"github.com/liujitcn/kratos-core/errorsx"

	"github.com/liujitcn/go-utils/mapper"
	_string "github.com/liujitcn/go-utils/string"
	"github.com/liujitcn/gorm-kit/repository"
	"github.com/liujitcn/kratos-kit/database/gorm"
)

// BaseMessageCategoryCase 消息分类业务实例。
type BaseMessageCategoryCase struct {
	*biz.BaseCase
	tx data.Transaction
	*data.BaseMessageCategoryRepository
	baseMessageRepo *data.BaseMessageRepository
	formMapper      *mapper.CopierMapper[adminv1.BaseMessageCategoryForm, models.BaseMessageCategory]
	mapper          *mapper.CopierMapper[adminv1.BaseMessageCategory, models.BaseMessageCategory]
}

// NewBaseMessageCategoryCase 创建消息分类业务实例。
func NewBaseMessageCategoryCase(
	baseCase *biz.BaseCase,
	tx data.Transaction,
	baseMessageCategoryRepo *data.BaseMessageCategoryRepository,
	baseMessageRepo *data.BaseMessageRepository,
) *BaseMessageCategoryCase {
	return &BaseMessageCategoryCase{
		BaseCase:                      baseCase,
		tx:                            tx,
		BaseMessageCategoryRepository: baseMessageCategoryRepo,
		baseMessageRepo:               baseMessageRepo,
		formMapper:                    mapper.NewCopierMapper[adminv1.BaseMessageCategoryForm, models.BaseMessageCategory](),
		mapper:                        mapper.NewCopierMapper[adminv1.BaseMessageCategory, models.BaseMessageCategory](),
	}
}

// ensureDefaultTenantAccess 校验消息分类维护权限，仅默认租户可以修改公共分类。
func (c *BaseMessageCategoryCase) ensureDefaultTenantAccess(ctx context.Context) error {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}
	if authInfo.TenantCode != gorm.DefaultTenantCode {
		return errorsx.PermissionDenied("只有默认租户可以维护消息分类")
	}
	return nil
}

// OptionBaseMessageCategory 查询消息分类选项。
func (c *BaseMessageCategoryCase) OptionBaseMessageCategory(ctx context.Context, _ *adminv1.OptionBaseMessageCategoryRequest) (*commonv1.SelectOptionResponse, error) {
	query := c.Query(ctx).BaseMessageCategory
	opts := make([]repository.QueryOption, 0, 2)
	opts = append(opts, repository.Order(query.Sort.Asc()), repository.Order(query.ID.Asc()))
	var list []*models.BaseMessageCategory
	var err error
	list, err = c.List(ctx, opts...)
	if err != nil {
		return nil, err
	}
	options := make([]*commonv1.SelectOptionResponse_Option, 0, len(list))
	for _, item := range list {
		options = append(options, &commonv1.SelectOptionResponse_Option{
			Label:    item.Name,
			Value:    item.ID,
			Disabled: item.Status != coreconst.STATUS_STATUS_ENABLE,
		})
	}
	return &commonv1.SelectOptionResponse{List: options}, nil
}

// PageBaseMessageCategory 分页查询消息分类。
func (c *BaseMessageCategoryCase) PageBaseMessageCategory(ctx context.Context, req *adminv1.PageBaseMessageCategoryRequest) (*adminv1.PageBaseMessageCategoryResponse, error) {
	query := c.Query(ctx).BaseMessageCategory
	opts := make([]repository.QueryOption, 0, 6)
	opts = append(opts, repository.Order(query.Sort.Asc()), repository.Order(query.ID.Desc()))
	if req.GetName() != "" {
		opts = append(opts, repository.Where(query.Name.Like("%"+req.GetName()+"%")))
	}
	if req.GetCode() != "" {
		opts = append(opts, repository.Where(query.Code.Like("%"+req.GetCode()+"%")))
	}
	if req.Status != nil {
		opts = append(opts, repository.Where(query.Status.Eq(int32(req.GetStatus()))))
	}
	var list []*models.BaseMessageCategory
	var total int64
	var err error
	list, total, err = c.Page(ctx, req.GetPageNum(), req.GetPageSize(), opts...)
	if err != nil {
		return nil, err
	}
	result := make([]*adminv1.BaseMessageCategory, 0, len(list))
	for _, item := range list {
		value := c.mapper.ToDTO(item)
		result = append(result, value)
	}
	return &adminv1.PageBaseMessageCategoryResponse{BaseMessageCategories: result, Total: int32(total)}, nil
}

// GetBaseMessageCategory 查询消息分类详情。
func (c *BaseMessageCategoryCase) GetBaseMessageCategory(ctx context.Context, id int64) (*adminv1.BaseMessageCategoryForm, error) {
	entity, err := c.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return c.formMapper.ToDTO(entity), nil
}

// CreateBaseMessageCategory 创建消息分类。
func (c *BaseMessageCategoryCase) CreateBaseMessageCategory(ctx context.Context, req *adminv1.BaseMessageCategoryForm) error {
	err := c.ensureDefaultTenantAccess(ctx)
	if err != nil {
		return err
	}
	entity := c.formMapper.ToEntity(req)
	if entity.DefaultPriority == 0 {
		entity.DefaultPriority = int32(basev1.MessagePriority_MESSAGE_PRIORITY_NORMAL)
	}
	if entity.RetentionDays == 0 {
		entity.RetentionDays = defaultMessageRetentionDays
	}
	if entity.Status == 0 {
		entity.Status = coreconst.STATUS_STATUS_ENABLE
	}
	return c.tx.Transaction(ctx, func(txCtx context.Context) error {
		err = c.Create(txCtx, entity)
		if err != nil {
			if errorsx.IsDuplicateKey(err) {
				return errorsx.UniqueConflict("消息分类编码重复", "base_message_category", "code", "unique_base_message_category").WithCause(err)
			}
			return err
		}
		return nil
	})
}

// UpdateBaseMessageCategory 更新消息分类。
func (c *BaseMessageCategoryCase) UpdateBaseMessageCategory(ctx context.Context, req *adminv1.BaseMessageCategoryForm) error {
	err := c.ensureDefaultTenantAccess(ctx)
	if err != nil {
		return err
	}
	var oldEntity *models.BaseMessageCategory
	oldEntity, err = c.FindByID(ctx, req.GetId())
	if err != nil {
		return err
	}
	if req.GetCode() != oldEntity.Code {
		return errorsx.Conflict("消息分类编码不可修改")
	}
	entity := c.formMapper.ToEntity(req)
	entity.ID = oldEntity.ID
	if entity.Status == 0 {
		entity.Status = oldEntity.Status
	}
	return c.tx.Transaction(ctx, func(txCtx context.Context) error {
		err = c.UpdateByID(txCtx, entity)
		if err != nil {
			if errorsx.IsDuplicateKey(err) {
				return errorsx.UniqueConflict("消息分类编码重复", "base_message_category", "code", "unique_base_message_category").WithCause(err)
			}
			return err
		}
		return nil
	})
}

// DeleteBaseMessageCategory 删除未被消息引用的分类。
func (c *BaseMessageCategoryCase) DeleteBaseMessageCategory(ctx context.Context, id string) error {
	err := c.ensureDefaultTenantAccess(ctx)
	if err != nil {
		return err
	}
	ids := _string.ConvertStringToInt64Array(id)
	var list []*models.BaseMessageCategory
	list, err = c.ListByIDs(ctx, ids)
	if err != nil {
		return err
	}
	if len(list) != len(ids) {
		return errorsx.ResourceNotFound("删除消息分类失败，分类不存在")
	}
	query := c.baseMessageRepo.Query(ctx).BaseMessage
	var count int64
	count, err = c.baseMessageRepo.Count(ctx, repository.Where(query.CategoryID.In(ids...)))
	if err != nil {
		return err
	}
	if count > 0 {
		return errorsx.HasChildrenConflict("删除消息分类失败，仍有消息使用该分类", "base_message_category", "base_message")
	}
	return c.tx.Transaction(ctx, func(txCtx context.Context) error {
		err = c.DeleteByIDs(txCtx, ids)
		if err != nil {
			return err
		}
		return nil
	})
}

// SetBaseMessageCategoryStatus 设置消息分类状态。
func (c *BaseMessageCategoryCase) SetBaseMessageCategoryStatus(ctx context.Context, req *adminv1.SetBaseMessageCategoryStatusRequest) error {
	err := c.ensureDefaultTenantAccess(ctx)
	if err != nil {
		return err
	}
	var entity *models.BaseMessageCategory
	entity, err = c.FindByID(ctx, req.GetId())
	if err != nil {
		return err
	}
	if int32(req.GetStatus()) != coreconst.STATUS_STATUS_ENABLE && int32(req.GetStatus()) != coreconst.STATUS_STATUS_DISABLE {
		return errorsx.InvalidArgument("消息分类状态无效")
	}
	if entity.Status == int32(req.GetStatus()) {
		return nil
	}
	return c.UpdateByID(ctx, &models.BaseMessageCategory{ID: entity.ID, Status: int32(req.GetStatus())})
}
