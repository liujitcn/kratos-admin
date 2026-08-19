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
	i18nCase *biz.BaseI18nCase
}

// NewBaseI18nService 创建动态资源翻译服务。
func NewBaseI18nService(i18nCase *biz.BaseI18nCase) *BaseI18nService {
	return &BaseI18nService{i18nCase: i18nCase}
}

// DraftBaseI18n 翻译请求中的单个文本。
func (s *BaseI18nService) DraftBaseI18n(ctx context.Context, req *adminv1.DraftBaseI18nRequest) (*adminv1.DraftBaseI18nResponse, error) {
	response, err := s.i18nCase.DraftBaseI18n(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("DraftBaseI18n %v", err))
		return nil, errorsx.WrapInternal(err, "生成翻译失败")
	}
	return response, nil
}

// UpdateBaseI18n 修改或新增单个翻译信息。
func (s *BaseI18nService) UpdateBaseI18n(ctx context.Context, req *adminv1.UpdateBaseI18nRequest) (*emptypb.Empty, error) {
	err := s.i18nCase.UpdateBaseI18n(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("UpdateBaseI18n %v", err))
		return nil, errorsx.WrapInternal(err, "修改翻译信息失败")
	}
	return &emptypb.Empty{}, nil
}
