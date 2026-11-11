package biz

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/liujitcn/go-utils/id"
	"github.com/liujitcn/go-utils/mapper"
	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/rag/admin/v1"
	basebiz "github.com/liujitcn/kratos-admin/backend/internal/biz/base"
	ai "github.com/liujitcn/kratos-admin/backend/internal/biz/base/ai"
	"github.com/liujitcn/kratos-core/biz"
	"github.com/liujitcn/kratos-core/errorsx"
	"gorm.io/gorm"

	"github.com/go-kratos/kratos/v3/log"
)

// 知识库文档在对象存储中的根目录。
const knowledgeDocObjectDirectory = "rag"

// AiKnowledgeDocCase 管理AI知识库文档，切片与向量化委托给知识库引擎。
type AiKnowledgeDocCase struct {
	*biz.BaseCase
	engine       *ai.KnowledgeEngine
	aiPluginCase *basebiz.AiPluginCase
	docMapper    *mapper.CopierMapper[adminv1.AiKnowledgeDoc, ai.Doc]
}

// NewAiKnowledgeDocCase 创建AI知识库文档管理业务实例。
func NewAiKnowledgeDocCase(baseCase *biz.BaseCase, engine *ai.KnowledgeEngine, aiPluginCase *basebiz.AiPluginCase) *AiKnowledgeDocCase {
	return &AiKnowledgeDocCase{
		BaseCase:     baseCase,
		engine:       engine,
		aiPluginCase: aiPluginCase,
		docMapper:    mapper.NewCopierMapper[adminv1.AiKnowledgeDoc, ai.Doc](),
	}
}

// PageAiKnowledgeDoc 分页查询知识库文档。
func (c *AiKnowledgeDocCase) PageAiKnowledgeDoc(ctx context.Context, req *adminv1.PageAiKnowledgeDocRequest) (*adminv1.PageAiKnowledgeDocResponse, error) {
	if err := c.ensureKnowledge(ctx, req.GetId()); err != nil {
		return nil, err
	}
	items, total, err := c.engine.ListDocs(ctx, req.GetId(), req.GetPageNum(), req.GetPageSize())
	if err != nil {
		return nil, err
	}
	records := make([]*adminv1.AiKnowledgeDoc, 0, len(items))
	for _, item := range items {
		record := c.docMapper.ToDTO(item)
		// 处理状态按枚举显式转换，避免依赖复制器的隐式数值转换。
		record.Status = adminv1.AiKnowledgeDocStatus(item.Status)
		record.ExtractorConnectionId = item.ExtractorConnectionID
		records = append(records, record)
	}
	return &adminv1.PageAiKnowledgeDocResponse{Docs: records, Total: int32(total)}, nil
}

// UploadAiKnowledgeDocText 上传纯文本文档：原文落地后统一由 Xberg MCP 抽取。
func (c *AiKnowledgeDocCase) UploadAiKnowledgeDocText(ctx context.Context, knowledgeBaseID int64, name, content string, extractorConnectionID int64) error {
	if err := c.ensureKnowledge(ctx, knowledgeBaseID); err != nil {
		return err
	}
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}
	var filePath, fileHash string
	filePath, fileHash, err = c.storeDocFile(name+".txt", []byte(content))
	if err != nil {
		return err
	}
	return c.ingestDocument(ctx, knowledgeBaseID, strings.TrimSpace(name), name+".txt", []byte(content), filePath, fileHash, extractorConnectionID, authInfo.UserId)
}

// UploadAiKnowledgeDocFile 上传文档文件：原始文件落对象存储后统一由 Xberg MCP 抽取。
func (c *AiKnowledgeDocCase) UploadAiKnowledgeDocFile(ctx context.Context, knowledgeBaseID int64, fileName string, content []byte, docName string, extractorConnectionID int64) error {
	if err := c.ensureKnowledge(ctx, knowledgeBaseID); err != nil {
		return err
	}
	if docName == "" {
		docName = strings.TrimSuffix(filepath.Base(fileName), filepath.Ext(fileName))
	}
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}
	var filePath, fileHash string
	filePath, fileHash, err = c.storeDocFile(fileName, content)
	if err != nil {
		return err
	}
	return c.ingestDocument(ctx, knowledgeBaseID, docName, fileName, content, filePath, fileHash, extractorConnectionID, authInfo.UserId)
}

// DeleteAiKnowledgeDoc 删除文档：级联删除切片与对象存储中的原始文件。
func (c *AiKnowledgeDocCase) DeleteAiKnowledgeDoc(ctx context.Context, docID int64) error {
	item, err := c.engine.GetDoc(ctx, docID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errorsx.ResourceNotFound("AI知识库文档不存在")
		}
		return err
	}
	if err = c.engine.DeleteDoc(ctx, docID); err != nil {
		return err
	}
	c.deleteDocFile(item.FilePath)
	return nil
}

