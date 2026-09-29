package base

import (
	"context"
	"fmt"

	basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"
	biz "github.com/liujitcn/kratos-admin/backend/internal/biz/base"
	"github.com/liujitcn/kratos-core/errorsx"

	"github.com/go-kratos/kratos/v3/log"
)

// AiQueryService AI 助手智能问数服务
type AiQueryService struct {
	basev1.UnimplementedAiQueryServiceServer
	aiQueryCase *biz.AiQueryCase
}

// NewAiQueryService 创建 AI 助手智能问数服务
func NewAiQueryService(aiQueryCase *biz.AiQueryCase) *AiQueryService {
	return &AiQueryService{aiQueryCase: aiQueryCase}
}

// AskAiQuery 智能问数：将自然语言问题转换为只读 SQL 查询数据库并返回结构化结果
func (s *AiQueryService) AskAiQuery(ctx context.Context, req *basev1.AskAiQueryRequest) (*basev1.AskAiQueryResponse, error) {
	resp, err := s.aiQueryCase.AskAiQuery(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("AskAiQuery %v", err))
		return nil, errorsx.WrapInternal(err, "智能问数失败")
	}
	return resp, nil
}

// OptionAiQueryTable 查询智能问数可用的数据表目录
func (s *AiQueryService) OptionAiQueryTable(ctx context.Context, req *basev1.OptionAiQueryTableRequest) (*basev1.OptionAiQueryTableResponse, error) {
	resp, err := s.aiQueryCase.OptionAiQueryTable(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("OptionAiQueryTable %v", err))
		return nil, errorsx.WrapInternal(err, "查询智能问数表目录失败")
	}
	return resp, nil
}
