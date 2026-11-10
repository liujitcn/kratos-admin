package report

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// TestHTMLPreservesTablesAndBlocksActiveContent 检查报告保留数据表格并阻止主动内容。
func TestHTMLPreservesTablesAndBlocksActiveContent(t *testing.T) {
	content := "## 营收分析\n\n| 月份 | 营收 |\n| --- | ---: |\n| 一月 | 120 |\n\n<script>alert(1)</script>\n\n[链接](javascript:alert(1))\n\n![图片](http://127.0.0.1/private)"
	result, err := HTML("<script>标题</script>", content)
	if err != nil {
		t.Fatal(err)
	}
	text := string(result)
	for _, expected := range []string{"&lt;script&gt;标题&lt;/script&gt;", "<table>", "<td>一月</td>", "120", "default-src 'none'"} {
		if !strings.Contains(text, expected) {
			t.Fatalf("报告缺少内容 %q", expected)
		}
	}
	for _, forbidden := range []string{"<script>", "href=\"javascript:", "script-src 'unsafe-inline'"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("报告包含主动内容 %q", forbidden)
		}
	}
}

// TestPDFHonorsCancellation 检查取消请求无需启动浏览器即可返回。
func TestPDFHonorsCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err := PDF(ctx, []byte("<p>取消</p>"))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("预期取消错误，实际为 %v", err)
	}
}
