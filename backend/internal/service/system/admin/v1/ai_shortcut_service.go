package admin

import (
	"context"

	"github.com/go-kratos/kratos/v3/log"
	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	biz "github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin"
	"github.com/liujitcn/kratos-core/errorsx"
	"google.golang.org/protobuf/types/known/emptypb"
)

// AiShortcutService 提供管理端 AI 快捷入口接口。
type AiShortcutService struct {
	adminv1.UnimplementedAiShortcutServiceServer
	aiShortcutCase *biz.AiShortcutCase
}

// NewAiShortcutService 创建管理端快捷入口服务。
func NewAiShortcutService(aiShortcutCase *biz.AiShortcutCase) *AiShortcutService {
	return &AiShortcutService{aiShortcutCase: aiShortcutCase}
}

// PageAiShortcut 分页查询当前租户的快捷入口。
func (s *AiShortcutService) PageAiShortcut(ctx context.Context, req *adminv1.PageAiShortcutRequest) (*adminv1.PageAiShortcutResponse, error) {
	result, err := s.aiShortcutCase.PageAiShortcut(ctx, req)
	if err != nil {
		log.Error("PageAiShortcut", "error", err)
		return nil, errorsx.WrapInternal(err, "查询快捷入口列表失败")
	}
	return result, nil
}

// GetAiShortcut 查看入口完整配置。
func (s *AiShortcutService) GetAiShortcut(ctx context.Context, req *adminv1.GetAiShortcutRequest) (*adminv1.AiShortcutForm, error) {
	result, err := s.aiShortcutCase.GetAiShortcut(ctx, req.GetId())
	if err != nil {
		log.Error("GetAiShortcut", "error", err)
		return nil, errorsx.WrapInternal(err, "查询快捷入口失败")
	}
	return result, nil
}

// OptionAiShortcutTool 查询当前终端的可选 API 工具。
func (s *AiShortcutService) OptionAiShortcutTool(ctx context.Context, req *adminv1.OptionAiShortcutToolRequest) (*adminv1.OptionAiShortcutToolResponse, error) {
	result, err := s.aiShortcutCase.OptionAiShortcutTool(ctx, req.GetTerminal())
	if err != nil {
		log.Error("OptionAiShortcutTool", "error", err)
		return nil, errorsx.WrapInternal(err, "查询快捷入口工具失败")
	}
	return result, nil
}

// CreateAiShortcut 创建快捷入口。
func (s *AiShortcutService) CreateAiShortcut(ctx context.Context, req *adminv1.CreateAiShortcutRequest) (*emptypb.Empty, error) {
	err := s.aiShortcutCase.CreateAiShortcut(ctx, req.GetShortcut())
	if err != nil {
		log.Error("CreateAiShortcut", "error", err)
		return nil, errorsx.WrapInternal(err, "创建快捷入口失败")
	}
	return &emptypb.Empty{}, nil
}

// UpdateAiShortcut 更新快捷入口。
func (s *AiShortcutService) UpdateAiShortcut(ctx context.Context, req *adminv1.UpdateAiShortcutRequest) (*emptypb.Empty, error) {
	err := s.aiShortcutCase.UpdateAiShortcut(ctx, req.GetShortcut())
	if err != nil {
		log.Error("UpdateAiShortcut", "error", err)
		return nil, errorsx.WrapInternal(err, "更新快捷入口失败")
	}
	return &emptypb.Empty{}, nil
}

// DeleteAiShortcut 删除指定快捷入口。
func (s *AiShortcutService) DeleteAiShortcut(ctx context.Context, req *adminv1.DeleteAiShortcutRequest) (*emptypb.Empty, error) {
	err := s.aiShortcutCase.DeleteAiShortcut(ctx, req.GetId())
	if err != nil {
		log.Error("DeleteAiShortcut", "error", err)
		return nil, errorsx.WrapInternal(err, "删除快捷入口失败")
	}
	return &emptypb.Empty{}, nil
}

// SetAiShortcutStatus 切换快捷入口状态。
func (s *AiShortcutService) SetAiShortcutStatus(ctx context.Context, req *adminv1.SetAiShortcutStatusRequest) (*emptypb.Empty, error) {
	err := s.aiShortcutCase.SetAiShortcutStatus(ctx, req)
	if err != nil {
		log.Error("SetAiShortcutStatus", "error", err)
		return nil, errorsx.WrapInternal(err, "修改快捷入口状态失败")
	}
	return &emptypb.Empty{}, nil
}
