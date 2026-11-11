package base

import (
	"context"

	"github.com/go-kratos/kratos/v3/log"
	basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"
	biz "github.com/liujitcn/kratos-admin/backend/internal/biz/base"
	"github.com/liujitcn/kratos-core/errorsx"
	"google.golang.org/protobuf/types/known/emptypb"
)

// AiPluginService 提供插件目录、租户策略和本人账号连接的传输适配。
type AiPluginService struct {
	basev1.UnimplementedAiPluginServiceServer
	aiPluginCase *biz.AiPluginCase
}

// NewAiPluginService 创建插件服务。
func NewAiPluginService(aiPluginCase *biz.AiPluginCase) *AiPluginService {
	return &AiPluginService{aiPluginCase: aiPluginCase}
}

// ListAiPlugin 查询当前身份可见的插件目录。
func (s *AiPluginService) ListAiPlugin(ctx context.Context, req *basev1.ListAiPluginRequest) (*basev1.ListAiPluginResponse, error) {
	response, err := s.aiPluginCase.ListAiPlugin(ctx, req)
	if err != nil {
		log.Error("ListAiPlugin", "error", err)
		return nil, errorsx.WrapInternal(err, "查询当前身份可见的插件目录失败")
	}
	return response, nil
}

// GetAiPlugin 查询插件详情。
func (s *AiPluginService) GetAiPlugin(ctx context.Context, req *basev1.GetAiPluginRequest) (*basev1.AiPlugin, error) {
	response, err := s.aiPluginCase.GetAiPlugin(ctx, req)
	if err != nil {
		log.Error("GetAiPlugin", "error", err)
		return nil, errorsx.WrapInternal(err, "查询插件详情失败")
	}
	return response, nil
}

// CreateAiPlugin 创建插件定义。
func (s *AiPluginService) CreateAiPlugin(ctx context.Context, req *basev1.CreateAiPluginRequest) (*basev1.AiPlugin, error) {
	response, err := s.aiPluginCase.CreateAiPlugin(ctx, req)
	if err != nil {
		log.Error("CreateAiPlugin", "error", err)
		return nil, errorsx.WrapInternal(err, "创建插件定义失败")
	}
	return response, nil
}

// UpdateAiPlugin 更新插件定义并递增修订号。
func (s *AiPluginService) UpdateAiPlugin(ctx context.Context, req *basev1.UpdateAiPluginRequest) (*basev1.AiPlugin, error) {
	response, err := s.aiPluginCase.UpdateAiPlugin(ctx, req)
	if err != nil {
		log.Error("UpdateAiPlugin", "error", err)
		return nil, errorsx.WrapInternal(err, "更新插件定义并递增修订号失败")
	}
	return response, nil
}

// DeleteAiPlugin 删除无有效引用的插件定义。
func (s *AiPluginService) DeleteAiPlugin(ctx context.Context, req *basev1.DeleteAiPluginRequest) (*emptypb.Empty, error) {
	response, err := s.aiPluginCase.DeleteAiPlugin(ctx, req)
	if err != nil {
		log.Error("DeleteAiPlugin", "error", err)
		return nil, errorsx.WrapInternal(err, "删除无有效引用的插件定义失败")
	}
	return response, nil
}

// SetAiPluginInstallation 设置当前租户安装和终端策略。
func (s *AiPluginService) SetAiPluginInstallation(ctx context.Context, req *basev1.SetAiPluginInstallationRequest) (*basev1.AiPluginInstallation, error) {
	response, err := s.aiPluginCase.SetAiPluginInstallation(ctx, req)
	if err != nil {
		log.Error("SetAiPluginInstallation", "error", err)
		return nil, errorsx.WrapInternal(err, "设置当前租户安装和终端策略失败")
	}
	return response, nil
}

// ListAiPluginTool 查询工具发现快照和审核状态。
func (s *AiPluginService) ListAiPluginTool(ctx context.Context, req *basev1.ListAiPluginToolRequest) (*basev1.ListAiPluginToolResponse, error) {
	response, err := s.aiPluginCase.ListAiPluginTool(ctx, req)
	if err != nil {
		log.Error("ListAiPluginTool", "error", err)
		return nil, errorsx.WrapInternal(err, "查询工具发现快照和审核状态失败")
	}
	return response, nil
}

// SetAiPluginToolPolicy 审核工具契约并设置租户策略。
func (s *AiPluginService) SetAiPluginToolPolicy(ctx context.Context, req *basev1.SetAiPluginToolPolicyRequest) (*basev1.AiPluginTool, error) {
	response, err := s.aiPluginCase.SetAiPluginToolPolicy(ctx, req)
	if err != nil {
		log.Error("SetAiPluginToolPolicy", "error", err)
		return nil, errorsx.WrapInternal(err, "审核工具契约并设置租户策略失败")
	}
	return response, nil
}

