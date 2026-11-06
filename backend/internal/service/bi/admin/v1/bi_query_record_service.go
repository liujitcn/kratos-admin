package admin

import (
	"context"
	"fmt"

	"github.com/go-kratos/kratos/v3/log"
	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/bi/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/bi/admin"
	"github.com/liujitcn/kratos-core/errorsx"
)

// BiQueryRecordService 智能问数记录管理服务。
type BiQueryRecordService struct {
	adminv1.UnimplementedBiQueryRecordServiceServer
	biQueryRecordCase *admin.BiQueryRecordCase
}

// NewBiQueryRecordService 创建智能问数记录管理服务。
func NewBiQueryRecordService(biQueryRecordCase *admin.BiQueryRecordCase) *BiQueryRecordService {
	return &BiQueryRecordService{biQueryRecordCase: biQueryRecordCase}
}

// PageBiQuery 分页查询智能问数记录。
func (s *BiQueryRecordService) PageBiQuery(ctx context.Context, req *adminv1.PageBiQueryRequest) (*adminv1.PageBiQueryResponse, error) {
	result, err := s.biQueryRecordCase.PageBiQuery(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("PageBiQuery %v", err))
		return nil, errorsx.WrapInternal(err, "查询智能问数记录失败")
	}
	return result, nil
}

// GetBiQuery 查询智能问数记录详情。
func (s *BiQueryRecordService) GetBiQuery(ctx context.Context, req *adminv1.GetBiQueryRequest) (*adminv1.BiQueryRecord, error) {
	result, err := s.biQueryRecordCase.GetBiQuery(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("GetBiQuery %v", err))
		return nil, errorsx.WrapInternal(err, "查询智能问数记录详情失败")
	}
	return result, nil
}
