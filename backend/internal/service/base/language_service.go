package base

import (
	"context"
	"fmt"

	basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"
	"github.com/liujitcn/kratos-admin/backend/core/pkg/errorsx"
	biz "github.com/liujitcn/kratos-admin/backend/internal/biz/base"

	"github.com/go-kratos/kratos/v3/log"
)

// LanguageService 语言公共查询服务。
type LanguageService struct {
	basev1.UnimplementedLanguageServiceServer
	languageCase *biz.LanguageCase
}

// NewLanguageService 创建语言公共查询服务。
func NewLanguageService(languageCase *biz.LanguageCase) *LanguageService {
	return &LanguageService{languageCase: languageCase}
}

// GetLanguage 查询当前支持的语言和主语言。
func (s *LanguageService) GetLanguage(ctx context.Context, req *basev1.GetLanguageRequest) (*basev1.GetLanguageResponse, error) {
	resp, err := s.languageCase.GetLanguage(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("GetLanguage %v", err))
		return nil, errorsx.WrapInternal(err, "查询语言失败")
	}
	return resp, nil
}
