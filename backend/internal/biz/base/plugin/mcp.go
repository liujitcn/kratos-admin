package plugin

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"github.com/eino-contrib/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// IsAuthenticationRejected 判断 MCP SDK 是否因远端拒绝认证而失败。
func IsAuthenticationRejected(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "unauthorized") || strings.Contains(message, "forbidden")
}

// Discover 使用官方 SDK 完整分页发现；任意一页失败均不返回部分成功结果。
func Discover(ctx context.Context, endpoint, token string, config Config) ([]*mcp.Tool, error) {
	ctx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	_, err := config.ValidateURL(endpoint)
	if err != nil {
		return nil, err
	}
	client := mcp.NewClient(&mcp.Implementation{Name: "kratos-admin", Version: "1"}, nil)
	var session *mcp.ClientSession
	session, err = client.Connect(ctx, Transport(endpoint, token, config), nil)
	if err != nil {
		return nil, err
	}
	// 关闭仅尽力释放传输会话，失败不影响已完成的发现结果。
	defer session.Close()
	return discoverTools(ctx, session)
}

// discoverTools 只提交完整分页结果，并拒绝循环游标、重复工具及超限目录。
func discoverTools(ctx context.Context, session *mcp.ClientSession) ([]*mcp.Tool, error) {
	result := make([]*mcp.Tool, 0)
	cursors := make(map[string]bool)
	names := make(map[string]bool)
	cursor := ""
	for range 256 {
		page, err := session.ListTools(ctx, &mcp.ListToolsParams{Cursor: cursor})
		if err != nil {
			return nil, err
		}
		for _, remote := range page.Tools {
			if len(result) >= 1000 {
				return nil, fmt.Errorf("远端工具超过1000项限制")
			}
			if names[remote.Name] {
				return nil, fmt.Errorf("远端返回重复工具名称")
			}
			names[remote.Name] = true
			result = append(result, remote)
		}
		if page.NextCursor == "" {
			return result, nil
		}
		if cursors[page.NextCursor] {
			return nil, fmt.Errorf("远端返回循环分页游标")
		}
		cursors[page.NextCursor] = true
		cursor = page.NextCursor
	}
	return nil, fmt.Errorf("远端工具发现超过256页限制")
}

// RemoteTool 将确定连接和原始名称转换为增强型 Eino 工具，不接受模型覆盖连接。
type RemoteTool struct {
	Name        string
	Description string
	InputSchema string
	Invoke      func(context.Context, string) (*mcp.CallToolResult, error)
}

// Info 返回服务端绑定名称及已审查的输入契约。
func (t *RemoteTool) Info(context.Context) (*schema.ToolInfo, error) {
	var input jsonschema.Schema
	err := json.Unmarshal([]byte(t.InputSchema), &input)
	if err != nil {
		return nil, err
	}
	return &schema.ToolInfo{Name: t.Name, Desc: t.Description, ParamsOneOf: schema.NewParamsOneOfByJSONSchema(&input)}, nil
}

// InvokableRun 保留结构化结果和所有内容类型，远程资源链接不会被服务器自动读取。
func (t *RemoteTool) InvokableRun(ctx context.Context, arguments *schema.ToolArgument, _ ...tool.Option) (*schema.ToolResult, error) {
	result, err := t.Invoke(ctx, arguments.Text)
	if err != nil {
		return nil, err
	}
	if result.IsError {
		return nil, fmt.Errorf("远端工具执行失败，请检查连接或业务参数")
	}
	var raw []byte
	raw, err = json.Marshal(result)
	if err != nil {
		return nil, err
	}
	if len(raw) > 4<<20 {
		return nil, fmt.Errorf("远端工具结果超过4MiB限制")
	}
	parts := make([]schema.ToolOutputPart, 0, len(result.Content)+1)
	if result.StructuredContent != nil {
		raw, err = json.Marshal(result.StructuredContent)
		if err != nil {
			return nil, err
		}
		parts = append(parts, schema.ToolOutputPart{Type: schema.ToolPartTypeText, Text: string(raw)})
	}
	for _, content := range result.Content {
		switch value := content.(type) {
		case *mcp.TextContent:
			parts = append(parts, schema.ToolOutputPart{Type: schema.ToolPartTypeText, Text: value.Text})
		case *mcp.ImageContent:
			if !slices.Contains([]string{"image/png", "image/jpeg", "image/webp", "image/gif"}, value.MIMEType) {
				return nil, fmt.Errorf("远端图片类型不受支持")
			}
			encoded := base64.StdEncoding.EncodeToString(value.Data)
			parts = append(parts, schema.ToolOutputPart{Type: schema.ToolPartTypeImage, Image: &schema.ToolOutputImage{MessagePartCommon: schema.MessagePartCommon{Base64Data: &encoded, MIMEType: value.MIMEType}}})
		default:
			raw, err = json.Marshal(content)
			if err != nil {
				return nil, err
			}
			parts = append(parts, schema.ToolOutputPart{Type: schema.ToolPartTypeText, Text: "外部资源或其他内容（未自动读取）: " + string(raw)})
		}
	}
	return &schema.ToolResult{Parts: parts}, nil
}
