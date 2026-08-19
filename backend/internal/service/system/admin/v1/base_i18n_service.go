package admin

import (
	"context"
	"fmt"

	"github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin"
	"github.com/liujitcn/kratos-core/errorsx"

	"github.com/go-kratos/kratos/v3/log"
	"google.golang.org/protobuf/types/known/emptypb"
)

// BaseI18nService 管理单条动态资源翻译。
type BaseI18nService struct {
	adminv1.UnimplementedBaseI18nServiceServer
	translationCase *biz.BaseTranslationCase
}

// NewBaseI18nService 创建动态资源翻译服务。
func NewBaseI18nService(translationCase *biz.BaseTranslationCase) *BaseI18nService {
	return &BaseI18nService{translationCase: translationCase}
}

// DraftBaseTranslation 翻译请求中的单个文本。
func (s *BaseI18nService) DraftBaseTranslation(ctx context.Context, req *adminv1.DraftBaseTranslationRequest) (*adminv1.DraftBaseTranslationResponse, error) {
	response, err := s.translationCase.DraftBaseTranslation(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("DraftBaseTranslation %v", err))
		return nil, errorsx.WrapInternal(err, "生成翻译失败")
	}
	return response, nil
}

// UpdateBaseTranslation 修改或新增单个翻译信息。
func (s *BaseI18nService) UpdateBaseTranslation(ctx context.Context, req *adminv1.UpdateBaseTranslationRequest) (*emptypb.Empty, error) {
	err := s.translationCase.UpdateBaseTranslation(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("UpdateBaseTranslation %v", err))
		return nil, errorsx.WrapInternal(err, "修改翻译信息失败")
	}
	return &emptypb.Empty{}, nil
}
