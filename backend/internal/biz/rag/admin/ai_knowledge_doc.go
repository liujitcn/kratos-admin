package biz

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"html"
	"io"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/ledongthuc/pdf"
	"github.com/liujitcn/go-utils/id"
	"github.com/liujitcn/go-utils/mapper"
	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/rag/admin/v1"
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
	engine    *ai.KnowledgeEngine
	docMapper *mapper.CopierMapper[adminv1.AiKnowledgeDoc, ai.Doc]
}

// NewAiKnowledgeDocCase 创建AI知识库文档管理业务实例。
func NewAiKnowledgeDocCase(baseCase *biz.BaseCase, engine *ai.KnowledgeEngine) *AiKnowledgeDocCase {
	return &AiKnowledgeDocCase{
		BaseCase:  baseCase,
		engine:    engine,
		docMapper: mapper.NewCopierMapper[adminv1.AiKnowledgeDoc, ai.Doc](),
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
		records = append(records, record)
	}
	return &adminv1.PageAiKnowledgeDocResponse{Docs: records, Total: int32(total)}, nil
}

// UploadAiKnowledgeDocText 上传纯文本文档：原文落地为 .txt 对象后走统一入库链。
func (c *AiKnowledgeDocCase) UploadAiKnowledgeDocText(ctx context.Context, knowledgeBaseID int64, name, content string) error {
	if err := c.ensureKnowledge(ctx, knowledgeBaseID); err != nil {
		return err
	}
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}
	filePath, fileHash, err := c.storeDocFile(name+".txt", []byte(content))
	if err != nil {
		return err
	}
	_, err = c.engine.IngestText(ctx, knowledgeBaseID, strings.TrimSpace(name), content, filePath, fileHash, authInfo.UserId)
	return err
}

// UploadAiKnowledgeDocFile 上传文档文件：原始文件落对象存储，抽取文本后走同一入库链。
func (c *AiKnowledgeDocCase) UploadAiKnowledgeDocFile(ctx context.Context, knowledgeBaseID int64, fileName string, content []byte, docName string) error {
	if err := c.ensureKnowledge(ctx, knowledgeBaseID); err != nil {
		return err
	}
	text, err := extractDocText(fileName, content)
	if err != nil {
		return errorsx.InvalidArgument(err.Error())
	}
	if docName == "" {
		docName = strings.TrimSuffix(filepath.Base(fileName), filepath.Ext(fileName))
	}
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}
	filePath, fileHash, err := c.storeDocFile(fileName, content)
	if err != nil {
		return err
	}
	_, err = c.engine.IngestText(ctx, knowledgeBaseID, docName, text, filePath, fileHash, authInfo.UserId)
	return err
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
	content, err := c.readDocFile(item.FilePath, item.FileHash)
	if err != nil {
		return err
	}
	text, err := extractDocText(path.Base(item.FilePath), content)
	if err != nil {
		return errorsx.WithMessageKey(errorsx.InvalidArgument("重新抽取文档文本失败"), "rag.admin.ai.knowledge.doc.reextract.failed", nil).WithCause(err)
	}
	return c.engine.ReprocessDoc(ctx, docID, text)
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

// ---- 文档文本抽取 ----
// 从常见文档格式中抽取纯文本，供知识库 RAG 入库使用。
//
// 支持格式：纯文本族（txt/md/csv/log/json/xml/yaml/html）、docx（zip+XML 标准库解析）、
// pdf（ledongthuc/pdf 纯 Go 抽取，扫描件/图片型 PDF 无文本层会返回空并报错）。
// 其余扩展名直接拒绝——宁可报"不支持"也不要把二进制乱码灌进向量库。

// maxExtractFileSize 上传文件大小上限（10MB）。
const maxExtractFileSize = 10 << 20

// supportedExtensions 返回支持的扩展名清单。
func supportedExtensions() []string {
	return []string{".txt", ".md", ".markdown", ".csv", ".log", ".json", ".xml", ".yml", ".yaml", ".html", ".htm", ".docx", ".pdf"}
}

