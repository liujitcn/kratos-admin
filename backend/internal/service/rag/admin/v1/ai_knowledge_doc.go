package admin

import (
	"context"
	"fmt"

	"github.com/go-kratos/kratos/v3/log"
	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/rag/admin/v1"
	biz "github.com/liujitcn/kratos-admin/backend/internal/biz/rag/admin"
	"github.com/liujitcn/kratos-core/errorsx"
	"google.golang.org/protobuf/types/known/emptypb"
)

// AiKnowledgeDocService AI知识库文档管理服务。
type AiKnowledgeDocService struct {
	adminv1.UnimplementedAiKnowledgeDocServiceServer
	aiKnowledgeDocCase *biz.AiKnowledgeDocCase
}

// NewAiKnowledgeDocService 创建AI知识库文档管理服务。
func NewAiKnowledgeDocService(aiKnowledgeDocCase *biz.AiKnowledgeDocCase) *AiKnowledgeDocService {
	return &AiKnowledgeDocService{aiKnowledgeDocCase: aiKnowledgeDocCase}
}

// PageAiKnowledgeDoc 分页查询知识库文档。
func (s *AiKnowledgeDocService) PageAiKnowledgeDoc(ctx context.Context, req *adminv1.PageAiKnowledgeDocRequest) (*adminv1.PageAiKnowledgeDocResponse, error) {
	result, err := s.aiKnowledgeDocCase.PageAiKnowledgeDoc(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("PageAiKnowledgeDoc %v", err))
		return nil, errorsx.WrapInternal(err, "查询AI知识库文档失败")
	}
	return result, nil
}

// UploadAiKnowledgeDocText 上传纯文本文档。
func (s *AiKnowledgeDocService) UploadAiKnowledgeDocText(ctx context.Context, req *adminv1.UploadAiKnowledgeDocTextRequest) (*emptypb.Empty, error) {
	err := s.aiKnowledgeDocCase.UploadAiKnowledgeDocText(ctx, req.GetId(), req.GetName(), req.GetContent())
	if err != nil {
		log.Error(fmt.Sprintf("UploadAiKnowledgeDocText %v", err))
		return nil, errorsx.WrapInternal(err, "上传AI知识库文档失败")
	}
	return new(emptypb.Empty), nil
}

// UploadAiKnowledgeDocFile 上传文档文件。
func (s *AiKnowledgeDocService) UploadAiKnowledgeDocFile(ctx context.Context, req *adminv1.UploadAiKnowledgeDocFileRequest) (*emptypb.Empty, error) {
	err := s.aiKnowledgeDocCase.UploadAiKnowledgeDocFile(ctx, req.GetId(), req.GetFileName(), req.GetContentBase64(), req.GetDocName())
	if err != nil {
		log.Error(fmt.Sprintf("UploadAiKnowledgeDocFile %v", err))
		return nil, errorsx.WrapInternal(err, "上传AI知识库文档失败")
	}
	return new(emptypb.Empty), nil
}

// DeleteAiKnowledgeDoc 删除知识库文档。
func (s *AiKnowledgeDocService) DeleteAiKnowledgeDoc(ctx context.Context, req *adminv1.DeleteAiKnowledgeDocRequest) (*emptypb.Empty, error) {
	err := s.aiKnowledgeDocCase.DeleteAiKnowledgeDoc(ctx, req.GetDocId())
	if err != nil {
		log.Error(fmt.Sprintf("DeleteAiKnowledgeDoc %v", err))
		return nil, errorsx.WrapInternal(err, "删除AI知识库文档失败")
	}
	return new(emptypb.Empty), nil
}

// ReprocessAiKnowledgeDoc 重新处理失败的文档。
func (s *AiKnowledgeDocService) ReprocessAiKnowledgeDoc(ctx context.Context, req *adminv1.ReprocessAiKnowledgeDocRequest) (*emptypb.Empty, error) {
	err := s.aiKnowledgeDocCase.ReprocessAiKnowledgeDoc(ctx, req.GetDocId())
	if err != nil {
		log.Error(fmt.Sprintf("ReprocessAiKnowledgeDoc %v", err))
		return nil, errorsx.WrapInternal(err, "重新处理AI知识库文档失败")
	}
	return new(emptypb.Empty), nil
}
