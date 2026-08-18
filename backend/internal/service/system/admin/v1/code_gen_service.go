package admin

import (
	"context"

	systemadminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	biz "github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin"
	"github.com/liujitcn/kratos-core/errorsx"

	"github.com/go-kratos/kratos/v3/log"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

const _ = grpc.SupportPackageIsVersion7

// CodeGenService Admin代码生成执行服务。
type CodeGenService struct {
	systemadminv1.UnimplementedCodeGenServiceServer
	codeGenCase *biz.CodeGenCase
}

// NewCodeGenService 创建Admin代码生成执行服务。
func NewCodeGenService(codeGenCase *biz.CodeGenCase) *CodeGenService {
	return &CodeGenService{codeGenCase: codeGenCase}
}

// GetCodeGenTask 查询代码生成任务进度。
func (s *CodeGenService) GetCodeGenTask(ctx context.Context, req *systemadminv1.GetCodeGenTaskRequest) (*systemadminv1.CodeGenTask, error) {
	res, err := s.codeGenCase.GetCodeGenTask(ctx, req.GetTaskId())
	if err != nil {
		log.Error("GetCodeGenTask", "error", err)
		return nil, errorsx.WrapInternal(err, "查询代码生成任务进度失败")
	}
	return res, nil
}

// PreviewCodeGen 预览代码生成文件。
func (s *CodeGenService) PreviewCodeGen(ctx context.Context, req *systemadminv1.PreviewCodeGenRequest) (*systemadminv1.PreviewCodeGenResponse, error) {
	res, err := s.codeGenCase.PreviewCodeGen(ctx, req.GetTableId(), req.GetOutputPaths())
	if err != nil {
		log.Error("PreviewCodeGen", "error", err)
		return nil, errorsx.WrapInternal(err, "预览代码生成文件失败")
	}
	return res, nil
}

// StartCodeGenTask 启动代码生成任务。
func (s *CodeGenService) StartCodeGenTask(ctx context.Context, req *systemadminv1.StartCodeGenTaskRequest) (*systemadminv1.StartCodeGenTaskResponse, error) {
	res, err := s.codeGenCase.StartCodeGenTask(ctx, req)
	if err != nil {
		log.Error("StartCodeGenTask", "error", err)
		return nil, errorsx.WrapInternal(err, "启动代码生成任务失败")
	}
	return res, nil
}

// RestoreCodeGen 还原代码生成结果。
func (s *CodeGenService) RestoreCodeGen(ctx context.Context, req *systemadminv1.RestoreCodeGenRequest) (*emptypb.Empty, error) {
	err := s.codeGenCase.RestoreCodeGen(ctx, req.GetTableIds())
	if err != nil {
		log.Error("RestoreCodeGen", "error", err)
		return nil, errorsx.WrapInternal(err, "还原代码生成结果失败")
	}
	return &emptypb.Empty{}, nil
}
