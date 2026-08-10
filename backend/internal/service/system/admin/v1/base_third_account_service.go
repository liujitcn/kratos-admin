package admin

import (
	"context"
	"fmt"

	systemadminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	biz "github.com/liujitcn/kratos-admin/backend/internal/biz/base"
	"github.com/liujitcn/kratos-core/pkg/errorsx"

	"github.com/go-kratos/kratos/v3/log"
	"google.golang.org/grpc"
)

const _ = grpc.SupportPackageIsVersion7

// BaseThirdAccountService Admin用户三方账号服务。
type BaseThirdAccountService struct {
	systemadminv1.UnimplementedBaseThirdAccountServiceServer
	baseThirdAccountCase *biz.BaseThirdAccountCase
}

// NewBaseThirdAccountService 创建Admin用户三方账号服务。
func NewBaseThirdAccountService(baseThirdAccountCase *biz.BaseThirdAccountCase) *BaseThirdAccountService {
	return &BaseThirdAccountService{baseThirdAccountCase: baseThirdAccountCase}
}

// GetBaseThirdAccountIdentifier 查询用户三方账号标识。
func (s *BaseThirdAccountService) GetBaseThirdAccountIdentifier(ctx context.Context, req *systemadminv1.GetBaseThirdAccountIdentifierRequest) (*systemadminv1.GetBaseThirdAccountIdentifierResponse, error) {
	account, err := s.baseThirdAccountCase.FindByUserProvider(ctx, req.GetUserId(), req.GetProvider())
	if err != nil {
		log.Error(fmt.Sprintf("GetBaseThirdAccountIdentifier %v", err))
		return nil, errorsx.WrapInternal(err, "查询用户三方账号标识失败")
	}
	return &systemadminv1.GetBaseThirdAccountIdentifierResponse{Identifier: account.Identifier}, nil
}
