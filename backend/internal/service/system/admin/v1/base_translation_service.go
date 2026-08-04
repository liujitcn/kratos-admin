package admin

import (
	"context"
	"fmt"

	systemadminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/core/pkg/errorsx"
	biz "github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin/v1"

	"github.com/go-kratos/kratos/v3/log"
	"google.golang.org/protobuf/types/known/emptypb"
)

// BaseTranslationService 管理单条动态资源翻译。
type BaseTranslationService struct {
	systemadminv1.UnimplementedBaseTranslationServiceServer
	translationCase *biz.BaseTranslationCase
}

// NewBaseTranslationService 创建动态资源翻译服务。
func NewBaseTranslationService(translationCase *biz.BaseTranslationCase) *BaseTranslationService {
	return &BaseTranslationService{translationCase: translationCase}
}

// DraftBaseTranslation 翻译请求中的单个文本。
func (s *BaseTranslationService) DraftBaseTranslation(ctx context.Context, req *systemadminv1.DraftBaseTranslationRequest) (*systemadminv1.DraftBaseTranslationResponse, error) {
	response, err := s.translationCase.DraftBaseTranslation(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("DraftBaseTranslation %v", err))
		return nil, errorsx.WrapInternal(err, "生成翻译失败")
	}
	return response, nil
}

// UpdateBaseTranslation 修改或新增单个翻译信息，空文本时由系统补充机器译文。
func (s *BaseTranslationService) UpdateBaseTranslation(ctx context.Context, req *systemadminv1.UpdateBaseTranslationRequest) (*emptypb.Empty, error) {
	err := s.translationCase.UpdateBaseTranslation(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("UpdateBaseTranslation %v", err))
		return nil, errorsx.WrapInternal(err, "修改翻译信息失败")
	}
	return &emptypb.Empty{}, nil
}
