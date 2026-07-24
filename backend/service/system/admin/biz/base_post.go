package biz

import (
	"context"

	commonv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/common/v1"
	systemadminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/pkg/biz"
	_const "github.com/liujitcn/kratos-admin/backend/pkg/const"
	"github.com/liujitcn/kratos-admin/backend/pkg/errorsx"
	"github.com/liujitcn/kratos-admin/backend/pkg/gen/data"
	"github.com/liujitcn/kratos-admin/backend/pkg/gen/models"

	"github.com/liujitcn/go-utils/mapper"
	_string "github.com/liujitcn/go-utils/string"
	"github.com/liujitcn/gorm-kit/repository"
	databaseGorm "github.com/liujitcn/kratos-kit/database/gorm"
)

// BasePostCase 岗位业务实例。
type BasePostCase struct {
	*biz.BaseCase
	tx data.Transaction
	*data.BasePostRepository
	baseUserRepo *data.BaseUserRepository
	formMapper   *mapper.CopierMapper[systemadminv1.BasePostForm, models.BasePost]
	mapper       *mapper.CopierMapper[systemadminv1.BasePost, models.BasePost]
}

// NewBasePostCase 创建岗位业务实例。
func NewBasePostCase(
	baseCase *biz.BaseCase,
	tx data.Transaction,
	basePostRepo *data.BasePostRepository,
	baseUserRepo *data.BaseUserRepository,
) *BasePostCase {
	return &BasePostCase{
		BaseCase:           baseCase,
		tx:                 tx,
		BasePostRepository: basePostRepo,
		baseUserRepo:       baseUserRepo,
		formMapper:         mapper.NewCopierMapper[systemadminv1.BasePostForm, models.BasePost](),
		mapper:             mapper.NewCopierMapper[systemadminv1.BasePost, models.BasePost](),
	}
}

// OptionBasePost 查询岗位选项。
func (c *BasePostCase) OptionBasePost(ctx context.Context, req *systemadminv1.OptionBasePostRequest) (*commonv1.SelectOptionResponse, error) {
	query := c.Query(ctx).BasePost
	opts := make([]repository.QueryOption, 0, 3)
	opts = append(opts, repository.Order(query.Sort.Asc()))
	opts = append(opts, repository.Order(query.CreatedAt.Desc()))
	if req.GetTenantId() > 0 {
		opts = append(opts, repository.Where(query.TenantID.Eq(req.GetTenantId())))
	}
	list, err := c.List(ctx, opts...)
	if err != nil {
		return nil, err
	}

	options := make([]*commonv1.SelectOptionResponse_Option, 0, len(list))
	for _, item := range list {
		options = append(options, &commonv1.SelectOptionResponse_Option{
			Label:    item.Name,
			Value:    item.ID,
			Disabled: item.Status != _const.STATUS_ENABLE,
		})
	}
	return &commonv1.SelectOptionResponse{List: options}, nil
}

// PageBasePost 分页查询岗位。
func (c *BasePostCase) PageBasePost(ctx context.Context, req *systemadminv1.PageBasePostRequest) (*systemadminv1.PageBasePostResponse, error) {
	query := c.Query(ctx).BasePost
	opts := make([]repository.QueryOption, 0, 7)
	opts = append(opts, repository.Order(query.Sort.Asc()))
	opts = append(opts, repository.Order(query.CreatedAt.Desc()))
	if req.GetTenantId() > 0 {
		opts = append(opts, repository.Where(query.TenantID.Eq(req.GetTenantId())))
	}
	if req.Status != nil {
		opts = append(opts, repository.Where(query.Status.Eq(int32(req.GetStatus()))))
	}
	if req.GetName() != "" {
		opts = append(opts, repository.Where(query.Name.Like("%"+req.GetName()+"%")))
	}
	if req.GetCode() != "" {
		opts = append(opts, repository.Where(query.Code.Like("%"+req.GetCode()+"%")))
	}
	list, total, err := c.Page(ctx, req.GetPageNum(), req.GetPageSize(), opts...)
	if err != nil {
		return nil, err
	}

	resList := make([]*systemadminv1.BasePost, 0, len(list))
	for _, item := range list {
		resList = append(resList, c.mapper.ToDTO(item))
	}
	return &systemadminv1.PageBasePostResponse{BasePosts: resList, Total: int32(total)}, nil
}

