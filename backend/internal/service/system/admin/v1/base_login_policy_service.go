package admin

import (
	"context"
	"fmt"

	"github.com/go-kratos/kratos/v3/log"
	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	biz "github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin"
	"github.com/liujitcn/kratos-core/errorsx"
)

// BaseLoginPolicyService 提供 Admin 登录来源策略接口。
type BaseLoginPolicyService struct {
	adminv1.UnimplementedBaseLoginPolicyServiceServer                          // 提供未实现 RPC 的默认返回。
	baseLoginPolicyCase                               *biz.BaseLoginPolicyCase // 负责平台权限和策略缓存刷新。
}

// NewBaseLoginPolicyService 创建登录来源策略服务。
func NewBaseLoginPolicyService(baseLoginPolicyCase *biz.BaseLoginPolicyCase) *BaseLoginPolicyService {
	return &BaseLoginPolicyService{baseLoginPolicyCase: baseLoginPolicyCase}
}

// GetBaseLoginPolicy 查询登录来源策略。
func (s *BaseLoginPolicyService) GetBaseLoginPolicy(ctx context.Context, req *adminv1.GetBaseLoginPolicyRequest) (*adminv1.BaseLoginPolicy, error) {
	res, err := s.baseLoginPolicyCase.GetBaseLoginPolicy(ctx)
	if err != nil {
		log.Error(fmt.Sprintf("GetBaseLoginPolicy %v", err))
		return nil, errorsx.WrapInternal(err, "查询登录来源策略失败")
	}
	return res, nil
}

// UpdateBaseLoginPolicy 更新登录来源策略。
func (s *BaseLoginPolicyService) UpdateBaseLoginPolicy(ctx context.Context, req *adminv1.UpdateBaseLoginPolicyRequest) (*adminv1.BaseLoginPolicy, error) {
	res, err := s.baseLoginPolicyCase.UpdateBaseLoginPolicy(ctx, req.GetPolicy())
	if err != nil {
		log.Error(fmt.Sprintf("UpdateBaseLoginPolicy %v", err))
		return nil, errorsx.WrapInternal(err, "更新登录来源策略失败")
	}
	return res, nil
}
