package base

import (
	"context"

	"github.com/go-kratos/kratos/v3/log"
	basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"
	biz "github.com/liujitcn/kratos-admin/backend/internal/biz/base"
	"github.com/liujitcn/kratos-core/errorsx"
)

// AiAutomationService 提供本人自动化计划和执行服务。
type AiAutomationService struct {
	basev1.UnimplementedAiAutomationServiceServer
	aiAutomationCase *biz.AiAutomationCase
}

// NewAiAutomationService 创建自动化协议适配器。
func NewAiAutomationService(c *biz.AiAutomationCase) *AiAutomationService {
	return &AiAutomationService{aiAutomationCase: c}
}

// ListAiAutomation 调用自动化业务并保留业务错误分类。
func (s *AiAutomationService) ListAiAutomation(ctx context.Context, req *basev1.ListAiAutomationRequest) (*basev1.ListAiAutomationResponse, error) {
	response, err := s.aiAutomationCase.ListAiAutomation(ctx, req)
	if err != nil {
		log.Error("ListAiAutomation", "error", err)
		return nil, errorsx.WrapInternal(err, "AI自动化操作失败")
	}
	return response, nil
}

// GetAiAutomation 调用自动化业务并保留业务错误分类。
func (s *AiAutomationService) GetAiAutomation(ctx context.Context, req *basev1.GetAiAutomationRequest) (*basev1.AiAutomation, error) {
	response, err := s.aiAutomationCase.GetAiAutomation(ctx, req)
	if err != nil {
		log.Error("GetAiAutomation", "error", err)
		return nil, errorsx.WrapInternal(err, "AI自动化操作失败")
	}
	return response, nil
}

// PreviewAiAutomationSchedule 调用自动化业务并保留业务错误分类。
func (s *AiAutomationService) PreviewAiAutomationSchedule(ctx context.Context, req *basev1.PreviewAiAutomationScheduleRequest) (*basev1.PreviewAiAutomationScheduleResponse, error) {
	response, err := s.aiAutomationCase.PreviewAiAutomationSchedule(ctx, req)
	if err != nil {
		log.Error("PreviewAiAutomationSchedule", "error", err)
		return nil, errorsx.WrapInternal(err, "AI自动化操作失败")
	}
	return response, nil
}

// CreateAiAutomation 调用自动化业务并保留业务错误分类。
func (s *AiAutomationService) CreateAiAutomation(ctx context.Context, req *basev1.CreateAiAutomationRequest) (*basev1.AiAutomation, error) {
	response, err := s.aiAutomationCase.CreateAiAutomation(ctx, req)
	if err != nil {
		log.Error("CreateAiAutomation", "error", err)
		return nil, errorsx.WrapInternal(err, "AI自动化操作失败")
	}
	return response, nil
}

// UpdateAiAutomation 调用自动化业务并保留业务错误分类。
func (s *AiAutomationService) UpdateAiAutomation(ctx context.Context, req *basev1.UpdateAiAutomationRequest) (*basev1.AiAutomation, error) {
	response, err := s.aiAutomationCase.UpdateAiAutomation(ctx, req)
	if err != nil {
		log.Error("UpdateAiAutomation", "error", err)
		return nil, errorsx.WrapInternal(err, "AI自动化操作失败")
	}
	return response, nil
}

// SetAiAutomationStatus 调用自动化业务并保留业务错误分类。
func (s *AiAutomationService) SetAiAutomationStatus(ctx context.Context, req *basev1.SetAiAutomationStatusRequest) (*basev1.AiAutomation, error) {
	response, err := s.aiAutomationCase.SetAiAutomationStatus(ctx, req)
	if err != nil {
		log.Error("SetAiAutomationStatus", "error", err)
		return nil, errorsx.WrapInternal(err, "AI自动化操作失败")
	}
	return response, nil
}

// RunAiAutomation 调用自动化业务并保留业务错误分类。
func (s *AiAutomationService) RunAiAutomation(ctx context.Context, req *basev1.RunAiAutomationRequest) (*basev1.AiAutomationRun, error) {
	response, err := s.aiAutomationCase.RunAiAutomation(ctx, req)
	if err != nil {
		log.Error("RunAiAutomation", "error", err)
		return nil, errorsx.WrapInternal(err, "AI自动化操作失败")
	}
	return response, nil
}

// ListAiAutomationRun 调用自动化业务并保留业务错误分类。
func (s *AiAutomationService) ListAiAutomationRun(ctx context.Context, req *basev1.ListAiAutomationRunRequest) (*basev1.ListAiAutomationRunResponse, error) {
	response, err := s.aiAutomationCase.ListAiAutomationRun(ctx, req)
	if err != nil {
		log.Error("ListAiAutomationRun", "error", err)
		return nil, errorsx.WrapInternal(err, "AI自动化操作失败")
	}
	return response, nil
}

// GetAiAutomationRun 调用自动化业务并保留业务错误分类。
func (s *AiAutomationService) GetAiAutomationRun(ctx context.Context, req *basev1.GetAiAutomationRunRequest) (*basev1.AiAutomationRun, error) {
	response, err := s.aiAutomationCase.GetAiAutomationRun(ctx, req)
	if err != nil {
		log.Error("GetAiAutomationRun", "error", err)
		return nil, errorsx.WrapInternal(err, "AI自动化操作失败")
	}
	return response, nil
}

// StopAiAutomationRun 调用自动化业务并保留业务错误分类。
func (s *AiAutomationService) StopAiAutomationRun(ctx context.Context, req *basev1.StopAiAutomationRunRequest) (*basev1.AiAutomationRun, error) {
	response, err := s.aiAutomationCase.StopAiAutomationRun(ctx, req)
	if err != nil {
		log.Error("StopAiAutomationRun", "error", err)
		return nil, errorsx.WrapInternal(err, "AI自动化操作失败")
	}
	return response, nil
}

// ContinueAiAutomationRun 调用自动化业务并保留业务错误分类。
func (s *AiAutomationService) ContinueAiAutomationRun(ctx context.Context, req *basev1.ContinueAiAutomationRunRequest) (*basev1.CreateAiSessionResponse, error) {
	response, err := s.aiAutomationCase.ContinueAiAutomationRun(ctx, req)
	if err != nil {
		log.Error("ContinueAiAutomationRun", "error", err)
		return nil, errorsx.WrapInternal(err, "AI自动化操作失败")
	}
	return response, nil
}
