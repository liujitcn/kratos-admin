package base

import (
	"context"
	"fmt"

	basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"
	biz "github.com/liujitcn/kratos-admin/backend/internal/biz/base"
	"github.com/liujitcn/kratos-core/errorsx"

	"github.com/go-kratos/kratos/v3/log"
)

// AiSearchService AI 助手联网搜索服务
type AiSearchService struct {
	basev1.UnimplementedAiSearchServiceServer
	aiSearchCase *biz.AiSearchCase
}

// NewAiSearchService 创建 AI 助手联网搜索服务
func NewAiSearchService(aiSearchCase *biz.AiSearchCase) *AiSearchService {
	return &AiSearchService{aiSearchCase: aiSearchCase}
}

// SearchAiWeb 联网搜索公开信息并返回结果摘要
func (s *AiSearchService) SearchAiWeb(ctx context.Context, req *basev1.SearchAiWebRequest) (*basev1.SearchAiWebResponse, error) {
	resp, err := s.aiSearchCase.SearchAiWeb(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("SearchAiWeb %v", err))
		return nil, errorsx.WrapInternal(err, "联网搜索失败")
	}
	return resp, nil
}
