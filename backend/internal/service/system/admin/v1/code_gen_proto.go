package admin

import (
	"context"
	"fmt"

	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	biz "github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin"
	"github.com/liujitcn/kratos-core/errorsx"

	"github.com/go-kratos/kratos/v3/log"
	"google.golang.org/protobuf/types/known/emptypb"
)

// CodeGenProtoService Admin代码生成Proto接口配置服务。
type CodeGenProtoService struct {
	adminv1.UnimplementedCodeGenProtoServiceServer
	codeGenProtoCase *biz.CodeGenProtoCase
}

// NewCodeGenProtoService 创建Admin代码生成Proto接口配置服务。
func NewCodeGenProtoService(codeGenProtoCase *biz.CodeGenProtoCase) *CodeGenProtoService {
	return &CodeGenProtoService{codeGenProtoCase: codeGenProtoCase}
}

// ListCodeGenProto 查询代码生成Proto接口配置。
func (s *CodeGenProtoService) ListCodeGenProto(ctx context.Context, req *adminv1.ListCodeGenProtoRequest) (*adminv1.ListCodeGenProtoResponse, error) {
	res, err := s.codeGenProtoCase.ListCodeGenProto(ctx, req.GetTableId())
	if err != nil {
		log.Error(fmt.Sprintf("ListCodeGenProto %v", err))
		return nil, errorsx.WrapInternal(err, "查询代码生成Proto接口配置失败")
	}
	return res, nil
}

// SaveCodeGenProto 保存代码生成Proto接口配置。
func (s *CodeGenProtoService) SaveCodeGenProto(ctx context.Context, req *adminv1.SaveCodeGenProtoRequest) (*emptypb.Empty, error) {
	err := s.codeGenProtoCase.SaveCodeGenProto(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("SaveCodeGenProto %v", err))
		return nil, errorsx.WrapInternal(err, "保存代码生成Proto接口配置失败")
	}
	return new(emptypb.Empty), nil
}
