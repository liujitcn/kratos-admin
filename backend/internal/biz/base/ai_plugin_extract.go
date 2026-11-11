package biz

import (
	"context"
	"encoding/json"
	"fmt"
	"mime"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/go-kratos/kratos/v3/log"
	"github.com/liujitcn/gorm-kit/repository"
	basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/base/dto"
	"github.com/liujitcn/kratos-core/errorsx"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ExtractDocumentWithXberg 使用当前身份已连接并审核为只读的 Xberg 账号抽取文档文本。
func (c *AiPluginCase) ExtractDocumentWithXberg(ctx context.Context, connectionID int64, fileName string, content []byte) (string, error) {
	if connectionID <= 0 {
		return "", errorsx.InvalidArgument("Xberg连接编号不合法")
	}
	if fileName == "" || len(content) == 0 {
		return "", errorsx.InvalidArgument("文档文件名和内容不能为空")
	}
	query := c.Query(ctx).AiPlugin
	pluginRow, err := c.Find(ctx,
		repository.Where(query.Code.Eq("xberg")),
		repository.Where(query.Source.Eq("MCP")),
		repository.Where(query.Visibility.Eq("system")),
		repository.Where(query.Enabled.Eq(1)),
	)
	if isRecordNotFound(err) {
		return "", errorsx.ResourceNotFound("Xberg MCP插件未初始化")
	}
	if err != nil {
		return "", err
	}
	var refs []dto.AiPluginResolvedTool
	refs, err = c.resolveToolRefs(ctx, "admin", []*basev1.AiPluginSelection{{
		PluginId:     strconv.FormatInt(pluginRow.ID, 10),
		ConnectionId: strconv.FormatInt(connectionID, 10),
	}})
	if err != nil {
		return "", err
	}
	var ref *dto.AiPluginResolvedTool
	for index := range refs {
		candidate := &refs[index]
		if candidate.Snapshot.SourceName != "extract" {
			continue
		}
		if candidate.Policy == nil || candidate.Policy.Effect != "read" {
			return "", errorsx.PermissionDenied("Xberg extract 工具必须审核为只读")
		}
		if ref != nil {
			return "", errorsx.Conflict("Xberg extract 工具存在重复快照")
		}
		ref = candidate
	}
	if ref == nil {
		return "", errorsx.PermissionDenied("当前Xberg连接未审核 extract 工具")
	}
	var arguments string
	arguments, err = marshalXbergExtractArguments(fileName, content)
	if err != nil {
		return "", err
	}
	arguments, _, err = c.prepareResolvedTool(ctx, "admin", *ref, arguments)
	if err != nil {
		return "", err
	}
	var result *mcp.CallToolResult
	result, err = c.invokeRemoteTool(ctx, "admin", *ref, arguments)
	if err != nil {
		return "", err
	}
	var text string
	text, err = parseXbergExtractResult(result)
	if err != nil {
		log.Warn("Xberg文档抽取结果无效", "connection_id", connectionID, "file_name", fileName, "error", err)
		return "", errorsx.InvalidArgument("Xberg文档抽取结果无效").WithCause(err)
	}
	return text, nil
}

// marshalXbergExtractArguments 构造 Xberg extract 的 bytes 输入，避免把字节数组编码成 base64 字符串。
func marshalXbergExtractArguments(fileName string, content []byte) (string, error) {
	bytesValue := make([]int, len(content))
	for index, value := range content {
		bytesValue[index] = int(value)
	}
	mimeType := mime.TypeByExtension(strings.ToLower(filepath.Ext(fileName)))
	if mimeType == "" {
		mimeType = http.DetectContentType(content)
	}
	arguments, err := json.Marshal(map[string]any{
		"input": map[string]any{
			"kind":      "bytes",
			"bytes":     bytesValue,
			"mime_type": mimeType,
			"filename":  filepath.Base(fileName),
		},
	})
	if err != nil {
		return "", fmt.Errorf("构造Xberg抽取参数失败: %w", err)
	}
	return string(arguments), nil
}

// parseXbergExtractResult 只接受 Xberg extract 约定的 results[].content 文本。
func parseXbergExtractResult(result *mcp.CallToolResult) (string, error) {
	if result == nil || result.IsError {
		return "", fmt.Errorf("远端返回错误结果")
	}
	var candidates [][]byte
	if result.StructuredContent != nil {
		raw, err := json.Marshal(result.StructuredContent)
		if err != nil {
			return "", fmt.Errorf("编码结构化抽取结果失败: %w", err)
		}
		candidates = append(candidates, raw)
	}
	for _, item := range result.Content {
		textContent, ok := item.(*mcp.TextContent)
		if ok && strings.TrimSpace(textContent.Text) != "" {
			candidates = append(candidates, []byte(textContent.Text))
		}
	}
	var err error
	for _, raw := range candidates {
		var output struct {
			Results []struct {
				Content string `json:"content"`
			} `json:"results"`
			Errors  []json.RawMessage `json:"errors"`
			Summary json.RawMessage   `json:"summary"`
		}
		if err = json.Unmarshal(raw, &output); err != nil {
			continue
		}
		if len(output.Results) == 0 || len(output.Summary) == 0 || len(output.Errors) > 0 {
			continue
		}
		contents := make([]string, 0, len(output.Results))
		valid := true
		for _, item := range output.Results {
			text := strings.TrimSpace(item.Content)
			if text == "" {
				valid = false
				break
			}
			contents = append(contents, text)
		}
		if valid {
			return strings.Join(contents, "\n\n"), nil
		}
	}
	return "", fmt.Errorf("未发现有效的 results[].content 文本")
}