// GetBasePost 获取岗位。
func (c *BasePostCase) GetBasePost(ctx context.Context, id int64) (*systemadminv1.BasePostForm, error) {
	basePost, err := c.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return c.formMapper.ToDTO(basePost), nil
}

// CreateBasePost 创建岗位。
func (c *BasePostCase) CreateBasePost(ctx context.Context, req *systemadminv1.BasePostForm) error {
	basePost := c.formMapper.ToEntity(req)
	tenantID, err := c.resolveTenantID(ctx, req.GetTenantId())
	if err != nil {
		return err
	}
	basePost.TenantID = tenantID
	if basePost.Status == 0 {
		basePost.Status = _const.STATUS_ENABLE
	}
	return c.tx.Transaction(ctx, func(ctx context.Context) error {
		err = c.Create(ctx, basePost)
		if err != nil {
			if errorsx.IsMySQLDuplicateKey(err) {
				return errorsx.UniqueConflict("同一租户的岗位编号重复", "base_post", "code", "unique_base_post").WithCause(err)
			}
			return err
		}
		return nil
	})
}

// UpdateBasePost 更新岗位。
func (c *BasePostCase) UpdateBasePost(ctx context.Context, req *systemadminv1.BasePostForm) error {
	oldBasePost, err := c.FindByID(ctx, req.GetId())
	if err != nil {
		return err
	}
	basePost := c.formMapper.ToEntity(req)
	basePost.TenantID = oldBasePost.TenantID
	basePost.ID = oldBasePost.ID
	if basePost.Status == 0 {
		basePost.Status = oldBasePost.Status
	}
	return c.tx.Transaction(ctx, func(ctx context.Context) error {
		err = c.UpdateByID(ctx, basePost)
		if err != nil {
			if errorsx.IsMySQLDuplicateKey(err) {
				return errorsx.UniqueConflict("同一租户的岗位编号重复", "base_post", "code", "unique_base_post").WithCause(err)
			}
			return err
		}
		return nil
	})
}

// DeleteBasePost 删除岗位。
func (c *BasePostCase) DeleteBasePost(ctx context.Context, id string) error {
	ids := _string.ConvertStringToInt64Array(id)
	posts, err := c.ListByIDs(ctx, ids)
	if err != nil {
		return err
	}
	postMap := make(map[int64]*models.BasePost, len(posts))
	for _, item := range posts {
		postMap[item.ID] = item
	}
	for _, postID := range ids {
		if _, exists := postMap[postID]; !exists {
			return errorsx.ResourceNotFound("删除岗位失败，岗位不存在")
		}
	}

	query := c.baseUserRepo.Query(ctx).BaseUser
	opts := make([]repository.QueryOption, 0, 1)
	opts = append(opts, repository.Where(query.PostID.In(ids...)))
	count, err := c.baseUserRepo.Count(ctx, opts...)
	if err != nil {
		return err
	}
	if count > 0 {
		return errorsx.HasChildrenConflict("删除岗位失败，仍有用户使用该岗位", "base_post", "base_user")
	}
	return c.tx.Transaction(ctx, func(ctx context.Context) error {
		return c.DeleteByIDs(ctx, ids)
	})
}

// SetBasePostStatus 设置岗位状态。
func (c *BasePostCase) SetBasePostStatus(ctx context.Context, req *systemadminv1.SetBasePostStatusRequest) error {
	basePost, err := c.FindByID(ctx, req.GetId())
	if err != nil {
		return err
	}
	if req.GetStatus() != _const.STATUS_ENABLE && req.GetStatus() != _const.STATUS_DISABLE {
		return errorsx.InvalidArgument("岗位状态无效")
	}
	if basePost.Status == req.GetStatus() {
		return nil
	}
	return c.UpdateByID(ctx, &models.BasePost{ID: req.GetId(), Status: req.GetStatus()})
}

// resolveTenantID 解析岗位创建时的所属租户并校验租户范围。
func (c *BasePostCase) resolveTenantID(ctx context.Context, tenantID int64) (int64, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return 0, err
	}
	if tenantID == 0 {
		return authInfo.TenantId, nil
	}
	if authInfo.TenantCode != databaseGorm.DefaultTenantCode && tenantID != authInfo.TenantId {
		return 0, errorsx.PermissionDenied("不能操作其他租户的岗位")
	}
	return tenantID, nil
}
