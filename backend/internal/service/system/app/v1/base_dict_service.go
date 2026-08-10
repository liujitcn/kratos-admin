package app

import (
	"context"
	"fmt"

	systemappv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/app/v1"
	biz "github.com/liujitcn/kratos-admin/backend/internal/biz/system/app"
	"github.com/liujitcn/kratos-core/pkg/errorsx"

	"github.com/go-kratos/kratos/v3/log"
)

// BaseDictService 字典服务
type BaseDictService struct {
	systemappv1.UnimplementedBaseDictServiceServer
	baseDictCase *biz.BaseDictCase
}

// NewBaseDictService 创建字典服务
func NewBaseDictService(
	baseDictCase *biz.BaseDictCase,
) *BaseDictService {
	var ss = BaseDictService{
		baseDictCase: baseDictCase,
	}
	return &ss
}

// GetBaseDict 查询字典
func (s *BaseDictService) GetBaseDict(ctx context.Context, req *systemappv1.GetBaseDictRequest) (*systemappv1.BaseDictForm, error) {
	res, err := s.baseDictCase.GetBaseDict(ctx, req.GetCode())
	if err != nil {
		log.Error(fmt.Sprintf("GetBaseDict %v", err))
		return nil, errorsx.WrapInternal(err, "查询失败")
	}
	return res, nil
}