// ListAiPluginConnection 查询本人及授权共享连接。
func (s *AiPluginService) ListAiPluginConnection(ctx context.Context, req *basev1.ListAiPluginConnectionRequest) (*basev1.ListAiPluginConnectionResponse, error) {
	response, err := s.aiPluginCase.ListAiPluginConnection(ctx, req)
	if err != nil {
		log.Error("ListAiPluginConnection", "error", err)
		return nil, errorsx.WrapInternal(err, "查询本人及授权共享连接失败")
	}
	return response, nil
}

// CreateAiPluginConnection 创建本人连接并验证工具发现。
func (s *AiPluginService) CreateAiPluginConnection(ctx context.Context, req *basev1.CreateAiPluginConnectionRequest) (*basev1.AiPluginConnection, error) {
	response, err := s.aiPluginCase.CreateAiPluginConnection(ctx, req)
	if err != nil {
		log.Error("CreateAiPluginConnection", "error", err)
		return nil, errorsx.WrapInternal(err, "创建本人连接并验证工具发现失败")
	}
	return response, nil
}

// RefreshAiPluginConnection 检测连接并完整刷新工具。
func (s *AiPluginService) RefreshAiPluginConnection(ctx context.Context, req *basev1.RefreshAiPluginConnectionRequest) (*basev1.AiPluginConnection, error) {
	response, err := s.aiPluginCase.RefreshAiPluginConnection(ctx, req)
	if err != nil {
		log.Error("RefreshAiPluginConnection", "error", err)
		return nil, errorsx.WrapInternal(err, "检测连接并完整刷新工具失败")
	}
	return response, nil
}

// DisconnectAiPluginConnection 撤销本人连接并立即使旧许可失效。
func (s *AiPluginService) DisconnectAiPluginConnection(ctx context.Context, req *basev1.DisconnectAiPluginConnectionRequest) (*basev1.AiPluginConnection, error) {
	response, err := s.aiPluginCase.DisconnectAiPluginConnection(ctx, req)
	if err != nil {
		log.Error("DisconnectAiPluginConnection", "error", err)
		return nil, errorsx.WrapInternal(err, "撤销本人连接并立即使旧许可失效失败")
	}
	return response, nil
}

// StartAiPluginOAuth 开始账号授权。
func (s *AiPluginService) StartAiPluginOAuth(ctx context.Context, req *basev1.StartAiPluginOAuthRequest) (*basev1.StartAiPluginOAuthResponse, error) {
	response, err := s.aiPluginCase.StartAiPluginOAuth(ctx, req)
	if err != nil {
		log.Error("StartAiPluginOAuth", "error", err)
		return nil, errorsx.WrapInternal(err, "开始账号授权失败")
	}
	return response, nil
}

// CompleteAiPluginOAuth 一次性消费授权状态并交换令牌。
func (s *AiPluginService) CompleteAiPluginOAuth(ctx context.Context, req *basev1.CompleteAiPluginOAuthRequest) (*basev1.AiPluginConnection, error) {
	response, err := s.aiPluginCase.CompleteAiPluginOAuth(ctx, req)
	if err != nil {
		log.Error("CompleteAiPluginOAuth", "error", err)
		return nil, errorsx.WrapInternal(err, "一次性消费授权状态并交换令牌失败")
	}
	return response, nil
}

// ReauthorizeAiPluginConnection 在原账号内重新设置API令牌。
func (s *AiPluginService) ReauthorizeAiPluginConnection(ctx context.Context, req *basev1.ReauthorizeAiPluginConnectionRequest) (*basev1.AiPluginConnection, error) {
	response, err := s.aiPluginCase.ReauthorizeAiPluginConnection(ctx, req)
	if err != nil {
		log.Error("ReauthorizeAiPluginConnection", "error", err)
		return nil, errorsx.WrapInternal(err, "重新设置连接令牌失败")
	}
	return response, nil
}

// SetAiPluginConnectionSharing 显式设置本人账号的租户共享。
func (s *AiPluginService) SetAiPluginConnectionSharing(ctx context.Context, req *basev1.SetAiPluginConnectionSharingRequest) (*basev1.AiPluginConnection, error) {
	response, err := s.aiPluginCase.SetAiPluginConnectionSharing(ctx, req)
	if err != nil {
		log.Error("SetAiPluginConnectionSharing", "error", err)
		return nil, errorsx.WrapInternal(err, "设置账号共享失败")
	}
	return response, nil
}
