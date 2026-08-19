package admin

import (
	"context"
	"fmt"

	"github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin"
	"github.com/liujitcn/kratos-core/errorsx"

	"github.com/go-kratos/kratos/v3/log"
)

// BaseMigrationService 数据库迁移服务。
type BaseMigrationService struct {
	adminv1.UnimplementedBaseMigrationServiceServer
	baseMigrationCase *biz.BaseMigrationCase
}

// NewBaseMigrationService 创建数据库迁移服务。
func NewBaseMigrationService(baseMigrationCase *biz.BaseMigrationCase) *BaseMigrationService {
	return &BaseMigrationService{baseMigrationCase: baseMigrationCase}
}

// PageBaseMigration 分页查询数据库升级历史。
func (s *BaseMigrationService) PageBaseMigration(
	ctx context.Context,
	req *adminv1.PageBaseMigrationRequest,
) (*adminv1.PageBaseMigrationResponse, error) {
	res, err := s.baseMigrationCase.PageBaseMigration(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("PageBaseMigration %v", err))
		return nil, errorsx.WrapInternal(err, "查询数据库升级历史失败")
	}
	return res, nil
}

// GetBaseMigration 查询数据库迁移记录详情。
func (s *BaseMigrationService) GetBaseMigration(
	ctx context.Context,
	req *adminv1.GetBaseMigrationRequest,
) (*adminv1.BaseMigration, error) {
	res, err := s.baseMigrationCase.GetBaseMigration(ctx, req.GetId())
	if err != nil {
		log.Error(fmt.Sprintf("GetBaseMigration %v", err))
		return nil, errorsx.WrapInternal(err, "查询数据库迁移记录详情失败")
	}
	return res, nil
}