// extractDocText 按扩展名抽取纯文本。
func extractDocText(fileName string, data []byte) (string, error) {
	if len(data) == 0 {
		return "", fmt.Errorf("文件内容为空")
	}
	if len(data) > maxExtractFileSize {
		return "", fmt.Errorf("文件过大: %d 字节（上限 %d）", len(data), maxExtractFileSize)
	}

	ext := strings.ToLower(filepath.Ext(fileName))
	switch ext {
	case ".txt", ".md", ".markdown", ".csv", ".log", ".json", ".xml", ".yml", ".yaml", ".html", ".htm":
		text := sanitizeText(string(data))
		if strings.TrimSpace(text) == "" {
			return "", fmt.Errorf("未抽取到文本内容")
		}
		return text, nil
	case ".docx":
		text, err := extractDocx(data)
		if err != nil {
			return "", fmt.Errorf("docx 解析失败: %w", err)
		}
		return text, nil
	case ".pdf":
		text, err := extractPdf(data)
		if err != nil {
			return "", fmt.Errorf("pdf 解析失败: %w", err)
		}
		return text, nil
	default:
		return "", fmt.Errorf("不支持的文件类型 %q（支持: %s）", ext, strings.Join(supportedExtensions(), " "))
	}
}

// sanitizeText 清洗纯文本：强制有效 UTF-8、规范化换行。
func sanitizeText(s string) string {
	s = strings.ToValidUTF8(s, string([]byte{0xEF, 0xBF, 0xBD})) // U+FFFD
	s = strings.ReplaceAll(s, "\r\n", "\n")
	return s
}

// extractDocx 解 docx（zip 容器）：word/document.xml 里把段落边界换成换行、剥全部 XML 标签、反转义实体。
func extractDocx(data []byte) (string, error) {
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", fmt.Errorf("打开 zip 失败: %w", err)
	}

	const docXML = "word/document.xml"
	var xmlContent []byte
	for _, file := range reader.File {
		if file.Name != docXML {
			continue
		}
		f, err := file.Open()
		if err != nil {
			return "", fmt.Errorf("打开 %s 失败: %w", docXML, err)
		}
		xmlContent, err = io.ReadAll(f)
		_ = f.Close()
		if err != nil {
			return "", fmt.Errorf("读取 %s 失败: %w", docXML, err)
		}
		break
	}
	if xmlContent == nil {
		return "", fmt.Errorf("%s 不存在", docXML)
	}

	// 段落/换行/制表转成可读边界，再剥标签
	s := string(xmlContent)
	s = strings.ReplaceAll(s, "</w:p>", "\n")
	s = strings.ReplaceAll(s, "<w:br/>", "\n")
	s = strings.ReplaceAll(s, "<w:tab/>", "\t")

	var sb strings.Builder
	for i := 0; i < len(s); {
		if s[i] == '<' {
			if j := strings.IndexByte(s[i:], '>'); j >= 0 {
				i += j + 1
				continue
			}
		}
		sb.WriteByte(s[i])
		i++
	}
	text := sanitizeText(html.UnescapeString(sb.String()))
	text = strings.TrimSpace(text)
	if text == "" {
		return "", fmt.Errorf("docx 中未抽取到文本内容")
	}
	return text, nil
}

// extractPdf 抽取 PDF 文本层（非 OCR：扫描件无文本层会得到空内容）。
func extractPdf(data []byte) (string, error) {
	reader, err := pdf.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", fmt.Errorf("打开 pdf 失败: %w", err)
	}

	var sb strings.Builder
	pages := reader.NumPage()
	for i := 1; i <= pages; i++ {
		page := reader.Page(i)
		if page.V.IsNull() {
			continue
		}
		text, err := page.GetPlainText(nil)
		if err != nil {
			continue // 单页抽取失败不阻断整本
		}
		sb.WriteString(text)
		sb.WriteString("\n")
	}
	text := sanitizeText(strings.TrimSpace(sb.String()))
	if strings.TrimSpace(strings.ReplaceAll(text, "\n", "")) == "" {
		return "", fmt.Errorf("未发现文本层（扫描件/图片型 PDF 不支持）")
	}
	return text, nil
}
