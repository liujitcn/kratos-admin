package admin

import (
	"context"
	"fmt"

	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	biz "github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin"
	"github.com/liujitcn/kratos-core/errorsx"

	"github.com/go-kratos/kratos/v3/log"
	"github.com/liujitcn/kratos-kit/auth/authn/engine"
	"google.golang.org/protobuf/types/known/emptypb"
)

// OauthClientService 开放授权客户端管理服务。
type OauthClientService struct {
	adminv1.UnimplementedOauthClientServiceServer
	oauthClientCase *biz.OauthClientCase
	authenticator   engine.Authenticator
}

// NewOauthClientService 创建开放授权客户端管理服务。
func NewOauthClientService(oauthClientCase *biz.OauthClientCase, authenticator engine.Authenticator) *OauthClientService {
	return &OauthClientService{oauthClientCase: oauthClientCase, authenticator: authenticator}
}

// OptionOauthClientApi 查询可授权的开发接口选项。
func (s *OauthClientService) OptionOauthClientApi(ctx context.Context, req *adminv1.OptionOauthClientApiRequest) (*adminv1.OptionOauthClientApiResponse, error) {
	res, err := s.oauthClientCase.OptionOauthClientAPI(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("OptionOauthClientApi %v", err))
		return nil, errorsx.WrapInternal(err, "查询可授权接口失败")
	}
	return res, nil
}

// PageOauthClient 分页查询开放授权客户端。
func (s *OauthClientService) PageOauthClient(ctx context.Context, req *adminv1.PageOauthClientRequest) (*adminv1.PageOauthClientResponse, error) {
	res, err := s.oauthClientCase.PageOauthClient(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("PageOauthClient %v", err))
		return nil, errorsx.WrapInternal(err, "分页查询开放授权客户端失败")
	}
	return res, nil
}

// GetOauthClient 查询开放授权客户端详情。
func (s *OauthClientService) GetOauthClient(ctx context.Context, req *adminv1.GetOauthClientRequest) (*adminv1.OauthClientForm, error) {
	res, err := s.oauthClientCase.GetOauthClient(ctx, req.GetId())
	if err != nil {
		log.Error(fmt.Sprintf("GetOauthClient %v", err))
		return nil, errorsx.WrapInternal(err, "查询开放授权客户端详情失败")
	}
	return res, nil
}

// GetOauthClientCredentials 查询开放授权客户端非敏感凭据元数据。
func (s *OauthClientService) GetOauthClientCredentials(ctx context.Context, req *adminv1.GetOauthClientCredentialsRequest) (*adminv1.OauthClientCredentials, error) {
	res, err := s.oauthClientCase.GetOauthClientCredentials(ctx, req.GetId())
	if err != nil {
		log.Error(fmt.Sprintf("GetOauthClientCredentials %v", err))
		return nil, errorsx.WrapInternal(err, "查询开放授权客户端凭据失败")
	}
	return res, nil
}

// RotateOauthClientCredentials 轮换开放授权客户端凭据并返回一次性明文结果。
func (s *OauthClientService) RotateOauthClientCredentials(ctx context.Context, req *adminv1.RotateOauthClientCredentialsRequest) (*adminv1.OauthClientCredentials, error) {
	res, err := s.oauthClientCase.RotateOauthClientCredentials(ctx, req.GetId())
	if err != nil {
		log.Error(fmt.Sprintf("RotateOauthClientCredentials %v", err))
		return nil, errorsx.WrapInternal(err, "轮换开放授权客户端凭据失败")
	}
	return res, nil
}

// CreateOauthClient 创建开放授权客户端。
func (s *OauthClientService) CreateOauthClient(ctx context.Context, req *adminv1.CreateOauthClientRequest) (*emptypb.Empty, error) {
	err := s.oauthClientCase.CreateOauthClient(ctx, req.GetOauthClient())
	if err != nil {
		log.Error(fmt.Sprintf("CreateOauthClient %v", err))
		return nil, errorsx.WrapInternal(err, "创建开放授权客户端失败")
	}
	return new(emptypb.Empty), nil
}

// UpdateOauthClient 更新开放授权客户端。
func (s *OauthClientService) UpdateOauthClient(ctx context.Context, req *adminv1.UpdateOauthClientRequest) (*emptypb.Empty, error) {
	err := s.oauthClientCase.UpdateOauthClient(ctx, req.GetOauthClient())
	if err != nil {
		log.Error(fmt.Sprintf("UpdateOauthClient %v", err))
		return nil, errorsx.WrapInternal(err, "更新开放授权客户端失败")
	}
	return new(emptypb.Empty), nil
}

// DeleteOauthClient 删除开放授权客户端。
func (s *OauthClientService) DeleteOauthClient(ctx context.Context, req *adminv1.DeleteOauthClientRequest) (*emptypb.Empty, error) {
	err := s.oauthClientCase.DeleteOauthClient(ctx, req.GetId())
	if err != nil {
		log.Error(fmt.Sprintf("DeleteOauthClient %v", err))
		return nil, errorsx.WrapInternal(err, "删除开放授权客户端失败")
	}
	return new(emptypb.Empty), nil
}

// SetOauthClientStatus 设置开放授权客户端状态。
func (s *OauthClientService) SetOauthClientStatus(ctx context.Context, req *adminv1.SetOauthClientStatusRequest) (*emptypb.Empty, error) {
	err := s.oauthClientCase.SetOauthClientStatus(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("SetOauthClientStatus %v", err))
		return nil, errorsx.WrapInternal(err, "设置开放授权客户端状态失败")
	}
	return new(emptypb.Empty), nil
}
