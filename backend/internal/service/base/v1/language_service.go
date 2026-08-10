package base

import (
	"context"
	"fmt"

	basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"
	biz "github.com/liujitcn/kratos-admin/backend/internal/biz/base"
	"github.com/liujitcn/kratos-core/pkg/errorsx"

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

// OptionLanguage 查询当前支持的语言选项。
func (s *LanguageService) OptionLanguage(ctx context.Context, req *basev1.OptionLanguageRequest) (*basev1.OptionLanguageResponse, error) {
	resp, err := s.languageCase.OptionLanguage(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("OptionLanguage %v", err))
		return nil, errorsx.WrapInternal(err, "查询语言失败")
	}
	return resp, nil
}
