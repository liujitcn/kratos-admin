package admin

import (
	"context"
	"fmt"

	"github.com/go-kratos/kratos/v3/log"
	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	biz "github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin"
	"github.com/liujitcn/kratos-core/errorsx"
)

// AiQueryService 智能问数记录管理服务。
type AiQueryService struct {
	adminv1.UnimplementedAiQueryServiceServer
	aiQueryCase *biz.AiQueryCase
}

// NewAiQueryService 创建智能问数记录管理服务。
func NewAiQueryService(aiQueryCase *biz.AiQueryCase) *AiQueryService {
	return &AiQueryService{aiQueryCase: aiQueryCase}
}

// PageAiQuery 分页查询智能问数记录。
func (s *AiQueryService) PageAiQuery(ctx context.Context, req *adminv1.PageAiQueryRequest) (*adminv1.PageAiQueryResponse, error) {
	result, err := s.aiQueryCase.PageAiQuery(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("PageAiQuery %v", err))
		return nil, errorsx.WrapInternal(err, "查询智能问数记录失败")
	}
	return result, nil
}

// GetAiQuery 查询智能问数记录详情。
func (s *AiQueryService) GetAiQuery(ctx context.Context, req *adminv1.GetAiQueryRequest) (*adminv1.AiQueryRecord, error) {
	result, err := s.aiQueryCase.GetAiQuery(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("GetAiQuery %v", err))
		return nil, errorsx.WrapInternal(err, "查询智能问数记录详情失败")
	}
	return result, nil
}
