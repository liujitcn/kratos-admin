package admin

import (
	"context"
	"fmt"

	systemadminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/core/pkg/errorsx"
	biz "github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin/v1"

	"github.com/go-kratos/kratos/v3/log"
)

// BaseTranslationService 管理动态资源翻译草稿。
type BaseTranslationService struct {
	systemadminv1.UnimplementedBaseTranslationServiceServer
	translationCase *biz.BaseTranslationCase
}

// NewBaseTranslationService 创建动态资源翻译服务。
func NewBaseTranslationService(translationCase *biz.BaseTranslationCase) *BaseTranslationService {
	return &BaseTranslationService{translationCase: translationCase}
}

// GenerateTranslationDraft 为已保存资源生成机器翻译草稿。
func (s *BaseTranslationService) GenerateTranslationDraft(ctx context.Context, req *systemadminv1.GenerateTranslationDraftRequest) (*systemadminv1.GenerateTranslationDraftResponse, error) {
	response, err := s.translationCase.GenerateTranslationDraft(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("GenerateTranslationDraft %v", err))
		return nil, errorsx.WrapInternal(err, "生成翻译草稿失败")
	}
	return response, nil
}
