package biz

import (
	"context"
	"path"

	"github.com/liujitcn/go-utils/mapper"
	"github.com/liujitcn/gorm-kit/repository"
	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	"github.com/liujitcn/kratos-core/biz"
	"github.com/liujitcn/kratos-core/errorsx"
	databaseGorm "github.com/liujitcn/kratos-kit/database/gorm"
	"gorm.io/gen/field"
)

// BaseFileCase 提供文件资产元数据查询和删除能力。
type BaseFileCase struct {
	*biz.BaseCase
	baseFileRepo *data.BaseFileRepository
	mapper       *mapper.CopierMapper[adminv1.BaseFile, models.BaseFile]
}

// NewBaseFileCase 创建文件资产业务实例。
func NewBaseFileCase(baseCase *biz.BaseCase, baseFileRepo *data.BaseFileRepository) *BaseFileCase {
	return &BaseFileCase{
		BaseCase:     baseCase,
		baseFileRepo: baseFileRepo,
		mapper:       mapper.NewCopierMapper[adminv1.BaseFile, models.BaseFile](),
	}
}

// PageBaseFile 分页查询文件资产元数据。
func (c *BaseFileCase) PageBaseFile(ctx context.Context, req *adminv1.PageBaseFileRequest) (*adminv1.PageBaseFileResponse, error) {
	tenantID, err := c.resolveFileTenant(ctx, req.GetTenantId())
	if err != nil {
		return nil, err
	}
	query := c.baseFileRepo.Query(ctx).BaseFile
	opts := []repository.QueryOption{repository.Order(query.CreatedAt.Desc()), repository.Order(query.ID.Desc())}
	if tenantID > 0 {
		opts = append(opts, repository.Where(query.TenantID.Eq(tenantID)))
	}
	if keyword := req.GetKeyword(); keyword != "" {
		pattern := "%" + keyword + "%"
		opts = append(opts, repository.Where(field.Or(query.FileName.Like(pattern), query.ContentHash.Like(pattern))))
	}
	if extension := req.GetExtension(); extension != "" {
		opts = append(opts, repository.Where(query.Extension.Eq(extension)))
	}
	var list []*models.BaseFile
	var total int64
	list, total, err = c.baseFileRepo.Page(ctx, req.GetPageNum(), req.GetPageSize(), opts...)
	if err != nil {
		return nil, err
	}
	items := make([]*adminv1.BaseFile, 0, len(list))
	for _, item := range list {
		items = append(items, c.toBaseFile(item))
	}
	return &adminv1.PageBaseFileResponse{BaseFiles: items, Total: total}, nil
}

// GetBaseFile 查询文件资产详情。
func (c *BaseFileCase) GetBaseFile(ctx context.Context, id int64) (*adminv1.BaseFile, error) {
	tenantID, err := c.resolveFileTenant(ctx, 0)
	if err != nil {
		return nil, err
	}
	query := c.baseFileRepo.Query(ctx).BaseFile
	opts := []repository.QueryOption{repository.Where(query.ID.Eq(id))}
	if tenantID > 0 {
		opts = append(opts, repository.Where(query.TenantID.Eq(tenantID)))
	}
	var item *models.BaseFile
	item, err = c.baseFileRepo.Find(ctx, opts...)
	if err != nil {
		return nil, err
	}
	return c.toBaseFile(item), nil
}

// DeleteBaseFile 删除文件对象及其元数据。
func (c *BaseFileCase) DeleteBaseFile(ctx context.Context, id int64) error {
	tenantID, err := c.resolveFileTenant(ctx, 0)
	if err != nil {
		return err
	}
	query := c.baseFileRepo.Query(ctx).BaseFile
	opts := []repository.QueryOption{repository.Where(query.ID.Eq(id))}
	if tenantID > 0 {
		opts = append(opts, repository.Where(query.TenantID.Eq(tenantID)))
	}
	var item *models.BaseFile
	item, err = c.baseFileRepo.Find(ctx, opts...)
	if err != nil {
		return err
	}
	objectPath := path.Join(item.FileDirectory, item.SaveFileName)
	if err = c.OSS.DeleteFile(objectPath); err != nil {
		return errorsx.Internal("删除文件对象失败").WithCause(err)
	}
	if err = c.baseFileRepo.DeleteByIDs(ctx, []int64{id}); err != nil {
		return err
	}
	return nil
}

// resolveFileTenant 将普通租户限制在自身文件范围，默认租户保留跨租户查询能力。
func (c *BaseFileCase) resolveFileTenant(ctx context.Context, requested int64) (int64, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return 0, err
	}
	if requested > 0 && authInfo.TenantCode != databaseGorm.DefaultTenantCode && requested != authInfo.TenantId {
		return 0, errorsx.PermissionDenied("无权访问其他租户文件")
	}
	if authInfo.TenantCode != databaseGorm.DefaultTenantCode {
		return authInfo.TenantId, nil
	}
	return requested, nil
}

// toBaseFile 将文件模型转换为接口响应并保留文件大小精度。
func (c *BaseFileCase) toBaseFile(item *models.BaseFile) *adminv1.BaseFile {
	result := c.mapper.ToDTO(item)
	if result == nil {
		return nil
	}
	result.Size = item.Size
	return result
}
