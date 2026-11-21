package model

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// responsesPDFTransport 修正当前 SDK 对内联 PDF 的请求转换，复用原出站安全策略。
// shortcut: agenticopenai 将原始 Base64 写入 file_data；升级到修复该转换的版本后移除此适配。
type responsesPDFTransport struct {
	http.RoundTripper
}

// RoundTrip 仅为 Responses 内联 PDF 补充协议前缀，保留其他请求字段和数值精度。
func (transport responsesPDFTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	if request.Method != http.MethodPost || !strings.HasSuffix(request.URL.Path, "/responses") || request.Body == nil {
		return transport.RoundTripper.RoundTrip(request)
	}
	body, err := io.ReadAll(request.Body)
	closeErr := request.Body.Close()
	if err != nil {
		return nil, fmt.Errorf("读取 Responses 请求失败: %w", err)
	}
	if closeErr != nil {
		return nil, fmt.Errorf("关闭 Responses 请求体失败: %w", closeErr)
	}
	if bytes.Contains(body, []byte(`"file_data"`)) {
		var payload map[string]any
		decoder := json.NewDecoder(bytes.NewReader(body))
		decoder.UseNumber()
		err = decoder.Decode(&payload)
		if err != nil {
			return nil, fmt.Errorf("解析 Responses 请求失败: %w", err)
		}
		changed := false
		inputs, _ := payload["input"].([]any)
		for _, input := range inputs {
			item, _ := input.(map[string]any)
			contents, _ := item["content"].([]any)
			for _, content := range contents {
				part, _ := content.(map[string]any)
				data, _ := part["file_data"].(string)
				// JVBERi0 是 PDF 文件头 %PDF- 的 Base64 前缀，避免依赖可能缺失或错误的扩展名。
				if part["type"] == "input_file" && strings.HasPrefix(data, "JVBERi0") {
					part["file_data"] = "data:application/pdf;base64," + data
					changed = true
				}
			}
		}
		if changed {
			body, err = json.Marshal(payload)
			if err != nil {
				return nil, fmt.Errorf("编码 Responses PDF 请求失败: %w", err)
			}
		}
	}
	cloned := request.Clone(request.Context())
	cloned.Body = io.NopCloser(bytes.NewReader(body))
	cloned.ContentLength = int64(len(body))
	cloned.GetBody = func() (io.ReadCloser, error) { return io.NopCloser(bytes.NewReader(body)), nil }
	return transport.RoundTripper.RoundTrip(cloned)
}
