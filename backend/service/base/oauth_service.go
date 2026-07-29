package base

import (
	"context"
	"fmt"

	basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"
	"github.com/liujitcn/kratos-admin/backend/pkg/errorsx"
	"github.com/liujitcn/kratos-admin/backend/service/base/biz"

	"github.com/go-kratos/kratos/v3/log"
	kratosHTTP "github.com/go-kratos/kratos/v3/transport/http"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

const _ = grpc.SupportPackageIsVersion7

// OauthService 三方登录公共服务。
type OauthService struct {
	basev1.UnimplementedOauthServiceServer
	oauthCase *biz.OauthCase
}

// NewOauthService 创建三方登录公共服务。
func NewOauthService(oauthCase *biz.OauthCase) *OauthService {
	return &OauthService{
		oauthCase: oauthCase,
	}
}

// ListOauthBinding 查询个人中心三方账号绑定列表。
func (s *OauthService) ListOauthBinding(ctx context.Context, req *basev1.ListOauthBindingRequest) (*basev1.ListOauthBindingResponse, error) {
	res, err := s.oauthCase.ListOauthBinding(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("ListOauthBinding %v", err))
		return nil, errorsx.WrapInternal(err, "查询三方账号绑定失败")
	}
	return res, nil
}

// ListOauthProvider 查询三方登录方式。
func (s *OauthService) ListOauthProvider(ctx context.Context, req *basev1.ListOauthProviderRequest) (*basev1.ListOauthProviderResponse, error) {
	res, err := s.oauthCase.ListOauthProvider(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("ListOauthProvider %v", err))
		return nil, errorsx.WrapInternal(err, "查询三方登录方式失败")
	}
	return res, nil
}

// CreateOauthAuthorization 创建三方登录授权地址。
func (s *OauthService) CreateOauthAuthorization(ctx context.Context, req *basev1.CreateOauthAuthorizationRequest) (*basev1.CreateOauthAuthorizationResponse, error) {
	res, err := s.oauthCase.CreateOauthAuthorization(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("CreateOauthAuthorization %v", err))
		return nil, errorsx.WrapInternal(err, "创建三方登录授权失败")
	}
	return res, nil
}

// CreateOauthBindingAuthorization 创建个人中心三方账号绑定授权地址。
func (s *OauthService) CreateOauthBindingAuthorization(ctx context.Context, req *basev1.CreateOauthBindingAuthorizationRequest) (*basev1.CreateOauthBindingAuthorizationResponse, error) {
	res, err := s.oauthCase.CreateOauthBindingAuthorization(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("CreateOauthBindingAuthorization %v", err))
		return nil, errorsx.WrapInternal(err, "创建三方账号绑定授权失败")
	}
	return res, nil
}

// CreateOauthSession 创建三方登录会话。
func (s *OauthService) CreateOauthSession(ctx context.Context, req *basev1.CreateOauthSessionRequest) (*basev1.CreateOauthSessionResponse, error) {
	res, err := s.oauthCase.CreateOauthSession(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("CreateOauthSession %v", err))
		return nil, errorsx.WrapInternal(err, "创建三方登录会话失败")
	}
	return res, nil
}

// BindOauthSession 绑定已有账号并创建三方登录会话。
func (s *OauthService) BindOauthSession(ctx context.Context, req *basev1.BindOauthSessionRequest) (*basev1.CreateOauthSessionResponse, error) {
	res, err := s.oauthCase.BindOauthSession(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("BindOauthSession %v", err))
		return nil, errorsx.WrapInternal(err, "绑定三方账号失败")
	}
	return res, nil
}

// HandleOauthCallback 处理三方登录回调。
func (s *OauthService) HandleOauthCallback(ctx context.Context, req *basev1.HandleOauthCallbackRequest) (*basev1.HandleOauthCallbackResponse, error) {
	res, err := s.oauthCase.HandleOauthCallback(ctx, req)
	if err != nil {
		if _, ok := err.(kratosHTTP.Redirector); ok {
			return nil, err
		}
		log.Error(fmt.Sprintf("HandleOauthCallback %v", err))
		return nil, errorsx.WrapInternal(err, "处理三方登录回调失败")
	}
	return res, nil
}

// ExchangeOauthTicket 兑换三方登录票据。
func (s *OauthService) ExchangeOauthTicket(ctx context.Context, req *basev1.ExchangeOauthTicketRequest) (*basev1.ExchangeOauthTicketResponse, error) {
	res, err := s.oauthCase.ExchangeOauthTicket(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("ExchangeOauthTicket %v", err))
		return nil, errorsx.WrapInternal(err, "兑换三方登录票据失败")
	}
	return res, nil
}

// HandleOauthBindingCallback 处理个人中心三方账号绑定回调。
func (s *OauthService) HandleOauthBindingCallback(ctx context.Context, req *basev1.HandleOauthBindingCallbackRequest) (*basev1.HandleOauthBindingCallbackResponse, error) {
	err := s.oauthCase.HandleOauthBindingCallback(ctx, req)
	if err != nil {
		if _, ok := err.(kratosHTTP.Redirector); ok {
			return nil, err
		}
		log.Error(fmt.Sprintf("HandleOauthBindingCallback %v", err))
		return nil, errorsx.WrapInternal(err, "处理三方账号绑定回调失败")
	}
	return new(basev1.HandleOauthBindingCallbackResponse), nil
}

// UnbindOauthAccount 解绑个人中心三方账号。
func (s *OauthService) UnbindOauthAccount(ctx context.Context, req *basev1.UnbindOauthAccountRequest) (*emptypb.Empty, error) {
	err := s.oauthCase.UnbindOauthAccount(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("UnbindOauthAccount %v", err))
		return nil, errorsx.WrapInternal(err, "解绑三方账号失败")
	}
	return new(emptypb.Empty), nil
}
