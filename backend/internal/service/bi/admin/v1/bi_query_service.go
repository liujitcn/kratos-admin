package admin

import (
	"context"
	"fmt"

	"github.com/go-kratos/kratos/v3/log"
	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/bi/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/bi/admin"
	"github.com/liujitcn/kratos-core/errorsx"
)

// BiQueryService 提供 BI 数据源目录和语义查询接口。
type BiQueryService struct {
	adminv1.UnimplementedBiQueryServiceServer
	biQueryCase *admin.BiQueryCase
}

// NewBiQueryService 创建 BI 语义查询服务。
func NewBiQueryService(biQueryCase *admin.BiQueryCase) *BiQueryService {
	return &BiQueryService{biQueryCase: biQueryCase}
}

// ListBiSources 查询当前用户可用的数据源目录。
func (s *BiQueryService) ListBiSources(ctx context.Context, req *adminv1.ListBiSourcesRequest) (*adminv1.ListBiSourcesResponse, error) {
	result, err := s.biQueryCase.ListBiSources(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("ListBiSources %v", err))
		return nil, errorsx.WrapInternal(err, "查询 BI 数据源失败")
	}
	return result, nil
}

// PreviewBiQuery 预览一个语义查询。
func (s *BiQueryService) PreviewBiQuery(ctx context.Context, req *adminv1.PreviewBiQueryRequest) (*adminv1.BiQueryResult, error) {
	result, err := s.biQueryCase.PreviewBiQuery(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("PreviewBiQuery %v", err))
		return nil, errorsx.WrapInternal(err, "预览 BI 查询失败")
	}
	return result, nil
}

// AskBiQuery 根据自然语言生成语义查询并预览。
func (s *BiQueryService) AskBiQuery(ctx context.Context, req *adminv1.AskBiQueryRequest) (*adminv1.AskBiQueryResponse, error) {
	result, err := s.biQueryCase.AskBiQuery(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("AskBiQuery %v", err))
		return nil, errorsx.WrapInternal(err, "BI 智能问数失败")
	}
	return result, nil
}
