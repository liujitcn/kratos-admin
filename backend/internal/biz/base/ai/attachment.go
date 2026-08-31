package ai

import (
	"bytes"
	"encoding/json"
	"mime"
	"strings"

	basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"
)

// attachmentPayload 表示附件 JSON 落库结构。
type attachmentPayload struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Size     int64  `json:"size"`
	URL      string `json:"url"`
	MIMEType string `json:"mime_type"`
}

// NormalizeAttachments 清理前端传入的附件列表。
func NormalizeAttachments(values []*basev1.AiAttachment) []*basev1.AiAttachment {
	result := make([]*basev1.AiAttachment, 0, len(values))
	for _, item := range values {
		// 附件来自前端数组，跳过 nil 可避免异常请求污染后续附件读取流程。
		if item == nil {
			continue
		}
		name := item.GetName()
		// 前端或历史数据缺少文件名时，保留一份可读名称供模型提示词和页面展示使用。
		if name == "" {
			name = "未命名附件"
		}
		result = append(result, &basev1.AiAttachment{
			Id:       item.GetId(),
			Name:     name,
			Size:     item.GetSize(),
			Url:      item.GetUrl(),
			MimeType: item.GetMimeType(),
		})
	}
	return result
}

// DetectAttachmentMIME 推断附件 MIME 类型。
func DetectAttachmentMIME(fileName string, rawMIMEType string) string {
	// 上传接口已提供 MIME 时优先使用原值，避免后缀推断覆盖浏览器识别结果。
	if rawMIMEType != "" {
		return rawMIMEType
	}
	// markdown 和日志文件在部分系统中无法通过标准库识别，这里明确按文本处理。
	switch strings.ToLower(pathExt(fileName)) {
	case ".md", ".markdown", ".log":
		return "text/plain; charset=utf-8"
	default:
		return mime.TypeByExtension(strings.ToLower(pathExt(fileName)))
	}
}

// ExtractAttachmentText 从文本类附件字节中提取可拼入模型上下文的内容。
func ExtractAttachmentText(fileBytes []byte, mimeType string) string {
	// 空文件没有可提供给模型的上下文，直接忽略。
	if len(fileBytes) == 0 {
		return ""
	}
	cleanMIMEType := strings.ToLower(mimeType)
	// 当前只把文本类内容拼入提示词，二进制文档需要独立解析能力后再放开。
	if !isTextAttachmentMIME(cleanMIMEType) {
		return ""
	}
	text := string(bytes.TrimSpace(fileBytes))
	runes := []rune(text)
	// 限制单个附件文本长度，避免长文件挤占主问题和历史上下文窗口。
	if len(runes) > maxAttachmentTextLength {
		return string(runes[:maxAttachmentTextLength])
	}
	return text
}

// MarshalAttachments 序列化附件 JSON，供消息表持久化。
func MarshalAttachments(attachments []*basev1.AiAttachment) string {
	payload := make([]attachmentPayload, 0, len(attachments))
	for _, item := range attachments {
		payload = append(payload, attachmentPayload{
			ID:       item.GetId(),
			Name:     item.GetName(),
			Size:     item.GetSize(),
			URL:      item.GetUrl(),
			MIMEType: item.GetMimeType(),
		})
	}
	raw, err := json.Marshal(payload)
	// 附件元信息序列化失败不应阻塞主消息落库，使用空数组保持字段可解析。
	if err != nil {
		return "[]"
	}
	return string(raw)
}

// ParseAttachments 反序列化消息表中的附件 JSON。
func ParseAttachments(raw string) []*basev1.AiAttachment {
	// 历史消息可能没有附件字段，统一返回空数组方便前端渲染。
	if raw == "" {
		return []*basev1.AiAttachment{}
	}
	values := make([]attachmentPayload, 0)
	err := json.Unmarshal([]byte(raw), &values)
	// 附件 JSON 损坏时不影响消息正文展示，前端只是不展示附件卡片。
	if err != nil {
		return []*basev1.AiAttachment{}
	}
	result := make([]*basev1.AiAttachment, 0, len(values))
	for _, item := range values {
		result = append(result, &basev1.AiAttachment{
			Id:       item.ID,
			Name:     item.Name,
			Size:     item.Size,
			Url:      item.URL,
			MimeType: item.MIMEType,
		})
	}
	return result
}

// pathExt 返回文件扩展名，文件名无扩展名时返回空字符串。
func pathExt(name string) string {
	_, ext, found := strings.CutLast(name, ".")
	// 无扩展名或点号在末尾时，不能作为可靠后缀参与 MIME 推断。
	if !found || ext == "" {
		return ""
	}
	return "." + ext
}

// isTextAttachmentMIME 判断附件 MIME 是否可以按文本方式读取。
func isTextAttachmentMIME(mimeType string) bool {
	return strings.HasPrefix(mimeType, "text/") ||
		strings.Contains(mimeType, "json") ||
		strings.Contains(mimeType, "xml") ||
		strings.Contains(mimeType, "csv")
}
