package base

import (
	"context"
	"fmt"

	basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"
	biz "github.com/liujitcn/kratos-admin/backend/internal/biz/base"
	"github.com/liujitcn/kratos-core/errorsx"

	"github.com/go-kratos/kratos/v3/log"
)

// OauthClientService 开放授权客户端令牌服务。
type OauthClientService struct {
	basev1.UnimplementedOauthClientServiceServer
	oauthClientTokenCase *biz.OauthClientTokenCase
}

// NewOauthClientService 创建开放授权客户端令牌服务。
func NewOauthClientService(oauthClientTokenCase *biz.OauthClientTokenCase) *OauthClientService {
	return &OauthClientService{oauthClientTokenCase: oauthClientTokenCase}
}

// IssueOauthClientToken 使用客户端凭据换取访问令牌。
func (s *OauthClientService) IssueOauthClientToken(ctx context.Context, req *basev1.IssueOauthClientTokenRequest) (*basev1.IssueOauthClientTokenResponse, error) {
	res, err := s.oauthClientTokenCase.IssueOauthClientToken(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("IssueOauthClientToken %v", err))
		return nil, errorsx.WrapInternal(err, "签发客户端访问令牌失败")
	}
	return res, nil
}
