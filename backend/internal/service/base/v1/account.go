package base

import (
	"context"
	"fmt"

	basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"
	biz "github.com/liujitcn/kratos-admin/backend/internal/biz/base"
	"github.com/liujitcn/kratos-core/errorsx"

	"github.com/go-kratos/kratos/v3/log"
	"google.golang.org/protobuf/types/known/emptypb"
)

// AccountService 账户安全公共服务。
type AccountService struct {
	basev1.UnimplementedAccountServiceServer
	accountCase *biz.AccountCase
}

// NewAccountService 创建账户安全公共服务。
func NewAccountService(accountCase *biz.AccountCase) *AccountService {
	return &AccountService{accountCase: accountCase}
}

// GetCurrentPasswordPolicy 获取当前用户生效的密码策略。
func (s *AccountService) GetCurrentPasswordPolicy(ctx context.Context, req *basev1.GetCurrentPasswordPolicyRequest) (*basev1.CurrentPasswordPolicy, error) {
	res, err := s.accountCase.GetCurrentPasswordPolicy(ctx)
	if err != nil {
		log.Error(fmt.Sprintf("GetCurrentPasswordPolicy %v", err))
		return nil, errorsx.WrapInternal(err, "获取密码策略失败")
	}
	return res, nil
}

// UpdateUserPassword 修改当前用户密码。
func (s *AccountService) UpdateUserPassword(ctx context.Context, req *basev1.UpdateUserPasswordRequest) (*emptypb.Empty, error) {
	err := s.accountCase.UpdateUserPassword(ctx, req.GetUserPassword())
	if err != nil {
		log.Error(fmt.Sprintf("UpdateUserPassword %v", err))
		return nil, errorsx.WrapInternal(err, "修改密码失败")
	}
	return new(emptypb.Empty), nil
}
