package admin

import (
	"context"
	"fmt"

	"github.com/go-kratos/kratos/v3/log"
	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/bi/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/bi/admin"
	"github.com/liujitcn/kratos-core/errorsx"
)

// BiSourceService 提供 BI 数据源配置管理服务。
type BiSourceService struct {
	adminv1.UnimplementedBiSourceServiceServer
	biSourceCase *admin.BiSourceCase
}

// NewBiSourceService 创建 BI 数据源配置服务。
func NewBiSourceService(biSourceCase *admin.BiSourceCase) *BiSourceService {
	return &BiSourceService{biSourceCase: biSourceCase}
}

// PageBiSource 分页查询 BI 数据源配置。
func (s *BiSourceService) PageBiSource(ctx context.Context, req *adminv1.PageBiSourceRequest) (*adminv1.PageBiSourceResponse, error) {
	result, err := s.biSourceCase.PageBiSource(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("PageBiSource %v", err))
		return nil, errorsx.WrapInternal(err, "查询 BI 数据源失败")
	}
	return result, nil
}

// GetBiSource 查询 BI 数据源配置详情。
func (s *BiSourceService) GetBiSource(ctx context.Context, req *adminv1.GetBiSourceRequest) (*adminv1.BiSource, error) {
	result, err := s.biSourceCase.GetBiSource(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("GetBiSource %v", err))
		return nil, errorsx.WrapInternal(err, "查询 BI 数据源详情失败")
	}
	return result, nil
}

// CreateBiSource 创建 BI 数据源配置。
func (s *BiSourceService) CreateBiSource(ctx context.Context, req *adminv1.CreateBiSourceRequest) (*adminv1.BiSource, error) {
	result, err := s.biSourceCase.CreateBiSource(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("CreateBiSource %v", err))
		return nil, errorsx.WrapInternal(err, "创建 BI 数据源失败")
	}
	return result, nil
}

// UpdateBiSource 更新 BI 数据源配置。
func (s *BiSourceService) UpdateBiSource(ctx context.Context, req *adminv1.UpdateBiSourceRequest) (*adminv1.BiSource, error) {
	result, err := s.biSourceCase.UpdateBiSource(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("UpdateBiSource %v", err))
		return nil, errorsx.WrapInternal(err, "更新 BI 数据源失败")
	}
	return result, nil
}

// DeleteBiSource 删除 BI 数据源配置。
func (s *BiSourceService) DeleteBiSource(ctx context.Context, req *adminv1.DeleteBiSourceRequest) (*adminv1.DeleteBiSourceResponse, error) {
	result, err := s.biSourceCase.DeleteBiSource(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("DeleteBiSource %v", err))
		return nil, errorsx.WrapInternal(err, "删除 BI 数据源失败")
	}
	return result, nil
}

// SetBiSourceStatus 修改 BI 数据源启用状态。
func (s *BiSourceService) SetBiSourceStatus(ctx context.Context, req *adminv1.SetBiSourceStatusRequest) (*adminv1.BiSource, error) {
	result, err := s.biSourceCase.SetBiSourceStatus(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("SetBiSourceStatus %v", err))
		return nil, errorsx.WrapInternal(err, "修改 BI 数据源状态失败")
	}
	return result, nil
}

// OptionBiSourceDatabase 查询服务端数据库客户端选项。
func (s *BiSourceService) OptionBiSourceDatabase(ctx context.Context, req *adminv1.OptionBiSourceDatabaseRequest) (*adminv1.OptionBiSourceDatabaseResponse, error) {
	result, err := s.biSourceCase.OptionBiSourceDatabase(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("OptionBiSourceDatabase %v", err))
		return nil, errorsx.WrapInternal(err, "查询 BI 数据库客户端失败")
	}
	return result, nil
}

// TestBiSource 测试 BI 数据源连接和表配置。
func (s *BiSourceService) TestBiSource(ctx context.Context, req *adminv1.TestBiSourceRequest) (*adminv1.TestBiSourceResponse, error) {
	result, err := s.biSourceCase.TestBiSource(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("TestBiSource %v", err))
		return nil, errorsx.WrapInternal(err, "测试 BI 数据源失败")
	}
	return result, nil
}

// ListBiSourceTables 查询 BI 数据源中的表和字段结构。
func (s *BiSourceService) ListBiSourceTables(ctx context.Context, req *adminv1.ListBiSourceTablesRequest) (*adminv1.ListBiSourceTablesResponse, error) {
	result, err := s.biSourceCase.ListBiSourceTables(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("ListBiSourceTables %v", err))
		return nil, errorsx.WrapInternal(err, "读取 BI 数据源表结构失败")
	}
	return result, nil
}

// SaveBiSourceModel 保存 BI 数据源语义模型。
func (s *BiSourceService) SaveBiSourceModel(ctx context.Context, req *adminv1.SaveBiSourceModelRequest) (*adminv1.BiSource, error) {
	result, err := s.biSourceCase.SaveBiSourceModel(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("SaveBiSourceModel %v", err))
		return nil, errorsx.WrapInternal(err, "保存 BI 数据源模型失败")
	}
	return result, nil
}