// ReprocessAiKnowledgeDoc 重新处理失败的文档：从对象存储读取原始文件，
// 校验 sha256 未被篡改后重新抽取文本，交给引擎重切片、重向量化。
func (c *AiKnowledgeDocCase) ReprocessAiKnowledgeDoc(ctx context.Context, docID int64) error {
	item, err := c.engine.GetDoc(ctx, docID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errorsx.ResourceNotFound("AI知识库文档不存在")
		}
		return err
	}
	if item.Status != ai.DocStatusFailed {
		return errorsx.InvalidArgument("仅处理失败的文档支持重试")
	}
	if item.FilePath == "" {
		return errorsx.InvalidArgument("文档未保存原始文件，无法重试，请删除后重新上传")
	}
	var content []byte
	content, err = c.readDocFile(item.FilePath, item.FileHash)
	if err != nil {
		return err
	}
	if item.ExtractorConnectionID <= 0 {
		return errorsx.InvalidArgument("文档未绑定Xberg MCP连接，无法重试")
	}
	var text string
	text, err = c.aiPluginCase.ExtractDocumentWithXberg(ctx, item.ExtractorConnectionID, path.Base(item.FilePath), content)
	if err != nil {
		return errorsx.WithMessageKey(errorsx.InvalidArgument("重新抽取文档文本失败"), "rag.admin.ai.knowledge.doc.reextract.failed", nil).WithCause(err)
	}
	return c.engine.ReprocessDoc(ctx, docID, text)
}

// ingestDocument 使用 Xberg 抽取文本后进入切片和向量化；抽取失败保留失败文档供原连接重试。
func (c *AiKnowledgeDocCase) ingestDocument(ctx context.Context, knowledgeBaseID int64, docName, fileName string, content []byte, filePath, fileHash string, extractorConnectionID, createdBy int64) error {
	text, err := c.aiPluginCase.ExtractDocumentWithXberg(ctx, extractorConnectionID, fileName, content)
	if err != nil {
		failed := &ai.Doc{
			KnowledgeBaseID:       knowledgeBaseID,
			Name:                  docName,
			FilePath:              filePath,
			FileHash:              fileHash,
			ExtractorConnectionID: extractorConnectionID,
			Status:                ai.DocStatusFailed,
			ErrorMessage:          err.Error(),
			CreatedBy:             createdBy,
		}
		if createErr := c.engine.CreateDoc(ctx, failed); createErr != nil {
			return errors.Join(err, createErr)
		}
		return err
	}
	_, err = c.engine.IngestText(ctx, knowledgeBaseID, docName, text, filePath, fileHash, extractorConnectionID, createdBy)
	return err
}

// storeDocFile 将文档原始文件保存到对象存储，返回对象路径与 sha256。
// 存储路径与通用文件上传保持一致：rag/文件类型/年/月/日/文件。
func (c *AiKnowledgeDocCase) storeDocFile(fileName string, content []byte) (string, string, error) {
	if c.OSS == nil {
		return "", "", errorsx.Internal("对象存储未配置")
	}
	extension := strings.ToLower(filepath.Ext(fileName))
	if extension == "" {
		extension = ".txt"
	}
	directory := path.Join(knowledgeDocObjectDirectory, docObjectCategory(extension), time.Now().Format("2006/01/02"))
	storedName := fmt.Sprintf("%d%s", id.GenSnowflakeID(), extension)
	if _, err := c.OSS.UploadByByte(storedName, directory, content); err != nil {
		return "", "", errorsx.Internal("文档原始文件上传失败").WithCause(err)
	}
	sum := sha256.Sum256(content)
	return path.Join(directory, storedName), hex.EncodeToString(sum[:]), nil
}

// docObjectCategory 按扩展名归类对象存储目录，目录取值与通用文件上传的约定一致。
func docObjectCategory(extension string) string {
	switch strings.TrimPrefix(strings.ToLower(extension), ".") {
	case "bmp", "gif", "jpeg", "jpg", "png", "webp":
		return "images"
	case "mp3", "wav":
		return "audios"
	case "mp4":
		return "videos"
	default:
		return "docs"
	}
}

// readDocFile 从对象存储读取原始文件并校验 sha256 防篡改。
func (c *AiKnowledgeDocCase) readDocFile(filePath, fileHash string) ([]byte, error) {
	if c.OSS == nil {
		return nil, errorsx.Internal("对象存储未配置")
	}
	content, err := c.OSS.GetFileByte(filePath)
	if err != nil {
		return nil, errorsx.Internal("读取文档原始文件失败，请确认对象存储可用").WithCause(err)
	}
	sum := sha256.Sum256(content)
	if hex.EncodeToString(sum[:]) != fileHash {
		return nil, errorsx.InvalidArgument("文档原始文件校验失败（内容与上传时不一致），请删除后重新上传")
	}
	return content, nil
}

// deleteDocFile 尽力删除对象存储中的原始文件，失败仅记录不阻断。
func (c *AiKnowledgeDocCase) deleteDocFile(filePath string) {
	if c.OSS == nil || filePath == "" {
		return
	}
	if err := c.OSS.DeleteFile(filePath); err != nil {
		log.Warn("删除知识库文档原始文件失败", "file_path", filePath, "error", err)
	}
}

// ensureKnowledge 校验知识库存在。
func (c *AiKnowledgeDocCase) ensureKnowledge(ctx context.Context, id int64) error {
	if _, err := c.engine.GetKnowledge(ctx, id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errorsx.ResourceNotFound("AI知识库不存在")
		}
		return err
	}
	return nil
}
