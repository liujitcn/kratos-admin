package base

import (
	"context"

	basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"
	biz "github.com/liujitcn/kratos-admin/backend/internal/biz/base"
	"github.com/liujitcn/kratos-core/errorsx"

	"github.com/go-kratos/kratos/v3/log"
)

// AiReportService 提供 AI 助手报告生成接口。
type AiReportService struct {
	basev1.UnimplementedAiReportServiceServer
	aiReportCase *biz.AiReportCase
}

// NewAiReportService 创建 AI 助手报告服务。
func NewAiReportService(aiReportCase *biz.AiReportCase) *AiReportService {
	return &AiReportService{aiReportCase: aiReportCase}
}

// GenerateAiReport 生成可下载的 HTML 和 PDF 报告。
func (s *AiReportService) GenerateAiReport(ctx context.Context, req *basev1.GenerateAiReportRequest) (*basev1.GenerateAiReportResponse, error) {
	res, err := s.aiReportCase.GenerateAiReport(ctx, req)
	if err != nil {
		log.Error("GenerateAiReport", "error", err)
		return nil, errorsx.WrapInternal(err, "生成AI报告失败")
	}
	return res, nil
}
